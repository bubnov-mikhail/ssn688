package theaterpreview

import (
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const srcHeader = 40

// Canvas maps world yards to a mission preview background (PNG from render_theater_routes).
type Canvas struct {
	Base    *ebiten.Image
	MinX    float64
	MinY    float64
	MaxX    float64
	MaxY    float64
	Mission string
	srcW    int
	srcMapH int
	mapX    int
	mapY    int
	mapW    int
	mapH    int
	// drawn map rect (letterboxed inside mapX,mapY,mapW,mapH)
	drawX float64
	drawY float64
	drawW float64
	drawH float64
}

// LoadCanvas opens a theater preview PNG and bathy bounds for coordinate transform.
func LoadCanvas(previewPNG string, bathy *world.Bathymetry, missionID string) (*Canvas, error) {
	if bathy == nil || !bathy.Valid() {
		return nil, fmt.Errorf("invalid bathy")
	}
	f, err := os.Open(previewPNG)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	mapH := b.Dy() - srcHeader
	if mapH < 1 {
		return nil, fmt.Errorf("preview too small")
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	return &Canvas{
		Base:    ebiten.NewImageFromImage(img),
		MinX:    minX,
		MinY:    minY,
		MaxX:    maxX,
		MaxY:    maxY,
		Mission: missionID,
		srcW:    b.Dx(),
		srcMapH: mapH,
	}, nil
}

// FindPreviewPNG locates scenarios_generated/theater_previews/*__<mission>.png
func FindPreviewPNG(missionID string) (string, error) {
	dir := filepath.Join("scenarios_generated", "theater_previews")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var best string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		if strings.Contains(e.Name(), "__coast") || strings.HasPrefix(e.Name(), "00_") {
			continue
		}
		if strings.Contains(e.Name(), "__"+missionID+".png") {
			best = filepath.Join(dir, e.Name())
		}
	}
	if best == "" {
		return "", fmt.Errorf("no preview PNG for mission %q in %s", missionID, dir)
	}
	return best, nil
}

// Layout places the map in the screen rectangle (scales preview to fit, preserves aspect).
func (c *Canvas) Layout(mapX, mapY, mapW, mapH int) {
	c.mapX = mapX
	c.mapY = mapY
	c.mapW = mapW
	c.mapH = mapH
	if c == nil || c.srcW <= 0 || c.srcMapH <= 0 {
		return
	}
	sx := float64(mapW) / float64(c.srcW)
	sy := float64(mapH) / float64(c.srcMapH)
	s := math.Min(sx, sy)
	c.drawW = float64(c.srcW) * s
	c.drawH = float64(c.srcMapH) * s
	c.drawX = float64(mapX) + (float64(mapW)-c.drawW)/2
	c.drawY = float64(mapY) + (float64(mapH)-c.drawH)/2
}

// DrawBase blits the static bathy/routes preview (map portion only).
func (c *Canvas) DrawBase(screen *ebiten.Image) {
	if c == nil || c.Base == nil || c.drawW <= 0 {
		return
	}
	sub := c.Base.SubImage(image.Rect(0, srcHeader, c.srcW, srcHeader+c.srcMapH)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(c.drawW/float64(c.srcW), c.drawH/float64(c.srcMapH))
	op.GeoM.Translate(c.drawX, c.drawY)
	screen.DrawImage(sub, op)
}

// WorldToScreen converts yards to pixel coordinates on the drawn map.
func (c *Canvas) WorldToScreen(wx, wy float64) (sx, sy float64, ok bool) {
	if c == nil || c.drawW <= 0 {
		return 0, 0, false
	}
	spanX := c.MaxX - c.MinX
	spanY := c.MaxY - c.MinY
	if spanX <= 0 || spanY <= 0 {
		return 0, 0, false
	}
	const pad = 8.0
	innerW := float64(c.srcW) - 2*pad
	innerH := float64(c.srcMapH) - 2*pad
	px := pad + (wx-c.MinX)/spanX*innerW
	py := pad + (c.MaxY-wy)/spanY*innerH
	sx = c.drawX + px*(c.drawW/float64(c.srcW))
	sy = c.drawY + py*(c.drawH/float64(c.srcMapH))
	return sx, sy, true
}

// ContainsScreen reports whether a screen point is inside the map panel.
func (c *Canvas) ContainsScreen(px, py int) bool {
	return px >= c.mapX && px < c.mapX+c.mapW && py >= c.mapY && py < c.mapY+c.mapH
}
