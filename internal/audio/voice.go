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

const voicePlayDelay = 400 * time.Millisecond

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
	mu           sync.Mutex
	playing      []*playerVoice
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
	clipID   ClipID
	pcm      []byte
	volume   float64
	subtitle string
}

type playerVoice struct {
	data   []byte
	pos    int
	volume float64
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

func (m *Manager) coalesceClipLocked(id ClipID, subtitle string) bool {
	if m.pending != nil && m.pending.id == id {
		m.pending.subtitle = subtitle
		m.subtitle = subtitle
		m.subtitleAt = time.Now()
		return true
	}
	if m.activeClipID == id && len(m.playing) > 0 {
		m.subtitle = subtitle
		m.subtitleAt = time.Now()
		return true
	}
	for i := range m.queue {
		if m.queue[i].clipID == id {
			m.queue[i].subtitle = subtitle
			m.subtitle = subtitle
			m.subtitleAt = time.Now()
			return true
		}
	}
	return false
}

func (m *Manager) startVoiceLocked(id ClipID, pcm []byte, volume float64, subtitle string) {
	m.activeClipID = id
	if len(m.playing) == 0 && len(m.queue) == 0 {
		m.playing = append(m.playing, &playerVoice{data: pcm, volume: volume})
		return
	}
	m.queue = append(m.queue, queuedVoice{clipID: id, pcm: pcm, volume: volume, subtitle: subtitle})
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

func (m *Manager) PlayPing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	wav := generateToneWAV(m.sampleRate, 880, 0.08, 0.3)
	m.playing = append(m.playing, &playerVoice{data: wav, volume: m.fxVol * m.masterVol * 0.6})
}

func (m *Manager) PlayEnemyPing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pcm, ok := m.clips[ClipSonarEnemyPing]; ok {
		dup := make([]byte, len(pcm))
		copy(dup, pcm)
		m.playing = append(m.playing, &playerVoice{data: dup, volume: m.fxVol * m.masterVol * 0.55})
		return
	}
	wav := generateEnemyPingWAV(m.sampleRate)
	m.playing = append(m.playing, &playerVoice{data: wav, volume: m.fxVol * m.masterVol * 0.55})
}

func (m *Manager) PlayTorpedoLaunch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	wav := generateSweepWAV(m.sampleRate, 200, 80, 0.5)
	m.playing = append(m.playing, &playerVoice{data: wav, volume: m.fxVol * m.masterVol})
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

func (s *streamPlayer) Read(buf []byte) (int, error) {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()

	s.m.flushPendingLocked()

	// Ebiten expects stereo 16-bit PCM; mono samples are duplicated to L/R.
	for i := 0; i < len(buf); i += 4 {
		var sample float64
		remaining := s.m.playing[:0]
		for _, v := range s.m.playing {
			if v.pos+1 >= len(v.data) {
				continue
			}
			val := int16(binary.LittleEndian.Uint16(v.data[v.pos:]))
			sample += float64(val) / 32768 * v.volume
			v.pos += 2
			remaining = append(remaining, v)
		}
		s.m.playing = remaining
		if len(s.m.playing) == 0 && len(s.m.queue) > 0 {
			next := s.m.queue[0]
			s.m.queue = s.m.queue[1:]
			s.m.activeClipID = next.clipID
			s.m.playing = append(s.m.playing, &playerVoice{data: next.pcm, volume: next.volume})
			s.m.subtitle = next.subtitle
			s.m.subtitleAt = time.Now()
		}
		if len(s.m.playing) == 0 {
			s.m.activeClipID = ""
			s.m.flushPendingLocked()
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
	case ClipCaptMissionBrief:
		return "Rig ship for silent running. Locate and engage assigned targets."
	case ClipCaptHoldSimulation:
		return "Hold simulation."
	case ClipCaptSaveComplete:
		return "Save complete."
	case ClipSonarPassiveOn:
		return "Passive sonar online."
	case ClipSonarPassiveOff:
		return "Passive sonar offline."
	case ClipSonarActiveStandby:
		return "Active sonar standby."
	case ClipSonarActivePing:
		return "Transmitting active pulse."
	case ClipWepsImpactConfirmed:
		return "Weapon impact confirmed."
	case ClipWepsOuterDoorClosed:
		return "Outer door closed."
	case ClipWepsGyroSet:
		return "Gyro angle set."
	case ClipWepsRunDepthSet:
		return "Run depth set."
	case ClipWepsSpeedHigh:
		return "Torpedo speed HIGH."
	case ClipWepsSpeedLow:
		return "Torpedo speed LOW."
	case ClipWepsSeekerOn:
		return "Seeker enabled."
	case ClipWepsSeekerOff:
		return "Seeker disabled."
	case ClipWepsWireCut:
		return "Wire cut."
	case ClipDiveComeLeft:
		return "Come left, aye."
	case ClipDiveComeRight:
		return "Come right, aye."
	case ClipDiveMakeDepth:
		return "Make depth, aye."
	case ClipNavSpeedHalf:
		return "Time acceleration 0.5x."
	case ClipNavSpeedNormal:
		return "Time acceleration normal."
	case ClipNavSpeedDouble:
		return "Time acceleration 2x."
	case ClipNavSpeedQuad:
		return "Time acceleration 4x."
	case ClipNavSpeedEight:
		return "Time acceleration 8x."
	default:
		if strings.HasPrefix(string(id), "weps/outer_door_open_") {
			return "Outer door open."
		}
		if strings.HasPrefix(string(id), "weps/torpedo_away_") {
			return "Torpedo away."
		}
		return string(id)
	}
}

func generateToneWAV(sampleRate int, freq, duration, volume float64) []byte {
	n := int(float64(sampleRate) * duration)
	pcm := make([]byte, n*2)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-t * 3)
		val := math.Sin(2*math.Pi*freq*t) * volume * env
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(val*32767)))
	}
	return pcm
}

func generateSweepWAV(sampleRate int, startFreq, endFreq, duration float64) []byte {
	n := int(float64(sampleRate) * duration)
	pcm := make([]byte, n*2)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		f := startFreq + (endFreq-startFreq)*t/duration
		val := math.Sin(2*math.Pi*f*t) * 0.4 * (1 - t/duration)
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(val*32767)))
	}
	return pcm
}

func generateEnemyPingWAV(sampleRate int) []byte {
	duration := 0.95
	n := int(float64(sampleRate) * duration)
	pcm := make([]byte, n*2)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		f0 := 820.0 - 40.0*(t/duration)
		core := math.Sin(2 * math.Pi * f0 * t)
		harm := 0.22 * math.Sin(2*math.Pi*(f0*2.02)*t+0.35)
		ring := 0.18 * math.Sin(2*math.Pi*(f0*0.51)*t+1.1)
		echo1, echo2 := 0.0, 0.0
		if et := t - 0.16; et > 0 {
			echo1 = 0.30 * math.Sin(2*math.Pi*690.0*et) * math.Exp(-et*3.8)
		}
		if et := t - 0.33; et > 0 {
			echo2 = 0.18 * math.Sin(2*math.Pi*560.0*et) * math.Exp(-et*3.2)
		}
		attack := math.Min(1, t/0.02)
		env := attack * math.Exp(-t*2.6)
		val := (core + harm + ring + echo1 + echo2) * 0.46 * env
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(val*32767)))
	}
	return pcm
}
