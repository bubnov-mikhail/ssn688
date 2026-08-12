package audio

import (
	"encoding/binary"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	voicePlayDelay    = 250 * time.Millisecond
	maxVoiceQueue     = 2
	maxVoiceQueueAge  = 5 * time.Second
)

// Compartment identifies which watch station speaks.
type Compartment string

const (
	CompSonar   Compartment = "SONAR"
	CompWeps    Compartment = "WEPS"
	CompDive    Compartment = "DIVE"
	CompNav     Compartment = "NAV"
	CompCaptain Compartment = "CAPT"
)

type Manager struct {
	ctx          *audio.Context
	sampleRate   int
	masterVol    float64
	voiceVol     float64
	fxVol        float64
	clips        map[ClipID][]byte
	fxClips      map[FXID][]byte
	mu           sync.Mutex
	voicePlaying []*playerVoice // officer lines only
	fxPlaying    []*playerVoice // pings / launch FX — never block voice queue
	loops        map[FXID]*loopTrack
	queue        []queuedVoice
	pending      *pendingClip
	activeClipID ClipID
	subtitle     string
	subtitleAt   time.Time
}

type pendingClip struct {
	id       ClipID
	pcm      []byte
	volume   float64
	subtitle string
	playAt   time.Time
}

type queuedVoice struct {
	clipID     ClipID
	pcm        []byte
	volume     float64
	subtitle   string
	enqueuedAt time.Time
}

type playerVoice struct {
	data   []byte
	pos    int
	volume float64
}

// loopTrack is one ambient FX loop mixed under voices (propeller, bow wash, …).
// Defined in loop_stretch.go.

// SetLoopingFX starts/updates one ambient FX loop track (others keep playing).
// gain is 0..1 relative to Effects volume; speed is pitch-neutral playback rate (1 = nominal).
// gain<=0 or unknown id stops that track.
func (m *Manager) SetLoopingFX(id FXID, gain, speed float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loops == nil {
		m.loops = make(map[FXID]*loopTrack)
	}
	if id == "" || gain <= 0.001 {
		delete(m.loops, id)
		return
	}
	pcm, ok := m.fxClips[id]
	if !ok || len(pcm) < 4 {
		delete(m.loops, id)
		return
	}
	if gain > 1 {
		gain = 1
	}
	speed = clampLoopSpeed(speed)
	if tr, ok := m.loops[id]; ok && tr != nil {
		tr.gain = gain
		tr.speed = speed
		return
	}
	m.loops[id] = newLoopTrack(pcm, gain, speed)
}

