// Command sim_player replays a headless AFK mission capture with DEBUG-style overlays.
//
//	go run ./tools/sim_player
//	go run ./tools/sim_player -scenario scenarios_generated/foo.json -record-only -mission m1
package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/simreplay"
	"github.com/bubnov-mikhail/ssn688/internal/theaterpreview"
)

const (
	screenW   = 1400
	screenH   = 920
	topH      = 200
	controlsH = 88
	skipSec   = 600.0
)

var speedSteps = []float64{1, 2, 8, 16, 32}

func main() {
	scenarioPath := flag.String("scenario", "", "scenario JSON (default: auto from scenarios_generated/)")
	missionID := flag.String("mission", "", "mission id for recording when no replay")
	replayPath := flag.String("replay", "", "replay JSON (default: pick from scenarios_generated/sim_replays/)")
	doRecord := flag.Bool("record", false, "record replay before play (or if file missing)")
	recordOnly := flag.Bool("record-only", false, "record replay and exit (no player window)")
	seed := flag.Int64("seed", 1, "sim RNG seed for recording")
	maxMin := flag.Float64("max-min", 90, "mission duration to simulate (minutes; default 90)")
	flag.Parse()

	if err := initWorkspaceRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	*scenarioPath = resolvePath(*scenarioPath)
	*replayPath = resolvePath(*replayPath)

	scPath, err := resolveScenarioPath(*scenarioPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scenario: %v\n", err)
		os.Exit(1)
	}

	scData, err := os.ReadFile(scPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read scenario: %v\n", err)
		os.Exit(1)
	}
	scDef, err := campaign.ParseScenarioJSON(scData, scPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse scenario: %v\n", err)
		os.Exit(1)
	}
	scenarioID := string(scDef.ID)

	repPath, err := resolveReplayPath(scenarioID, *replayPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		os.Exit(1)
	}

	mid := *missionID
	if repPath != "" {
		if peek, err := peekReplay(repPath); err == nil && peek.MissionID != "" {
			mid = peek.MissionID
		}
	}
	if mid == "" {
		mid, err = resolveMissionID(&scDef, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "mission: %v\n", err)
			os.Exit(1)
		}
	}

	if repPath == "" {
		repPath = filepath.Join(scenariosDir, "sim_replays", mid+".replay.json")
	}

	needRecord := *doRecord || *recordOnly
	if !needRecord {
		if _, err := os.Stat(repPath); os.IsNotExist(err) {
			fmt.Printf("replay missing — recording %s …\n", repPath)
			needRecord = true
		}
	}
	if needRecord {
		var err error
		if *recordOnly {
			err = recordReplayCLI(scPath, mid, repPath, *seed, *maxMin*60)
		} else {
			err = runRecordWithWindow(scPath, mid, repPath, *seed, *maxMin*60)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "record: %v\n", err)
			os.Exit(1)
		}
		if *recordOnly {
			return
		}
	}

	rep, err := simreplay.Load(repPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load replay: %v\n", err)
		os.Exit(1)
	}

	missionMap, err := theaterpreview.LoadMissionMap(scPath, rep.MissionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "map: %v\n", err)
		os.Exit(1)
	}

	if err := render.InitFonts(); err != nil {
		fmt.Fprintf(os.Stderr, "fonts: %v\n", err)
		os.Exit(1)
	}

	settings, _ := config.Load()
	lang := i18n.NormalizeLang(settings.Language)
	if len(rep.Comm) == 0 {
		fmt.Printf("replay has no COMM log — capturing traffic (seed %d)…\n", rep.Seed)
		comm, startSec, err := simreplay.CaptureCommTimeline(simreplay.RecordOptions{
			ScenarioPath: scPath,
			MissionID:    rep.MissionID,
			Seed:         rep.Seed,
			MaxSec:       rep.DurationSec,
			PlayerIdle:   true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "comm capture: %v\n", err)
		} else {
			rep.Comm = comm
			rep.MissionStartSec = startSec
		}
	}

	g := &playerGame{
		replay:  rep,
		mapView: theaterpreview.NewMapView(missionMap),
		title:   fmt.Sprintf("%s — %s (AFK)", rep.ScenarioID, rep.MissionTitle),
		lang:    lang,
	}
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("SSN-688 Sim Player")
	if err := ebiten.RunGame(g); err != nil {
		fmt.Fprintf(os.Stderr, "player: %v\n", err)
		os.Exit(1)
	}
}

