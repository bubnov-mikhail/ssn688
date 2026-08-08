package save

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/sim"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

func TestSaveLoadRoundTripPlatformsAndTorpedoes(t *testing.T) {
	sc := world.NewTrainingScenario()
	eng := sim.NewEngine(sc)

	player := eng.Scenario.Player
	player.X, player.Y = 1234.5, -678.25
	player.DepthFt = 312
	player.HeadingDeg = 0 // true north — regression for LaunchHeadDeg==0 bug
	player.OrderedHead = 7
	player.SpeedKts = 9.5
	player.OrderedSpeed = 12
	player.OrderedDepth = 280
	player.AIState = ""
	player.LengthFt = 360

	var enemySub, enemyShip *world.Entity
	for _, e := range eng.Scenario.Entities {
		switch e.ID {
		case "enemy_foxtrot":
			enemySub = e
		case "enemy_grisha":
			enemyShip = e
		}
	}
	if enemySub == nil || enemyShip == nil {
		t.Fatal("expected training enemies")
	}
	enemySub.X, enemySub.Y = 4500, 2200
	enemySub.DepthFt = 190
	enemySub.HeadingDeg = 215.5
	enemySub.OrderedHead = 200
	enemySub.SpeedKts = 7
	enemySub.OrderedSpeed = 10
	enemySub.AIState = "ATTACK"
	enemySub.ActiveSonar = true
	enemySub.LastPingTime = 88.5
	enemySub.LastPingPower = 0.6
	enemySub.LengthFt = 300

	enemyShip.X, enemyShip.Y = -3000, 5000
	enemyShip.HeadingDeg = 45
	enemyShip.OrderedHead = 50
	enemyShip.SpeedKts = 18
	enemyShip.OrderedSpeed = 22
	enemyShip.Status = world.StatusSinking
	enemyShip.SinkRateFPM = 35
	enemyShip.WreckNoiseUntil = 500
	enemyShip.AIState = "SEARCH"
	enemyShip.LengthFt = 235

	eng.Clock.GameTime = 120.25
	eng.Clock.TimeScale = 2
	eng.Scenario.FailReason = ""
	eng.Acoustics.Env.LayerSurveyKnown = true
	eng.Acoustics.Env.LayerSurveyStartAt = 10
	eng.Acoustics.Env.LayerSurveyEndAt = 0
	eng.Sonar.ListenBand = acoustics.ListenHF
	eng.Sonar.LastPingTime = 100.5
	eng.Sonar.PassiveArray = acoustics.PassiveArrayTowed
	eng.Sonar.TowedCablePct = 0.65
	eng.Periscope.Order = acoustics.PeriMastRaise
	eng.Periscope.Extension = 0.85
	eng.Periscope.TrainRelDeg = -25
	eng.Periscope.Zoom = acoustics.PeriZoomMed
	eng.Periscope.LockEntityID = "enemy_foxtrot"
	eng.Sonar.Contacts = []acoustics.Contact{{
		ID: "C1", BearingDeg: 33, EstimatedRangeYd: 4200, SNR: 12,
		BestMatchID: "foxtrot", BestMatchName: "Foxtrot SS", Confidence: 0.7,
		SourceEntityID: "enemy_foxtrot", DetectedBy: "passive", Kind: world.KindSubmarine,
		ConfirmedID: "foxtrot", ConfirmedClass: "Pr.641",
		UncBearingDeg: 4, UncRangeYd: 800, LastUpdate: 120, FirstSeen: 40, ListenTime: 80,
		LastActiveBearingDeg: 34, LastActiveRangeYd: 4100, LastActiveFixAt: 110,
		TMACourseDeg: 78, TMASpeedKts: 16, TMAAccuracy: 0.82,
	}}

	eng.FireControl.MagazineLeft = 17
	eng.FireControl.SelectedTube = 2
	eng.FireControl.GyroAngleDeg = 90
	eng.FireControl.RunDepthFt = 250
	eng.FireControl.SpeedSetting = "LOW"
	eng.FireControl.SeekerEnabled = true
	eng.FireControl.EnemyMagazine["enemy_foxtrot"] = 9
	eng.FireControl.SetTorpedoSeq(7)
	eng.FireControl.Tubes[0].State = weapons.TubeFired
	eng.FireControl.Tubes[0].TorpedoID = "MK48-7"
	eng.FireControl.Tubes[0].WireIntact = true

	fish := &weapons.Torpedo{
		ID: "MK48-7", ParentSubID: "player", TubeNumber: 1, Side: world.SidePlayer,
		X: 1300, Y: -600, DepthFt: 260, HeadingDeg: 0, OrderedHead: 90,
		SpeedKts: 30, CruiseKts: 28, RunDepthFt: 250,
		SeekerOn: true, Armed: true, Alive: true, Mode: weapons.ModeSearch,
		Age: 40, LastPingTime: 118, LaunchHeadDeg: 0, GyroCourseDeg: 90,
		ClearDistYd: 400, EnableSearchAfterClear: true,
	}
	fish.MarkGyroEnabled(true)
	hostile := &weapons.Torpedo{
		ID: "ETORP-3", ParentSubID: "enemy_foxtrot", TargetID: "player", Side: world.SideEnemy,
		X: 4400, Y: 2100, DepthFt: 180, HeadingDeg: 200, OrderedHead: 210,
		SpeedKts: 40, CruiseKts: 48, RunDepthFt: 300,
		SeekerOn: false, Armed: true, Alive: true, Mode: weapons.ModeWire,
		Age: 12, LastPingTime: -1, LaunchHeadDeg: 215, GyroCourseDeg: 210,
		ClearDistYd: 50, WireCut: false,
	}
	eng.FireControl.ActiveTorpedoes = []*weapons.Torpedo{fish, hostile}
	eng.Scenario.Objectives[0].Complete = true
	eng.PlotMarkers = []world.PlotMarker{
		{ID: "MARK-1", X: 1000, Y: -500},
		{ID: "MARK-2", X: -200, Y: 3000},
	}
	eng.SetPlotMarkerSeq(2)

	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.sav")
	if err := Save(path, eng); err != nil {
		t.Fatal(err)
	}

	got, err := LoadClean(path)
	if err != nil {
		t.Fatal(err)
	}

	gp := got.Scenario.Player
	assertNear(t, "player.X", gp.X, player.X)
	assertNear(t, "player.Y", gp.Y, player.Y)
	assertNear(t, "player.Depth", gp.DepthFt, player.DepthFt)
	assertNear(t, "player.Head", gp.HeadingDeg, player.HeadingDeg)
	assertNear(t, "player.OrdHead", gp.OrderedHead, player.OrderedHead)
	assertNear(t, "player.Spd", gp.SpeedKts, player.SpeedKts)
	assertNear(t, "player.OrdSpd", gp.OrderedSpeed, player.OrderedSpeed)
	assertNear(t, "player.Len", gp.LengthFt, 360)

	gs := findEnt(got, "enemy_foxtrot")
	assertNear(t, "sub.X", gs.X, enemySub.X)
	assertNear(t, "sub.Y", gs.Y, enemySub.Y)
	assertNear(t, "sub.Head", gs.HeadingDeg, enemySub.HeadingDeg)
	assertNear(t, "sub.OrdHead", gs.OrderedHead, enemySub.OrderedHead)
	if gs.Status != world.StatusActive {
		t.Fatalf("sub status %v", gs.Status)
	}
	if gs.AIState != "ATTACK" || !gs.ActiveSonar {
		t.Fatalf("sub AI/sonar: %s active=%v", gs.AIState, gs.ActiveSonar)
	}
	assertNear(t, "sub.Ping", gs.LastPingTime, 88.5)
	assertNear(t, "sub.Len", gs.LengthFt, 300)

	gship := findEnt(got, "enemy_grisha")
	if gship.Status != world.StatusSinking {
		t.Fatalf("ship status %v", gship.Status)
	}
	assertNear(t, "ship.X", gship.X, enemyShip.X)
	assertNear(t, "ship.Sink", gship.SinkRateFPM, 35)
	assertNear(t, "ship.Wreck", gship.WreckNoiseUntil, 500)
	assertNear(t, "ship.Len", gship.LengthFt, 235)

	if !got.Acoustics.Env.LayerSurveyKnown {
		t.Fatal("layer survey known lost")
	}
	if got.Sonar.ListenBand != acoustics.ListenHF {
		t.Fatalf("listen band %v", got.Sonar.ListenBand)
	}
	if got.Periscope.Order != acoustics.PeriMastRaise || got.Periscope.Zoom != acoustics.PeriZoomMed {
		t.Fatalf("peri order/zoom %v/%d", got.Periscope.Order, got.Periscope.Zoom)
	}
	assertNear(t, "peri.ext", got.Periscope.Extension, 0.85)
	assertNear(t, "peri.train", got.Periscope.TrainRelDeg, -25)
	if got.Periscope.LockEntityID != "enemy_foxtrot" {
		t.Fatalf("peri.lock=%q", got.Periscope.LockEntityID)
	}
	assertNear(t, "sonar.ping", got.Sonar.LastPingTime, 100.5)
	if len(got.Sonar.Contacts) != 1 || got.Sonar.Contacts[0].SourceEntityID != "enemy_foxtrot" {
		t.Fatalf("contacts %#v", got.Sonar.Contacts)
	}
	assertNear(t, "contact.activeFix", got.Sonar.Contacts[0].LastActiveFixAt, 110)
	assertNear(t, "contact.tmaCourse", got.Sonar.Contacts[0].TMACourseDeg, 78)
	assertNear(t, "contact.tmaSpeed", got.Sonar.Contacts[0].TMASpeedKts, 16)
	assertNear(t, "contact.tmaAccuracy", got.Sonar.Contacts[0].TMAAccuracy, 0.82)

	if got.FireControl.MagazineLeft != 17 || got.FireControl.EnemyMagazine["enemy_foxtrot"] != 9 {
		t.Fatalf("magazines player=%d enemy=%v", got.FireControl.MagazineLeft, got.FireControl.EnemyMagazine)
	}
	if got.FireControl.TorpedoSeq() < 7 {
		t.Fatalf("torpedo seq %d", got.FireControl.TorpedoSeq())
	}
	if len(got.FireControl.ActiveTorpedoes) != 2 {
		t.Fatalf("torps %d", len(got.FireControl.ActiveTorpedoes))
	}
	gf := got.FireControl.TorpedoByID("MK48-7")
	if gf == nil || !gf.Alive || gf.Mode != weapons.ModeSearch || !gf.GyroEnabled() {
		t.Fatalf("player fish %#v", gf)
	}
	assertNear(t, "fish.X", gf.X, 1300)
	assertNear(t, "fish.Head", gf.HeadingDeg, 0)
	assertNear(t, "fish.Launch", gf.LaunchHeadDeg, 0)
	assertNear(t, "fish.Ping", gf.LastPingTime, 118)
	assertNear(t, "fish.Clear", gf.ClearDistYd, 400)

	gh := got.FireControl.TorpedoByID("ETORP-3")
	if gh == nil || gh.TargetID != "player" || gh.Mode != weapons.ModeWire {
		t.Fatalf("hostile %#v", gh)
	}
	assertNear(t, "hostile.X", gh.X, 4400)
	assertNear(t, "hostile.Ord", gh.OrderedHead, 210)

	if !got.Scenario.Objectives[0].Complete {
		t.Fatal("objective not restored")
	}
	if len(got.PlotMarkers) != 2 {
		t.Fatalf("markers %d", len(got.PlotMarkers))
	}
	assertNear(t, "mark1.X", got.PlotMarkers[0].X, 1000)
	assertNear(t, "mark2.Y", got.PlotMarkers[1].Y, 3000)
	if got.PlotMarkerSeq() < 2 {
		t.Fatalf("marker seq %d", got.PlotMarkerSeq())
	}

	// Reload must not re-arm tube-clear (gyro already enabled).
	if !gf.GyroEnabled() {
		t.Fatal("gyro should stay enabled after load")
	}
	ord := gf.OrderedHead
	gf.Advance(0.1, got.Clock.GameTime+0.1, nil, nil, nil)
	if math.Abs(gf.OrderedHead-ord) > 1e-6 {
		t.Fatalf("OrderedHead changed after load advance: %.3f -> %.3f", ord, gf.OrderedHead)
	}
}

