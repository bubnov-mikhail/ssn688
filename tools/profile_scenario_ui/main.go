// Command profile_scenario_ui runs the scenario list screen for heap/pprof capture.
//
//	go run ./tools/profile_scenario_ui/ -pprof localhost:6060 -seconds 20
//
// While running, in another terminal:
//
//	curl -o /tmp/h1.gz http://localhost:6060/debug/pprof/heap
//	sleep 10 && curl -o /tmp/h2.gz http://localhost:6060/debug/pprof/heap
//	go tool pprof -top -base /tmp/h1.gz /tmp/h2.gz
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/ui"
)

const sampleRate = 44100

type profileGame struct {
	app       *ui.App
	frames    int
	maxFrames int
}

func (g *profileGame) Update() error {
	g.frames++
	if g.frames >= g.maxFrames {
		return ebiten.Termination
	}
	return g.app.Update()
}

func (g *profileGame) Draw(screen *ebiten.Image) {
	g.app.Draw(screen)
}

func (g *profileGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return render.ScreenW, render.ScreenH
}

func writeHeap(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pprof.WriteHeapProfile(f)
}

func main() {
	pprofAddr := flag.String("pprof", "localhost:6060", "pprof HTTP addr (empty = off)")
	seconds := flag.Int("seconds", 20, "run duration")
	scenarioID := flag.String("scenario", "taiwan_formosa_watch", "preferred scenario id")
	heapOut := flag.String("heap-out", "", "write final heap profile to path")
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Printf("pprof: http://%s/debug/pprof/", *pprofAddr)
			_ = http.ListenAndServe(*pprofAddr, nil)
		}()
	}

	if err := render.InitFonts(); err != nil {
		log.Fatal(err)
	}
	settings, err := config.Load()
	if err != nil {
		settings = config.DefaultSettings()
	}
	audioMgr := audio.NewManager(sampleRate)
	app := ui.NewApp(settings, audioMgr)
	app.EnterScenarioListForProfile(campaign.ScenarioID(*scenarioID))

	ebiten.SetWindowTitle("SSN688 scenario list profile")
	ebiten.SetWindowSize(settings.WindowWidth, settings.WindowHeight)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetScreenClearedEveryFrame(true)
	ebiten.SetScreenFilterEnabled(false)

	player := audioMgr.Stream()
	player.Play()

	maxFrames := *seconds * 60
	if maxFrames < 60 {
		maxFrames = 60
	}
	g := &profileGame{app: app, maxFrames: maxFrames}

	if err := writeHeap("/tmp/ssn688_heap_start.pb.gz"); err == nil {
		log.Printf("wrote /tmp/ssn688_heap_start.pb.gz")
	}

	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}

	out := *heapOut
	if out == "" {
		out = "/tmp/ssn688_heap_end.pb.gz"
	}
	if err := writeHeap(out); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("frames=%d heap profile: %s\n", g.frames, out)
	fmt.Printf("compare: go tool pprof -top -base /tmp/ssn688_heap_start.pb.gz %s\n", out)
	time.Sleep(100 * time.Millisecond)
}