func resolveMissionID(scDef *campaign.ScenarioDef, flagID string) (string, error) {
	if flagID != "" {
		if campaign.FindMission(scDef, campaign.MissionID(flagID)) == nil {
			return "", fmt.Errorf("mission %q not in scenario", flagID)
		}
		return flagID, nil
	}
	if len(scDef.Missions) == 1 {
		return string(scDef.Missions[0].ID), nil
	}
	if len(scDef.Missions) == 0 {
		return "", fmt.Errorf("scenario has no missions")
	}
	labels := make([]string, len(scDef.Missions))
	for i, m := range scDef.Missions {
		title := m.Title.GetText("en")
		if title == "" {
			title = string(m.ID)
		}
		labels[i] = title + " (" + string(m.ID) + ")"
	}
	idx, err := pickFromList("Select mission to record", labels)
	if err != nil || idx < 0 {
		return "", fmt.Errorf("mission selection cancelled")
	}
	return string(scDef.Missions[idx].ID), nil
}

type playerGame struct {
	replay      *simreplay.Replay
	mapView     *theaterpreview.MapView
	title       string
	lang        string
	playing     bool
	speedIdx    int
	curTime     float64
	scrub       bool
	commScroll  int
	panDragging bool
	panLastMX   int
	panLastMY   int
}

func (g *playerGame) mapRect() (x, y, w, h int) {
	return 0, topH, screenW-commPanelW, screenH - topH - controlsH
}

func (g *playerGame) Update() error {
	dt := 1.0 / 60.0
	if g.playing {
		g.curTime += dt * speedSteps[g.speedIdx]
		if g.curTime > g.replay.DurationSec {
			g.curTime = g.replay.DurationSec
			g.playing = false
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.playing = !g.playing
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || g.hitBtn("back10") {
		g.curTime = math.Max(0, g.curTime-skipSec)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) || g.hitBtn("fwd10") {
		g.curTime = math.Min(g.replay.DurationSec, g.curTime+skipSec)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) || g.hitBtn("play") {
		g.playing = !g.playing
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.mapView.FitAll()
	}
	for i, sp := range speedSteps {
		key := ebiten.Key1
		switch i {
		case 1:
			key = ebiten.Key2
		case 2:
			key = ebiten.Key3
		case 3:
			key = ebiten.Key4
		case 4:
			key = ebiten.Key5
		}
		if inpututil.IsKeyJustPressed(key) || g.hitBtn(fmt.Sprintf("sp%.0f", sp)) {
			g.speedIdx = i
		}
	}

	mx, my := ebiten.CursorPosition()
	mapX, mapY, mapW, mapH := g.mapRect()
	g.mapView.SetRect(mapX, mapY, mapW, mapH)
	inMap := g.mapView.ContainsScreen(mx, my)

	cy := screenH - controlsH
	onScrub := my >= cy+44 && my < cy+64

	// PLOT-style pan: MMB or RMB on map.
	panHeld := inMap && (ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight))
	if panHeld {
		if !g.panDragging {
			g.panDragging = true
			g.panLastMX, g.panLastMY = mx, my
		} else {
			g.mapView.PanByScreenDelta(mx-g.panLastMX, my-g.panLastMY)
			g.panLastMX, g.panLastMY = mx, my
		}
	} else {
		g.panDragging = false
	}

	if inMap {
		_, wheelY := ebiten.Wheel()
		if wheelY != 0 && !g.scrollCommWheel(mx, my) {
			g.mapView.ZoomAt(mx, my, wheelY > 0)
		}
	} else {
		g.scrollCommWheel(mx, my)
	}

	if !g.panDragging && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && onScrub {
		g.scrub = true
		g.scrubAt(mx)
	}
	if g.scrub {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && onScrub {
			g.scrubAt(mx)
		} else {
			g.scrub = false
		}
	}
	return nil
}

func (g *playerGame) scrubAt(mx int) {
	sx, sw := g.scrubRect()
	if sw <= 0 {
		return
	}
	t := float64(mx-sx) / float64(sw)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	g.curTime = t * g.replay.DurationSec
}

func (g *playerGame) scrubRect() (x, w int) {
	x = 120
	w = screenW - commPanelW - 240
	return x, w
}

func (g *playerGame) hitBtn(id string) bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	mx, my := ebiten.CursorPosition()
	x, y, w, h := g.buttonRect(id)
	return mx >= x && mx < x+w && my >= y && my < y+h
}

func (g *playerGame) buttonRect(id string) (x, y, w, h int) {
	const bh = 32
	cy := screenH - controlsH + 8
	switch id {
	case "back10":
		return 12, cy, 72, bh
	case "play":
		return 92, cy, 72, bh
	case "fwd10":
		return 172, cy, 72, bh
	case "sp1":
		return 260, cy, 36, bh
	case "sp2":
		return 300, cy, 36, bh
	case "sp8":
		return 340, cy, 36, bh
	case "sp16":
		return 380, cy, 40, bh
	case "sp32":
		return 424, cy, 40, bh
	default:
		return 0, 0, 0, 0
	}
}

