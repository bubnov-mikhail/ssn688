package ai

import (
	"math"
	"testing"

	"github.com/ssn688/sim/internal/world"
)

func TestRandomCrewSkillClampsAndJitters(t *testing.T) {
	lo := world.RandomCrewSkill(30, 10, 0)
	hi := world.RandomCrewSkill(30, 10, 1)
	if lo < 19.5 || lo > 20.5 {
		t.Fatalf("u=0 → ~20, got %.1f", lo)
	}
	if hi < 39.5 || hi > 40.5 {
		t.Fatalf("u=1 → ~40, got %.1f", hi)
	}
	if world.RandomCrewSkill(5, 20, 0) != 0 {
		t.Fatal("should clamp to 0")
	}
	if world.RandomCrewSkill(95, 20, 1) != 100 {
		t.Fatal("should clamp to 100")
	}
}

func TestTrackClassifiedGateBySkill(t *testing.T) {
	green := &world.Entity{CrewSkill: 0, Track: world.AITrack{Valid: true, ClassConf: 0.50}}
	if TrackClassified(green) {
		t.Fatal("green should need ~0.58 ClassConf")
	}
	green.Track.ClassConf = 0.60
	if !TrackClassified(green) {
		t.Fatal("green at 0.60 should prosecute")
	}
	vet := &world.Entity{CrewSkill: 100, Track: world.AITrack{Valid: true, ClassConf: 0.25}}
	if !TrackClassified(vet) {
		t.Fatal("veteran should prosecute from ~0.22")
	}
}

func TestUpdateCrewTrackVeteranConvergesFaster(t *testing.T) {
	player := &world.Entity{
		ID: "p", Status: world.StatusActive, X: 0, Y: 3000, DepthFt: 200,
		HeadingDeg: 90, SpeedKts: 8,
	}
	green := &world.Entity{ID: "g", Status: world.StatusActive, CrewSkill: 5}
	vet := &world.Entity{ID: "v", Status: world.StatusActive, CrewSkill: 95}
	const dt = 0.1
	for i := 0; i < 400; i++ { // 40s
		tSec := float64(i) * dt
		UpdateCrewTrack(green, player, true, false, 14, tSec, dt)
		UpdateCrewTrack(vet, player, true, false, 14, tSec, dt)
	}
	if !TrackClassified(vet) {
		t.Fatalf("veteran should classify after 40s (conf=%.2f)", vet.Track.ClassConf)
	}
	if TrackClassified(green) {
		t.Fatalf("green should still be unsure after 40s (conf=%.2f)", green.Track.ClassConf)
	}
	vetErr := math.Hypot(vet.Track.X-player.X, vet.Track.Y-player.Y)
	greenErr := math.Hypot(green.Track.X-player.X, green.Track.Y-player.Y)
	if vetErr >= greenErr {
		t.Fatalf("veteran localize err %.0f should beat green %.0f", vetErr, greenErr)
	}
	crsErrV := math.Abs(shortestRel(vet.Track.CourseDeg - player.HeadingDeg))
	crsErrG := math.Abs(shortestRel(green.Track.CourseDeg - player.HeadingDeg))
	if crsErrV >= crsErrG {
		t.Fatalf("veteran TMA course err %.0f should beat green %.0f", crsErrV, crsErrG)
	}
}

func TestWireGuideSkillExtremes(t *testing.T) {
	if WireGuideGain(0) >= WireGuideGain(1) {
		t.Fatal("veterans should steer harder on the wire")
	}
	if WireGuideNoiseDeg(0) <= WireGuideNoiseDeg(1) {
		t.Fatal("green wire steer should be noisier")
	}
	if WireHandoffAgeSec(0) >= WireHandoffAgeSec(1) {
		t.Fatal("veterans should hold wire longer before seeker handoff")
	}
}

func TestSpawnHostileTorpedoGyroSmearBySkill(t *testing.T) {
	// Covered in weapons package via CrewSkill on sub — smoke via TrackAim ghost.
	hunter := &world.Entity{
		ID: "ss", Status: world.StatusActive, CrewSkill: 0,
		Track: world.AITrack{Valid: true, ClassConf: 0.7, X: 100, Y: 2000, DepthFt: 180, CourseDeg: 10, SpeedKts: 6},
	}
	truth := &world.Entity{ID: "p", Status: world.StatusActive, X: 0, Y: 3000, DepthFt: 200}
	aim := TrackAimEntity(hunter, truth)
	if aim == truth || aim.X != 100 || aim.Y != 2000 {
		t.Fatal("classified track should produce ghost aim, not truth")
	}
}