func TestLoadLegacySaveWithoutLengthFt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.sav")
	content := `# SSN688 Save File
format=1
scenario=Test
game_time=10.000
time_scale=1.000
paused=false

[entity:player]
name=USS Los Angeles
kind=0
side=0
status=0
signature=los_angeles
x=100.000
y=200.000
depth_ft=180.000
heading_deg=90.000
speed_kts=8.000
ordered_speed=8.000
ordered_depth=180.000
ordered_heading=90.000
active_sonar=false
last_ping_time=0.000
last_ping_power=0.000
ai_state=

[entity:enemy_surface]
name=Hostile DDG
kind=1
side=1
status=0
signature=udaloy
x=1000.000
y=2000.000
depth_ft=0.000
heading_deg=10.000
speed_kts=14.000
ordered_speed=14.000
ordered_depth=0.000
ordered_heading=10.000
active_sonar=false
last_ping_time=0.000
last_ping_power=0.000
ai_state=SEARCH

[objectives]
objective=obj_surface|Destroy hostile surface combatant|false|enemy_surface
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadClean(path)
	if err != nil {
		t.Fatal(err)
	}
	ship := findEnt(got, "enemy_surface")
	if ship.LengthFt != 535 {
		t.Fatalf("expected udaloy length fallback, got %.1f", ship.LengthFt)
	}
	assertNear(t, "legacy.x", got.Scenario.Player.X, 100)
}

func findEnt(e *sim.Engine, id string) *world.Entity {
	for _, ent := range e.Scenario.Entities {
		if ent.ID == id {
			return ent
		}
	}
	return nil
}

func assertNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.02 {
		t.Fatalf("%s: got %.4f want %.4f", name, got, want)
	}
}
