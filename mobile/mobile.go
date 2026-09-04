// Package mobile is the ebitenmobile entry point for Android/iOS.
//
// Build AAR (from repo root):
//
//	ebitenmobile bind -target android -javapkg com.bubnov.ssn688 \
//	  -o mobile/android/ssn688lib/ssn688.aar ./mobile
//
// Activity must call SetDataRoot(getFilesDir().getAbsolutePath()) in onCreate
// before the EbitenView starts updating.
package mobile

import (
	"log"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/bubnov-mikhail/ssn688/internal/audio"
	"github.com/bubnov-mikhail/ssn688/internal/config"
	"github.com/bubnov-mikhail/ssn688/internal/layout"
	"github.com/bubnov-mikhail/ssn688/internal/render"
	"github.com/bubnov-mikhail/ssn688/internal/ui"
)

const sampleRate = 44100

func init() {
	mobile.SetGame(&lazyGame{})
}

// SetDataRoot sets the Android/iOS files directory (e.g. Context.getFilesDir()).
// Saves → <root>/ssn688/saves, scenarios → <root>/ssn688/scenarios,
// import inbox → <root>/ssn688/import.
// Call from Activity.onCreate before the game view runs.
func SetDataRoot(path string) {
	config.SetDataRoot(path)
}

// DataRoot returns the path last set via SetDataRoot (empty if unset).
func DataRoot() string {
	return config.DataRoot()
}

// Dummy forces gomobile to compile this package (needs an exported symbol).
func Dummy() {}

// lazyGame defers App construction until the first Update so SetDataRoot
// from Java can run first (init() of this package runs at library load).
type lazyGame struct {
	once   sync.Once
	app    *ui.App
	player *ebitenaudio.Player
}

func (g *lazyGame) ensure() {
	g.once.Do(func() {
		if err := render.InitFonts(); err != nil {
			log.Printf("font init: %v", err)
		}
		settings, err := config.Load()
		if err != nil {
			log.Printf("settings: %v", err)
			settings = config.DefaultSettings()
		}
		audioMgr := audio.NewManager(sampleRate)
		audioMgr.SetVolumes(settings.MasterVolume, settings.VoiceVolume, settings.EffectsVolume)
		g.app = ui.NewApp(settings, audioMgr)
		ebiten.SetFullscreen(true)
		ebiten.SetVsyncEnabled(true)
		ebiten.SetScreenClearedEveryFrame(true)
		ebiten.SetScreenFilterEnabled(false)
		player := audioMgr.Stream()
		player.Play()
		g.player = player
	})
}

func (g *lazyGame) Update() error {
	g.ensure()
	err := g.app.Update()
	if ui.IsQuit(err) {
		return ebiten.Termination
	}
	return err
}

func (g *lazyGame) Draw(screen *ebiten.Image) {
	g.ensure()
	g.app.Draw(screen)
}

func (g *lazyGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := outsideWidth, outsideHeight
	if h > w {
		w, h = h, w
	}
	if h < 1 {
		h = 1
	}
	logicalH := layout.BaseScreenH
	logicalW := int(math.Round(float64(logicalH) * float64(w) / float64(h)))
	render.SetLogicalSize(logicalW, logicalH)
	return render.ScreenW, render.ScreenH
}