func init() {
	wavDecoder = func(sampleRate int, r io.Reader) (io.Reader, error) {
		stream, err := wav.DecodeWithSampleRate(sampleRate, r)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func NewManager(sampleRate int) *Manager {
	return &Manager{
		ctx:        audio.NewContext(sampleRate),
		sampleRate: sampleRate,
		masterVol:  0.8,
		voiceVol:   0.9,
		fxVol:      0.7,
		clips:      MustLoadVoiceClips(sampleRate),
		fxClips:    MustLoadFXClips(sampleRate),
	}
}

func (m *Manager) SetVolumes(master, voice, fx float64) {
	m.masterVol = master
	m.voiceVol = voice
	m.fxVol = fx
}

// PlayClip plays a pre-recorded officer voice line with subtitle overlay.
func (m *Manager) PlayClip(id ClipID, subtitle string) {
	pcm, ok := m.clips[id]
	if !ok {
		m.setSubtitle(clipCompartment(id), subtitle)
		return
	}
	comp := clipCompartment(id)
	if subtitle == "" {
		subtitle = humanSubtitle(id)
	}
	fullSubtitle := string(comp) + ": " + subtitle

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.coalesceClipLocked(id, fullSubtitle) {
		return
	}

	dup := make([]byte, len(pcm))
	copy(dup, pcm)

	// Pending slot is the newest line. Displace a prior pending carefully:
	// keep critical lines by promoting them into the voice queue; drop routine
	// spam so HELM/WEPS button chatter cannot backlog for minutes.
	if m.pending != nil && m.pending.id != id {
		old := m.pending
		m.pending = nil
		if isCriticalClip(old.id) || isCriticalClip(id) {
			m.startVoiceLocked(old.id, old.pcm, old.volume, old.subtitle)
		}
	}
	m.pending = &pendingClip{
		id:       id,
		pcm:      dup,
		volume:   m.voiceVol * m.masterVol,
		subtitle: fullSubtitle,
		playAt:   time.Now().Add(voicePlayDelay),
	}
	m.subtitle = fullSubtitle
	m.subtitleAt = time.Now()
}

func isCriticalClip(id ClipID) bool {
	switch id {
	case ClipWepsTorpedoInWater, ClipWepsTorpedoHeadingOwnship, ClipWepsImpactConfirmed,
		ClipCaptHoldSimulation,
		ClipCaptOwnshipHit, ClipCaptCriticalDamage, ClipCaptOwnshipLost, ClipCaptCommMessage:
		return true
	default:
		// Tube fire callouts are short and situational — treat as critical.
		if strings.Contains(string(id), "torpedo_away") {
			return true
		}
		return false
	}
}

func (m *Manager) coalesceClipLocked(id ClipID, subtitle string) bool {
	if m.pending != nil && m.pending.id == id {
		m.pending.subtitle = subtitle
		m.subtitle = subtitle
		m.subtitleAt = time.Now()
		return true
	}
	if m.activeClipID == id && len(m.voicePlaying) > 0 {
		m.subtitle = subtitle
		m.subtitleAt = time.Now()
		return true
	}
	for i := range m.queue {
		if m.queue[i].clipID == id {
			m.queue[i].subtitle = subtitle
			m.queue[i].enqueuedAt = time.Now()
			m.subtitle = subtitle
			m.subtitleAt = time.Now()
			return true
		}
	}
	return false
}

func (m *Manager) startVoiceLocked(id ClipID, pcm []byte, volume float64, subtitle string) {
	m.pruneStaleQueueLocked()
	if len(m.voicePlaying) == 0 {
		m.activeClipID = id
		m.voicePlaying = append(m.voicePlaying[:0], &playerVoice{data: pcm, volume: volume})
		return
	}
	// Replace last queued line from the same watch station (depth/heading spam).
	comp := clipCompartment(id)
	for i := len(m.queue) - 1; i >= 0; i-- {
		if clipCompartment(m.queue[i].clipID) == comp && !isCriticalClip(m.queue[i].clipID) {
			m.queue[i] = queuedVoice{clipID: id, pcm: pcm, volume: volume, subtitle: subtitle, enqueuedAt: time.Now()}
			return
		}
	}
	m.queue = append(m.queue, queuedVoice{clipID: id, pcm: pcm, volume: volume, subtitle: subtitle, enqueuedAt: time.Now()})
	for len(m.queue) > maxVoiceQueue {
		// Prefer dropping non-critical from the front.
		dropped := false
		for i := range m.queue {
			if !isCriticalClip(m.queue[i].clipID) {
				m.queue = append(m.queue[:i], m.queue[i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			m.queue = m.queue[1:]
		}
	}
}

func (m *Manager) pruneStaleQueueLocked() {
	if len(m.queue) == 0 {
		return
	}
	now := time.Now()
	alive := m.queue[:0]
	for _, q := range m.queue {
		if now.Sub(q.enqueuedAt) <= maxVoiceQueueAge || isCriticalClip(q.clipID) {
			if now.Sub(q.enqueuedAt) <= maxVoiceQueueAge*2 {
				alive = append(alive, q)
			}
		}
	}
	m.queue = alive
}

func (m *Manager) flushPendingLocked() {
	if m.pending == nil || time.Now().Before(m.pending.playAt) {
		return
	}
	p := m.pending
	m.pending = nil
	m.startVoiceLocked(p.id, p.pcm, p.volume, p.subtitle)
}

func (m *Manager) setSubtitle(comp Compartment, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subtitle = string(comp) + ": " + text
	m.subtitleAt = time.Now()
}

func (m *Manager) playFXLocked(pcm []byte, volume float64) {
	m.fxPlaying = append(m.fxPlaying, &playerVoice{data: pcm, volume: volume})
}

func (m *Manager) PlayPing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playFXLocked(generateToneWAV(m.sampleRate, 880, 0.08, 0.3), m.fxVol*m.masterVol*0.6)
}

func (m *Manager) PlayEnemyPing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pcm, ok := m.clips[ClipSonarEnemyPing]; ok {
		dup := make([]byte, len(pcm))
		copy(dup, pcm)
		m.playFXLocked(dup, m.fxVol*m.masterVol*0.55)
		return
	}
	m.playFXLocked(generateEnemyPingWAV(m.sampleRate), m.fxVol*m.masterVol*0.55)
}

func (m *Manager) PlayTorpedoLaunch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pcm, ok := m.fxClips[FXTorpedoLaunch]; ok && len(pcm) >= 4 {
		dup := make([]byte, len(pcm))
		copy(dup, pcm)
		m.playFXLocked(dup, m.fxVol*m.masterVol*1.0)
		return
	}
	m.playFXLocked(generateSweepWAV(m.sampleRate, 200, 80, 0.5), m.fxVol*m.masterVol)
}

// PlayUnderwaterExplosion is a one-shot blast FX (any screen) after sound arrival.
func (m *Manager) PlayUnderwaterExplosion(gain float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gain <= 0.001 {
		return
	}
	if gain > 1 {
		gain = 1
	}
	pcm, ok := m.fxClips[FXUnderwaterExplosion]
	if !ok || len(pcm) < 4 {
		return
	}
	dup := make([]byte, len(pcm))
	copy(dup, pcm)
	m.playFXLocked(dup, m.fxVol*m.masterVol*gain)
}

// PlayTubeDoorOpen plays the hydraulic outer-door open FX (WEPS).
func (m *Manager) PlayTubeDoorOpen() {
	m.playOneShotFX(FXTubeDoorOpen, 0.85)
}

// PlayTubeDoorClose plays the time-reversed hydraulic outer-door close FX (WEPS).
func (m *Manager) PlayTubeDoorClose() {
	m.playOneShotFX(FXTubeDoorClose, 0.85)
}

// PlayMastHydraulic plays the shared raise/lower FX for ESM, COMM, or periscope.
func (m *Manager) PlayMastHydraulic() {
	m.playOneShotFX(FXMastHydraulic, 0.8)
}

func (m *Manager) playOneShotFX(id FXID, gain float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gain <= 0.001 {
		return
	}
	if gain > 1 {
		gain = 1
	}
	pcm, ok := m.fxClips[id]
	if !ok || len(pcm) < 4 {
		return
	}
	dup := make([]byte, len(pcm))
	copy(dup, pcm)
	m.playFXLocked(dup, m.fxVol*m.masterVol*gain)
}

// PlayESMHit is a short RWR-style chirp when a search radar main beam paints the mast.
func (m *Manager) PlayESMHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playFXLocked(generateESMChirpWAV(m.sampleRate), m.fxVol*m.masterVol*0.5)
}

// StopAll clears queued/playing voices and FX so a session can release cleanly.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.voicePlaying = nil
	m.fxPlaying = nil
	m.loops = nil
	m.queue = nil
	m.pending = nil
	m.activeClipID = ""
	m.subtitle = ""
	m.subtitleAt = time.Time{}
}

