package main

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/simreplay"
)

const (
	recordWinW = 640
	recordWinH = 140
)

type recordScreen struct {
	title    string
	outPath  string
	maxSec   float64
	progress float64
	status   string
	mu       sync.Mutex
	done     bool
	err      error
}

func runRecordWithWindow(scenarioPath, missionID, out string, seed int64, maxSec float64) error {
	if maxSec <= 0 {
		maxSec = simreplay.DefaultMaxSec
	}
	if err := render.InitFonts(); err != nil {
		return err
	}
	g := &recordScreen{
		title:   fmt.Sprintf("Recording %s", missionID),
		outPath: out,
		maxSec:  maxSec,
		status:  "Starting simulation…",
	}
	go g.record(scenarioPath, missionID, out, seed, maxSec)

	ebiten.SetWindowSize(recordWinW, recordWinH)
	ebiten.SetWindowTitle("SSN-688 Sim Player — recording")
	if err := ebiten.RunGame(g); err != nil {
		return err
	}
	return g.err
}

func (g *recordScreen) record(scenarioPath, missionID, out string, seed int64, maxSec float64) {
	t0 := time.Now()
	rep, err := simreplay.RecordMission(simreplay.RecordOptions{
		ScenarioPath: scenarioPath,
		MissionID:    missionID,
		Seed:         seed,
		MaxSec:       maxSec,
		SampleSec:    1.0,
		PlayerIdle:   true,
		Progress: func(gameTime, maxSec float64) {
			frac := 0.0
			if maxSec > 0 {
				frac = gameTime / maxSec
			}
			if frac > 1 {
				frac = 1
			}
			g.mu.Lock()
			g.progress = frac
			g.status = fmt.Sprintf("Simulating %s / %s", formatTime(gameTime), formatTime(maxSec))
			g.mu.Unlock()
		},
	})
	if err != nil {
		g.mu.Lock()
		g.err = err
		g.status = "Recording failed"
		g.done = true
		g.mu.Unlock()
		return
	}
	if err := simreplay.Save(out, rep); err != nil {
		g.mu.Lock()
		g.err = err
		g.status = "Save failed"
		g.done = true
		g.mu.Unlock()
		return
	}
	g.mu.Lock()
	g.progress = 1
	g.status = fmt.Sprintf("Saved %d frames in %.1fs", len(rep.Frames), time.Since(t0).Seconds())
	g.err = nil
	g.done = true
	g.mu.Unlock()
}

func (g *recordScreen) Update() error {
	g.mu.Lock()
	done := g.done
	g.mu.Unlock()
	if done {
		return ebiten.Termination
	}
	return nil
}

func (g *recordScreen) Draw(screen *ebiten.Image) {
	g.mu.Lock()
	frac := g.progress
	status := g.status
	title := g.title
	outPath := g.outPath
	g.mu.Unlock()

	render.FillRect(screen, 0, 0, recordWinW, recordWinH, color.RGBA{10, 12, 16, 255})
	render.DrawText(screen, title, 20, 28, render.ColorPhosphor, true)
	render.DrawText(screen, outPath, 20, 50, render.ColorDim, true)
	render.DrawText(screen, status, 20, 72, render.ColorHighlight, true)

	const barX, barY, barW, barH = 20, 92, recordWinW-40, 16
	render.FillRect(screen, barX, barY, barW, barH, color.RGBA{32, 38, 48, 255})
	fillW := int(float64(barW) * frac)
	if fillW > 0 {
		render.FillRect(screen, barX, barY, fillW, barH, color.RGBA{60, 140, 200, 255})
	}
	pct := int(frac*100 + 0.5)
	render.DrawText(screen, fmt.Sprintf("%d%%", pct), barX+barW-44, barY+14, render.ColorPhosphor, true)
}

func (g *recordScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return recordWinW, recordWinH
}

func recordReplayCLI(scenarioPath, missionID, out string, seed int64, maxSec float64) error {
	if maxSec <= 0 {
		maxSec = simreplay.DefaultMaxSec
	}
	bar := simreplay.NewTerminalProgress("recording")
	defer bar.Finish()
	t0 := time.Now()
	rep, err := simreplay.RecordMission(simreplay.RecordOptions{
		ScenarioPath: scenarioPath,
		MissionID:    missionID,
		Seed:         seed,
		MaxSec:       maxSec,
		SampleSec:    1.0,
		PlayerIdle:   true,
		Progress:     bar.Update,
	})
	if err != nil {
		return err
	}
	if err := simreplay.Save(out, rep); err != nil {
		return err
	}
	fmt.Printf("recorded %d frames (%.0f min) -> %s in %.1fs\n",
		len(rep.Frames), rep.DurationSec/60, out, time.Since(t0).Seconds())
	return nil
}
