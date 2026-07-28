package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/ssn688/sim/assets"
	"github.com/ssn688/sim/internal/audio"
	"github.com/ssn688/sim/internal/config"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/ui"
	"github.com/ssn688/sim/internal/world"
)

const sampleRate = 44100

type Game struct {
	app    *ui.App
	player *ebitenaudio.Player
}

func (g *Game) Update() error {
	err := g.app.Update()
	if ui.IsQuit(err) {
		return ebiten.Termination
	}
	return err
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.app.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return render.ScreenW, render.ScreenH
}

func main() {
	if err := render.InitFonts(); err != nil {
		log.Printf("font init: %v", err)
	}

	if bathy, err := world.LoadBathymetry(assets.BathyChart); err != nil {
		log.Printf("bathymetry: %v", err)
	} else {
		world.SetDefaultBathymetry(bathy)
	}

	settings, err := config.Load()
	if err != nil {
		log.Printf("settings: %v", err)
		settings = config.DefaultSettings()
	}

	audioMgr := audio.NewManager(sampleRate)
	audioMgr.SetVolumes(settings.MasterVolume, settings.VoiceVolume, settings.EffectsVolume)

	app := ui.NewApp(settings, audioMgr)

	ebiten.SetWindowTitle("SSN-688(I) Hunter/Killer")
	ebiten.SetWindowSize(settings.WindowWidth, settings.WindowHeight)
	ebiten.SetFullscreen(settings.Fullscreen)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetScreenClearedEveryFrame(true)
	ebiten.SetScreenFilterEnabled(false)

	g := &Game{app: app}
	player := audioMgr.Stream()
	player.Play()
	g.player = player

	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		fmt.Println(err)
		log.Fatal(err)
	}
}
