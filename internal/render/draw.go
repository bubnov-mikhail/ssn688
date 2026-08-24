package render

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	ScreenW = 1600
	ScreenH = 900
)

var (
	once       sync.Once
	faceLarge  font.Face
	faceMedium font.Face
	faceSmall  font.Face
	faceButton font.Face
	fontErr    error
)

func InitFonts() error {
	once.Do(func() {
		f, err := opentype.Parse(goregular.TTF)
		if err != nil {
			fontErr = err
			return
		}
		faceLarge, _ = opentype.NewFace(f, &opentype.FaceOptions{Size: 24, DPI: 72, Hinting: font.HintingFull})
		faceMedium, _ = opentype.NewFace(f, &opentype.FaceOptions{Size: 16, DPI: 72, Hinting: font.HintingFull})
		faceSmall, _ = opentype.NewFace(f, &opentype.FaceOptions{Size: 12, DPI: 72, Hinting: font.HintingFull})
		faceButton, _ = opentype.NewFace(f, &opentype.FaceOptions{Size: 11, DPI: 72, Hinting: font.HintingFull})
	})
	return fontErr
}

// ButtonLabelWidth returns the pixel width of a button label.
func ButtonLabelWidth(label string) int {
	if faceButton != nil {
		return font.MeasureString(faceButton, label).Ceil()
	}
	return len(label) * 6
}

// SmallLabelWidth returns the pixel width of a small (nav bar) label.
func SmallLabelWidth(label string) int {
	if faceSmall != nil {
		return font.MeasureString(faceSmall, label).Ceil()
	}
	return len(label) * 6
}

// SmallLabelBaseline returns the text baseline Y that vertically centers a
// small-font label inside a box of height h starting at boxY.
func SmallLabelBaseline(boxY, boxH int) int {
	if faceSmall == nil {
		return boxY + boxH/2 + 4
	}
	m := faceSmall.Metrics()
	return boxY + (boxH+m.Ascent.Ceil()-m.Descent.Ceil())/2
}

// ButtonLabelBaseline returns the text baseline Y that vertically centers a
// button-font label inside a box of height h starting at boxY.
func ButtonLabelBaseline(boxY, boxH int) int {
	if faceButton == nil {
		return boxY + boxH/2 + 4
	}
	m := faceButton.Metrics()
	return boxY + (boxH+m.Ascent.Ceil()-m.Descent.Ceil())/2
}

// LabelWidth returns the pixel width of medium (body) text.
func LabelWidth(label string) int {
	if faceMedium != nil {
		return font.MeasureString(faceMedium, label).Ceil()
	}
	return len(label) * 8
}

// ButtonWidth returns a button width that fits the label with horizontal padding.
func ButtonWidth(label string, hPad int) int {
	w := ButtonLabelWidth(label) + hPad
	if w < 32 {
		return 32
	}
	return w
}

// DrawButtonText draws label text using the button font face.
func DrawButtonText(screen *ebiten.Image, txt string, x, y int, clr color.Color) {
	if faceButton == nil {
		DrawText(screen, txt, x, y, clr, true)
		return
	}
	drawFace(screen, txt, x, y, clr, faceButton)
}

func DrawText(screen *ebiten.Image, txt string, x, y int, clr color.Color, small bool) {
	f := faceMedium
	if small {
		f = faceSmall
	}
	if f == nil {
		ebitenutil.DebugPrintAt(screen, txt, x, y)
		return
	}
	drawFace(screen, txt, x, y, clr, f)
}

func DrawTextLarge(screen *ebiten.Image, txt string, x, y int, clr color.Color) {
	if faceLarge == nil {
		ebitenutil.DebugPrintAt(screen, txt, x, y)
		return
	}
	drawFace(screen, txt, x, y, clr, faceLarge)
}

// DrawScreenTitle draws a primary instrument-panel title (large phosphor green).
func DrawScreenTitle(screen *ebiten.Image, txt string, x, y int) {
	DrawTextLarge(screen, txt, x, y, ColorPhosphorDim)
}

func drawFace(screen *ebiten.Image, txt string, x, y int, clr color.Color, f font.Face) {
	d := &font.Drawer{
		Dst:  screen,
		Src:  colorSource(clr),
		Face: f,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(txt)
}

// colorSource returns a reusable CPU-side uniform image for text glyphs.
// Never allocate ebiten.Image per draw call — that exhausts Metal drawables on macOS.
var colorSourceCache sync.Map

func colorSource(clr color.Color) image.Image {
	r, g, b, a := clr.RGBA()
	key := uint64(r)<<48 | uint64(g)<<32 | uint64(b)<<16 | uint64(a)
	if v, ok := colorSourceCache.Load(key); ok {
		return v.(image.Image)
	}
	src := image.NewUniform(clr)
	colorSourceCache.Store(key, src)
	return src
}

func FillRect(screen *ebiten.Image, x, y, w, h int, clr color.Color) {
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(w), float64(h), clr)
}

func DrawLine(screen *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	ebitenutil.DrawLine(screen, x1, y1, x2, y2, clr)
}

var (
	ColorBG            = color.RGBA{12, 12, 14, 255}
	ColorPanel         = color.RGBA{24, 24, 26, 255}
	ColorPanelStroke   = color.RGBA{40, 40, 44, 255}
	ColorMonitorFace   = color.RGBA{10, 10, 12, 255}
	ColorBorder        = color.RGBA{70, 72, 78, 255}
	ColorText          = color.RGBA{178, 180, 186, 255}
	ColorDim           = color.RGBA{0, 140, 100, 255}
	ColorWarn      = color.RGBA{255, 200, 0, 255}
	ColorDanger    = color.RGBA{255, 60, 40, 255}
	ColorSonar     = color.RGBA{0, 255, 120, 255}
	ColorActive    = color.RGBA{100, 200, 255, 255}
	ColorGrid          = color.RGBA{0, 70, 55, 255}
	ColorHighlight = color.RGBA{255, 255, 100, 255}
	ColorDebugCalm     = color.RGBA{60, 200, 80, 255}
	ColorDebugSearch   = color.RGBA{255, 210, 50, 255}
	ColorDebugAttack   = color.RGBA{255, 60, 40, 255}
	ColorDebugInactive = color.RGBA{100, 110, 120, 255}
	ColorDebugPlayer   = color.RGBA{80, 220, 255, 255}
	ColorDebugPanel    = color.RGBA{18, 18, 20, 220}
	ColorDebugRoute    = color.RGBA{190, 190, 200, 160}
	ColorDebugRouteWP  = color.RGBA{210, 210, 220, 200}
)