func (g *playerGame) Draw(screen *ebiten.Image) {
	render.FillRect(screen, 0, 0, screenW, screenH, color.RGBA{10, 12, 16, 255})

	frame := g.currentFrame()
	g.drawHeader(screen, frame)
	g.drawStats(screen, frame)

	mapX, mapY, mapW, mapH := g.mapRect()
	g.mapView.SetRect(mapX, mapY, mapW, mapH)
	g.mapView.Draw(screen)
	g.drawOverlay(screen, frame)
	render.DrawText(screen, "MMB/RMB: pan   wheel: zoom   F: fit view", mapX+10, mapY+mapH-8, render.ColorDim, true)
	g.drawCommPanel(screen)
	g.drawControls(screen)
}

func (g *playerGame) currentFrame() simreplay.Frame {
	idx := g.replay.FrameAt(g.curTime)
	return g.replay.Frames[idx]
}

func (g *playerGame) drawHeader(screen *ebiten.Image, frame simreplay.Frame) {
	render.DrawText(screen, g.title, 12, 24, render.ColorPhosphor, true)
	timer := fmt.Sprintf("MISSION TIME  %s / %s", formatTime(g.curTime), formatTime(g.replay.DurationSec))
	render.DrawText(screen, timer, 12, 48, render.ColorHighlight, true)
	status := "PAUSED"
	if g.playing {
		status = fmt.Sprintf("PLAY x%.0f", speedSteps[g.speedIdx])
	}
	render.DrawText(screen, status, screenW-160, 24, render.ColorAmber, true)
	render.DrawText(screen, fmt.Sprintf("frame t=%.0fs", frame.Time), screenW-160, 48, render.ColorDim, true)
}

func (g *playerGame) drawStats(screen *ebiten.Image, frame simreplay.Frame) {
	const y0 = 68
	headers := []string{"SIDE", "ID", "NAME", "STATUS", "DEFCON"}
	colX := []int{12, 72, 220, 420, 620}
	for i, h := range headers {
		render.DrawText(screen, h, colX[i], y0, render.ColorDim, true)
	}
	render.DrawLine(screen, 8, float64(y0+6), float64(screenW-8), float64(y0+6), color.RGBA{60, 70, 90, 255})
	row := 0
	for _, u := range frame.Units {
		if row >= 6 {
			break
		}
		y := y0 + 22 + row*18
		vals := []string{u.Side, u.ID, trim(u.Name, 22), u.Status, fmt.Sprintf("%d", u.Defcon)}
		clr := simreplay.UnitDebugColor(u)
		if !u.Alive {
			clr = render.ColorDebugInactive
		}
		for i, v := range vals {
			render.DrawText(screen, v, colX[i], y, clr, true)
		}
		row++
	}
}

func (g *playerGame) drawOverlay(screen *ebiten.Image, frame simreplay.Frame) {
	for _, u := range frame.Units {
		g.drawDebugEntity(screen, u.X, u.Y, u.Heading, u.SpeedKts, simreplay.UnitDebugColor(u), u.Alive, labelFor(u))
		if u.Side == "ENEMY" && u.Alive {
			sx, sy, ok := g.mapView.WorldToScreen(u.X, u.Y)
			if ok {
				render.DrawText(screen, fmt.Sprintf("%d", u.Defcon), int(sx)-6, int(sy)+16, render.ColorAmber, true)
			}
		}
	}
	for _, w := range frame.Weapons {
		if !w.Alive {
			continue
		}
		clr := simreplay.WeaponColor(w)
		g.drawDebugEntity(screen, w.X, w.Y, w.Heading, w.SpeedKts, clr, true, w.Label)
		if w.Kind == simreplay.WeaponRBU || w.Kind == simreplay.WeaponRastrub {
			sx0, sy0, ok0 := g.mapView.WorldToScreen(w.X, w.Y)
			sx1, sy1, ok1 := g.mapView.WorldToScreen(w.X1, w.Y1)
			if ok0 && ok1 {
				render.DrawLine(screen, sx0, sy0, sx1, sy1, simreplay.WeaponTrailColor(w))
				if w.Kind == simreplay.WeaponRBU {
					render.DrawText(screen, "SPLASH", int(sx1)-20, int(sy1)-6, render.ColorAmber, true)
				}
			}
		}
	}
	for _, f := range frame.Flashes {
		sx, sy, ok := g.mapView.WorldToScreen(f.X, f.Y)
		if ok {
			render.DrawText(screen, f.Label, int(sx)-2, int(sy)+4, color.RGBA{200, 160, 255, 230}, true)
		}
	}
	g.drawPlotMarkers(screen, frame.Markers)
}