func (m *Manager) Subtitle() (string, bool) {
	if time.Since(m.subtitleAt) > 5*time.Second {
		return "", false
	}
	return m.subtitle, true
}

func (m *Manager) Stream() *audio.Player {
	p := &streamPlayer{m: m}
	player, _ := m.ctx.NewPlayer(p)
	return player
}

type streamPlayer struct {
	m *Manager
}

func advancePlayers(list []*playerVoice) []*playerVoice {
	remaining := list[:0]
	for _, v := range list {
		if v.pos+1 >= len(v.data) {
			continue
		}
		remaining = append(remaining, v)
	}
	return remaining
}

func mixPlayers(list []*playerVoice) float64 {
	var sample float64
	for _, v := range list {
		if v.pos+1 >= len(v.data) {
			continue
		}
		val := int16(binary.LittleEndian.Uint16(v.data[v.pos:]))
		sample += float64(val) / 32768 * v.volume
		v.pos += 2
	}
	return sample
}

func mixLooping(tr *loopTrack, volume float64) float64 {
	if tr == nil || volume <= 0 {
		return 0
	}
	return tr.nextSample() * volume
}

func (s *streamPlayer) Read(buf []byte) (int, error) {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()

	s.m.flushPendingLocked()

	// Ebiten expects stereo 16-bit PCM; mono samples are duplicated to L/R.
	for i := 0; i < len(buf); i += 4 {
		sample := mixPlayers(s.m.voicePlaying) + mixPlayers(s.m.fxPlaying)
		for _, tr := range s.m.loops {
			if tr == nil || tr.gain <= 0 || len(tr.pcm) < 4 {
				continue
			}
			vol := tr.gain * s.m.fxVol * s.m.masterVol
			sample += mixLooping(tr, vol)
		}
		s.m.voicePlaying = advancePlayers(s.m.voicePlaying)
		s.m.fxPlaying = advancePlayers(s.m.fxPlaying)

		if len(s.m.voicePlaying) == 0 {
			s.m.activeClipID = ""
			s.m.pruneStaleQueueLocked()
			if len(s.m.queue) > 0 {
				next := s.m.queue[0]
				s.m.queue = s.m.queue[1:]
				s.m.activeClipID = next.clipID
				s.m.voicePlaying = append(s.m.voicePlaying, &playerVoice{data: next.pcm, volume: next.volume})
				s.m.subtitle = next.subtitle
				s.m.subtitleAt = time.Now()
			} else {
				s.m.flushPendingLocked()
			}
		}
		if sample > 1 {
			sample = 1
		}
		if sample < -1 {
			sample = -1
		}
		s16 := uint16(int16(sample * 32767))
		binary.LittleEndian.PutUint16(buf[i:], s16)
		binary.LittleEndian.PutUint16(buf[i+2:], s16)
	}
	return len(buf), nil
}

