package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"sync"
	"time"

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
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to file (stop on exit)")
	profilesec := flag.Int("profilesec", 0, "with -cpuprofile, auto-exit after N seconds")
	pprofAddr := flag.String("pprof", "", "serve net/http/pprof on addr (e.g. localhost:6060)")
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Printf("pprof listening on http://%s/debug/pprof/", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Printf("pprof server: %v", err)
			}
		}()
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		var once sync.Once
		stop := func() {
			once.Do(func() {
				pprof.StopCPUProfile()
				_ = f.Close()
			})
		}
		defer stop()
		if *profilesec > 0 {
			go func() {
				time.Sleep(time.Duration(*profilesec) * time.Second)
				log.Printf("profile window ended (%ds) — quitting", *profilesec)
				stop()
				os.Exit(0)
			}()
		}
	}

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