func labelFor(u simreplay.UnitSnap) string {
	if u.Name != "" {
		return u.Name
	}
	return u.ID
}

func (g *playerGame) drawPlotMarkers(screen *ebiten.Image, markers []simreplay.MarkerSnap) {
	const halfYd = 250.0
	clr := color.RGBA{220, 200, 80, 255}
	for _, m := range markers {
		sx, sy, ok := g.mapView.WorldToScreen(m.X, m.Y)
		if !ok {
			continue
		}
		sx0, _, ok0 := g.mapView.WorldToScreen(m.X-halfYd, m.Y)
		sx1, _, ok1 := g.mapView.WorldToScreen(m.X+halfYd, m.Y)
		halfPx := 6.0
		if ok0 && ok1 {
			halfPx = math.Abs(sx1-sx0) * 0.5
		}
		if halfPx < 4 {
			halfPx = 4
		}
		x0, y0 := sx-halfPx, sy-halfPx
		x1, y1 := sx+halfPx, sy+halfPx
		render.DrawLine(screen, x0, y0, x1, y0, clr)
		render.DrawLine(screen, x1, y0, x1, y1, clr)
		render.DrawLine(screen, x1, y1, x0, y1, clr)
		render.DrawLine(screen, x0, y1, x0, y0, clr)
		render.DrawLine(screen, x0, y0, x1, y1, clr)
		render.DrawLine(screen, x1, y0, x0, y1, clr)
		label := m.Name
		if label == "" {
			label = m.ID
		}
		render.DrawText(screen, label, int(sx)+int(halfPx)+4, int(sy)-2, clr, true)
	}
}

func (g *playerGame) drawDebugEntity(screen *ebiten.Image, wx, wy, heading, speedKts float64, clr color.Color, active bool, classLabel string) {
	sx, sy, ok := g.mapView.WorldToScreen(wx, wy)
	if !ok {
		return
	}
	if !active {
		clr = render.ColorDebugInactive
	}
	render.FillRect(screen, int(sx)-3, int(sy)-3, 7, 7, clr)
	rad := heading * math.Pi / 180
	ln := 14.0
	render.DrawLine(screen, sx, sy, sx+math.Sin(rad)*ln, sy-math.Cos(rad)*ln, clr)
	if classLabel != "" {
		render.DrawText(screen, classLabel, int(sx)+8, int(sy)-4, clr, true)
		render.DrawText(screen, fmt.Sprintf("%.0f kt", speedKts), int(sx)+8, int(sy)+8, clr, true)
	}
}

func (g *playerGame) drawControls(screen *ebiten.Image) {
	cy := screenH - controlsH
	render.FillRect(screen, 0, cy, screenW, controlsH, color.RGBA{18, 20, 26, 255})
	render.DrawLine(screen, 0, float64(cy), float64(screenW), float64(cy), color.RGBA{50, 60, 80, 255})

	g.drawBtn(screen, "back10", "<< 10m", false)
	playLabel := "PLAY"
	if g.playing {
		playLabel = "PAUSE"
	}
	g.drawBtn(screen, "play", playLabel, true)
	g.drawBtn(screen, "fwd10", "10m >>", false)
	for i, sp := range speedSteps {
		id := fmt.Sprintf("sp%.0f", sp)
		on := i == g.speedIdx
		g.drawBtn(screen, id, fmt.Sprintf("x%.0f", sp), on)
	}

	sx, sw := g.scrubRect()
	trackY := cy + 52
	render.FillRect(screen, sx, trackY, sw, 8, color.RGBA{40, 48, 60, 255})
	frac := 0.0
	if g.replay.DurationSec > 0 {
		frac = g.curTime / g.replay.DurationSec
	}
	knob := sx + int(frac*float64(sw))
	render.FillRect(screen, sx, trackY, knob-sx, 8, color.RGBA{60, 140, 200, 255})
	render.FillRect(screen, knob-4, trackY-4, 8, 16, render.ColorHighlight)
	render.DrawText(screen, formatTime(g.curTime), sx, trackY+20, render.ColorPhosphorDim, true)
	render.DrawText(screen, formatTime(g.replay.DurationSec), sx+sw-48, trackY+20, render.ColorPhosphorDim, true)
}

func (g *playerGame) drawBtn(screen *ebiten.Image, id, label string, hot bool) {
	x, y, w, h := g.buttonRect(id)
	bg := color.RGBA{32, 38, 48, 255}
	if hot {
		bg = color.RGBA{48, 72, 96, 255}
	}
	render.FillRect(screen, x, y, w, h, bg)
	render.DrawText(screen, label, x+8, y+22, render.ColorPhosphor, true)
}

func (g *playerGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

func formatTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