func (s *streamPlayer) Close() error { return nil }

func humanSubtitle(id ClipID) string {
	switch id {
	case ClipCaptHoldSimulation:
		return "Hold simulation."
	case ClipCaptSaveComplete:
		return "Save complete."
	case ClipCaptOwnshipHit:
		return "Own ship hit. Systems damaged."
	case ClipCaptCriticalDamage:
		return "Critical damage. System casualty."
	case ClipCaptOwnshipLost:
		return "Own ship lost. We are sinking."
	case ClipCaptCommMessage:
		return "Flash traffic. Incoming message."
	case ClipSonarPassiveOn:
		return "Passive sonar online."
	case ClipSonarPassiveOff:
		return "Passive sonar offline."
	case ClipSonarActiveStandby:
		return "Active sonar standby."
	case ClipSonarActiveOnline:
		return "Active sonar online."
	case ClipSonarDeployTowed:
		return "Deploying towed array."
	case ClipSonarTowedHeld:
		return "Towed array held."
	case ClipSonarRetractTowed:
		return "Retracting towed array."
	case ClipSonarBTLaunch:
		return "Launching bathythermograph."
	case ClipSonarLayerSurveyComplete:
		return "Layer survey complete."
	case ClipSonarContactClassified:
		return "Contact classified."
	case ClipWepsImpactConfirmed:
		return "Weapon impact confirmed."
	case ClipWepsTorpedoInWater:
		return "Torpedo in the water."
	case ClipWepsTorpedoHeadingOwnship:
		return "Incomming torpedo!"
	case ClipWepsOuterDoorClosed:
		return "Outer door closed."
	case ClipWepsRunDepthSet:
		return "Run depth set."
	case ClipWepsSpeedHigh:
		return "Torpedo speed HIGH."
	case ClipWepsSpeedLow:
		return "Torpedo speed LOW."
	case ClipWepsSeekerOn:
		return "Seeker on."
	case ClipWepsSeekerOff:
		return "Seeker off."
	case ClipWepsWireCut:
		return "Wire cut."
	case ClipDiveComeLeft:
		return "Come left."
	case ClipDiveComeRight:
		return "Come right."
	case ClipDiveMakeDepth:
		return "Make depth."
	case ClipDiveHoldDepth:
		return "Hold depth."
	case ClipDiveUnableDeeper:
		return "Unable to dive deeper."
	case ClipNavSpeedDouble:
		return "Time compression double."
	case ClipNavSpeedQuad:
		return "Time compression quadruple."
	case ClipNavSpeedNormal:
		return "Time compression normal."
	default:
		return string(id)
	}
}

func generateToneWAV(sampleRate int, freqHz, durationSec, amp float64) []byte {
	n := int(float64(sampleRate) * durationSec)
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		env := 1.0
		if t < 0.01 {
			env = t / 0.01
		} else if rem := durationSec - t; rem < 0.03 {
			env = rem / 0.03
		}
		s := math.Sin(2 * math.Pi * freqHz * t) * amp * env
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(s*32767)))
	}
	return out
}

func generateEnemyPingWAV(sampleRate int) []byte {
	return generateToneWAV(sampleRate, 2200, 0.12, 0.35)
}

// generateESMChirpWAV — brief descending RWR tick (~3.6→1.8 kHz, 60 ms).
func generateESMChirpWAV(sampleRate int) []byte {
	const (
		dur = 0.06
		f0  = 3600.0
		f1  = 1800.0
	)
	n := int(float64(sampleRate) * dur)
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		p := t / dur
		freq := f0 + (f1-f0)*p
		env := 1.0
		if p < 0.08 {
			env = p / 0.08
		} else if p > 0.55 {
			env = (1 - p) / 0.45
		}
		s := math.Sin(2*math.Pi*freq*t) * 0.38 * env
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(s*32767)))
	}
	return out
}

func generateSweepWAV(sampleRate int, f0, f1, durationSec float64) []byte {
	n := int(float64(sampleRate) * durationSec)
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		p := t / durationSec
		freq := f0 + (f1-f0)*p
		env := 1.0 - 0.5*p
		s := math.Sin(2*math.Pi*freq*t) * 0.4 * env
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(s*32767)))
	}
	return out
}
