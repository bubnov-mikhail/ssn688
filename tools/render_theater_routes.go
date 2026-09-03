// Command render_theater_routes writes PNG previews with bathy + mission routes.
//
//	go run ./tools/render_theater_routes.go
//	go run ./tools/render_theater_routes.go -scenario scenarios_generated/taiwan_formosa_watch.json
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	_ "image/jpeg"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const defaultScenario = "scenarios_generated/taiwan_formosa_watch.json"

var (
	defaultPreviewOut = filepath.Join("scenarios_generated", "theater_previews")
	defaultBathyDir   = filepath.Join("scenarios_generated", "theater_bathy")
)

// theaterGeo matches tools/gen_bathy_zone.py ZoneSpec centers (world yards = offset from center).
type theaterGeo struct {
	centerLat, centerLon float64
}

var theaterGeoCatalog = map[string]theaterGeo{
	"taiwan_east":   {23.97, 121.68},
	"taiwan_strait": {24.48, 118.70},
	"taiwan_penghu": {23.58, 119.52},
	"taiwan_south":  {21.97, 120.92},
	"taiwan_north":  {24.75, 121.95},
	"taiwan_lanyu":  {22.05, 121.55},
	"taiwan_overview": {23.40, 120.35},
}

const (
	metersPerDegLat       = 111320.0
	yardsPerMeter         = 1.0936133
	overviewOceanDepthFt  = 3500.0
)

var briefMapAmberBorder = color.RGBA{255, 176, 32, 255}

var routeColors = []color.RGBA{
	{255, 220, 60, 255}, {80, 200, 255, 255}, {255, 120, 80, 255},
	{160, 255, 120, 255}, {220, 120, 255, 255}, {255, 180, 220, 255},
	{180, 220, 255, 255}, {255, 255, 180, 255}, {120, 255, 200, 255},
	{255, 140, 140, 255}, {200, 200, 200, 255},
}

type missionPreview struct {
	num                            int
	id, title, theater             string
	bathy                          *world.Bathymetry
	latMin, latMax, lonMin, lonMax float64
	geoOK                          bool
}

func main() {
	scenarioPath := flag.String("scenario", defaultScenario, "scenario JSON with theaters and routes")
	outDir := flag.String("out", defaultPreviewOut, "output directory")
	missionFilter := flag.String("mission", "", "render only this mission id (e.g. tw_attribution)")
	coastOnly := flag.Bool("coast-only", false, "shoreline template only — no routes, no bathy shading")
	flag.Parse()

	data, err := os.ReadFile(*scenarioPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read scenario: %v\n", err)
		os.Exit(1)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		panic(err)
	}
	theaters := map[string]map[string]any{}
	for _, raw := range doc["theaters"].([]any) {
		th := raw.(map[string]any)
		theaters[th["id"].(string)] = th
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}

	if *coastOnly {
		if *missionFilter == "" {
			fmt.Fprintln(os.Stderr, "-coast-only requires -mission <id>")
			os.Exit(1)
		}
		for missionNum, mraw := range doc["missions"].([]any) {
			m := mraw.(map[string]any)
			mid := m["id"].(string)
			if mid != *missionFilter {
				continue
			}
			tid := m["theater_id"].(string)
			th := theaters[tid]
			bathy := loadTheater(th)
			title, _ := m["title"].(map[string]any)
			name, _ := title["en"].(string)
			prefix := fmt.Sprintf("%02d", missionNum+1)
			out := filepath.Join(*outDir, prefix+"_"+tid+"__"+mid+"__coast.png")
			renderCoastTemplate(bathy, tid+" — "+name, out)
			fmt.Printf("wrote %s (coast template)\n", out)
			return
		}
		fmt.Fprintf(os.Stderr, "mission %q not found\n", *missionFilter)
		os.Exit(1)
	}

	for _, old := range stalePreviewPaths(*outDir) {
		_ = os.Remove(old)
	}

	scenarioID, _ := doc["id"].(string)
	if scenarioID == "" {
		scenarioID = "scenario"
	}
	scenarioTitle, _ := doc["title"].(map[string]any)["en"].(string)
	if scenarioTitle == "" {
		scenarioTitle = scenarioID
	}

	manifest := make([]map[string]any, 0)
	missions := make([]missionPreview, 0)
	for missionNum, mraw := range doc["missions"].([]any) {
		m := mraw.(map[string]any)
		mid := m["id"].(string)
		tid := m["theater_id"].(string)
		th := theaters[tid]
		bathy := loadTheater(th)
		title, _ := m["title"].(map[string]any)
		name, _ := title["en"].(string)
		routes := parseRoutes(m["routes"].([]any))
		labels := routeLabelsFromUnits(m)
		prefix := fmt.Sprintf("%02d", missionNum+1)
		out := filepath.Join(*outDir, prefix+"_"+tid+"__"+mid+".png")
		var zones []engagementZone
		if mid == "tw_attribution" {
			zones = twAttributionEngagementZones
		}
		sub := subtitleFor(bathy, routes)
		if len(zones) > 0 {
			sub += " | DEFCON 3 | orange = engagement zones"
		}
		stats := renderPreview(bathy, routes, labels, parseUnitSpawns(m), tid+" — "+name, sub, out, zones)
		stats["mission"] = mid
		stats["mission_num"] = missionNum + 1
		stats["theater"] = tid
		stats["path"] = out
		manifest = append(manifest, stats)
		fmt.Printf("%s: land=%.1f%% route_land_hits=%v\n", filepath.Base(out), stats["land_pct"], stats["hits"])

		latMin, latMax, lonMin, lonMax, geoOK := theaterLatLonBounds(tid, bathy)
		missions = append(missions, missionPreview{
			num: missionNum + 1, id: mid, title: name, theater: tid, bathy: bathy,
			latMin: latMin, latMax: latMax, lonMin: lonMin, lonMax: lonMax, geoOK: geoOK,
		})
	}

	overviewPath := filepath.Join(*outDir, "00_"+scenarioID+"__overview.png")
	ovStats := renderScenarioOverview(missions, scenarioTitle, overviewPath)
	ovStats["scenario"] = scenarioID
	ovStats["path"] = overviewPath
	manifest = append([]map[string]any{ovStats}, manifest...)
	fmt.Printf("%s: missions=%d\n", filepath.Base(overviewPath), len(missions))

	for i, m := range missions {
		mraw := doc["missions"].([]any)[i].(map[string]any)
		cover := decodeCoverBlob(mraw)
		briefPath := filepath.Join(*outDir, fmt.Sprintf("%02d_%s__brief_map.png", m.num, m.id))
		bStats := renderMissionBriefMap(missions, i, cover, briefPath)
		bStats["mission"] = m.id
		bStats["mission_num"] = m.num
		bStats["path"] = briefPath
		bStats["has_cover"] = len(cover) > 0
		manifest = append(manifest, bStats)
		fmt.Printf("%s: past=%d current=%d cover=%v\n", filepath.Base(briefPath), i, i+1, len(cover) > 0)
	}

	idx := filepath.Join(*outDir, "manifest.json")
	b, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(idx, append(b, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d previews -> %s\n", len(manifest), *outDir)
}

func stalePreviewPaths(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		name := e.Name()
		if strings.Contains(name, "__location.png") {
			out = append(out, filepath.Join(dir, name))
			continue
		}
		if len(name) >= 3 && name[0] >= '0' && name[0] <= '9' && name[1] >= '0' && name[1] <= '9' && name[2] == '_' {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

func legacyPreviewPaths(dir string) []string {
	return stalePreviewPaths(dir)
}

func yardsToLatLon(centerLat, centerLon, xYd, yYd float64) (latDeg, lonDeg float64) {
	mPerDegLon := metersPerDegLat * math.Cos(centerLat*math.Pi/180)
	latDeg = centerLat + (yYd/yardsPerMeter)/metersPerDegLat
	lonDeg = centerLon + (xYd/yardsPerMeter)/mPerDegLon
	return latDeg, lonDeg
}

func theaterLatLonBounds(tid string, bathy *world.Bathymetry) (latMin, latMax, lonMin, lonMax float64, ok bool) {
	return bathyLatLonBounds(tid, bathy)
}

func bathyLatLonBounds(tid string, bathy *world.Bathymetry) (latMin, latMax, lonMin, lonMax float64, ok bool) {
	geo, ok := theaterGeoCatalog[tid]
	if !ok || bathy == nil || !bathy.Valid() {
		return 0, 0, 0, 0, false
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	latSW, lonSW := yardsToLatLon(geo.centerLat, geo.centerLon, minX, minY)
	latNE, lonNE := yardsToLatLon(geo.centerLat, geo.centerLon, maxX, maxY)
	latMin = math.Min(latSW, latNE)
	latMax = math.Max(latSW, latNE)
	lonMin = math.Min(lonSW, lonNE)
	lonMax = math.Max(lonSW, lonNE)
	return latMin, latMax, lonMin, lonMax, true
}

func renderScenarioOverview(missions []missionPreview, title, path string) map[string]any {
	const imgW, imgH = 1280, 1080
	const headerH = 44
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{8, 18, 32, 255}}, image.Point{}, draw.Src)
	drawHeader(img, title, "mission theaters in region — inset = bathy chart extent")

	mapH := imgH - headerH
	regional := loadRegionalOverviewBathy()
	latMin, latMax, lonMin, lonMax, geoOK := overviewViewBounds(missions, regional)
	if !geoOK {
		drawString(img, 12, headerH+12, "no geo catalog for theaters — overview skipped", color.RGBA{255, 180, 80, 255})
		writePNG(path, img)
		return map[string]any{"kind": "overview", "missions": len(missions), "geo": false}
	}

	latPad := (latMax - latMin) * 0.06
	lonPad := (lonMax - lonMin) * 0.06
	if regional != nil {
		latPad = (latMax - latMin) * 0.005
		lonPad = (lonMax - lonMin) * 0.005
	}
	if latPad < 0.08 && regional == nil {
		latPad = 0.08
	}
	if lonPad < 0.08 && regional == nil {
		lonPad = 0.08
	}
	latMin -= latPad
	latMax += latPad
	lonMin -= lonPad
	lonMax += lonPad

	latLonToPx := func(lat, lon float64) (int, int) {
		const pad = 24.0
		sx := pad + (lon-lonMin)/(lonMax-lonMin)*(float64(imgW)-2*pad)
		sy := float64(headerH) + pad + (latMax-lat)/(latMax-latMin)*(float64(mapH)-2*pad)
		return int(sx), int(sy)
	}
	pxToLatLon := func(px, py int) (lat, lon float64) {
		const pad = 24.0
		lon = lonMin + (float64(px)-pad)/(float64(imgW)-2*pad)*(lonMax-lonMin)
		lat = latMax - (float64(py)-float64(headerH)-pad)/(float64(mapH)-2*pad)*(latMax-latMin)
		return lat, lon
	}

	theaters := uniqueTheaterBathies(missions)
	if regional != nil {
		geo := theaterGeoCatalog["taiwan_overview"]
		drawRegionalGrayBathy(img, regional, geo, headerH, imgW, imgH, 5, pxToLatLon, overviewOceanDepthFt)
	} else {
		drawMergedTheaterGrayBathy(img, theaters, headerH, imgW, imgH, 5, pxToLatLon, overviewOceanDepthFt)
	}

	// Mission chart rectangles with inset bathy + labels.
	boxes := make([]map[string]any, 0, len(missions))
	for _, m := range missions {
		if !m.geoOK {
			continue
		}
		x0, y0 := latLonToPx(m.latMax, m.lonMin)
		x1, y1 := latLonToPx(m.latMin, m.lonMax)
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		rect := image.Rect(x0, y0, x1, y1)
		if rect.Dx() < 4 || rect.Dy() < 4 {
			continue
		}
		border := routeColors[(m.num-1)%len(routeColors)]
		drawBathyInset(img, m.bathy, rect)
		drawRectBorder(img, rect, border, 2)
		label := fmt.Sprintf("M%d %s", m.num, shortenLabel(m.title, 22))
		lx, ly := rect.Min.X+3, rect.Min.Y-11
		if ly < headerH+2 {
			ly = rect.Min.Y + 4
		}
		drawLabel(img, lx, ly, label, border)
		boxes = append(boxes, map[string]any{
			"mission": m.id, "theater": m.theater, "num": m.num,
			"lat": []float64{m.latMin, m.latMax}, "lon": []float64{m.lonMin, m.lonMax},
		})
	}

	drawLatLonGrid(img, latMin, latMax, lonMin, lonMax, headerH, imgW, mapH, latLonToPx)
	writePNG(path, img)
	return map[string]any{"kind": "overview", "missions": len(missions), "geo": true, "boxes": boxes}
}

type overviewLayout struct {
	imgW, imgH, headerH          int
	latMin, latMax, lonMin, lonMax float64
	latLonToPx                   func(lat, lon float64) (int, int)
	pxToLatLon                   func(px, py int) (lat, lon float64)
	regional                     *world.Bathymetry
	theaters                     map[string]*world.Bathymetry
	geoOK                        bool
}

func buildOverviewLayout(missions []missionPreview, imgW, imgH, headerH int) overviewLayout {
	lay := overviewLayout{imgW: imgW, imgH: imgH, headerH: headerH}
	mapH := imgH - headerH
	regional := loadRegionalOverviewBathy()
	lay.regional = regional
	lay.theaters = uniqueTheaterBathies(missions)
	latMin, latMax, lonMin, lonMax, geoOK := overviewViewBounds(missions, regional)
	lay.geoOK = geoOK
	if !geoOK {
		return lay
	}
	latPad := (latMax - latMin) * 0.06
	lonPad := (lonMax - lonMin) * 0.06
	if regional != nil {
		latPad = (latMax - latMin) * 0.005
		lonPad = (lonMax - lonMin) * 0.005
	}
	if latPad < 0.08 && regional == nil {
		latPad = 0.08
	}
	if lonPad < 0.08 && regional == nil {
		lonPad = 0.08
	}
	latMin -= latPad
	latMax += latPad
	lonMin -= lonPad
	lonMax += lonPad
	lay.latMin, lay.latMax, lay.lonMin, lay.lonMax = latMin, latMax, lonMin, lonMax
	lay.latLonToPx = func(lat, lon float64) (int, int) {
		const pad = 24.0
		sx := pad + (lon-lonMin)/(lonMax-lonMin)*(float64(imgW)-2*pad)
		sy := float64(headerH) + pad + (latMax-lat)/(latMax-latMin)*(float64(mapH)-2*pad)
		return int(sx), int(sy)
	}
	lay.pxToLatLon = func(px, py int) (lat, lon float64) {
		const pad = 24.0
		lon = lonMin + (float64(px)-pad)/(float64(imgW)-2*pad)*(lonMax-lonMin)
		lat = latMax - (float64(py)-float64(headerH)-pad)/(float64(mapH)-2*pad)*(latMax-latMin)
		return lat, lon
	}
	return lay
}

func drawOverviewGrayBathy(img *image.RGBA, lay overviewLayout) {
	if !lay.geoOK {
		return
	}
	pxStep := 4
	if lay.imgW >= 800 {
		pxStep = 2
	}
	if lay.regional != nil {
		geo := theaterGeoCatalog["taiwan_overview"]
		drawRegionalGrayBathy(img, lay.regional, geo, lay.headerH, lay.imgW, lay.imgH, pxStep, lay.pxToLatLon, overviewOceanDepthFt)
	} else {
		drawMergedTheaterGrayBathy(img, lay.theaters, lay.headerH, lay.imgW, lay.imgH, pxStep, lay.pxToLatLon, overviewOceanDepthFt)
	}
}

func missionPreviewRect(m missionPreview, latLonToPx func(float64, float64) (int, int)) image.Rectangle {
	if !m.geoOK {
		return image.Rectangle{}
	}
	x0, y0 := latLonToPx(m.latMax, m.lonMin)
	x1, y1 := latLonToPx(m.latMin, m.lonMax)
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return image.Rect(x0, y0, x1, y1)
}

func drawBathyInsetGray(img *image.RGBA, bathy *world.Bathymetry, rect image.Rectangle) {
	if bathy == nil || !bathy.Valid() {
		return
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	w := rect.Dx()
	h := rect.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	stepX := int(math.Max(1, math.Ceil(float64(w)/80)))
	stepY := int(math.Max(1, math.Ceil(float64(h)/80)))
	for py := 0; py < h; py += stepY {
		for px := 0; px < w; px += stepX {
			wx := minX + (maxX-minX)*float64(px)/float64(w)
			wy := maxY - (maxY-minY)*float64(py)/float64(h)
			col := overviewGrayColor(bathy.DepthAtFt(wx, wy))
			for dy := 0; dy < stepY; dy++ {
				for dx := 0; dx < stepX; dx++ {
					x, y := rect.Min.X+px+dx, rect.Min.Y+py+dy
					if image.Pt(x, y).In(rect) {
						img.Set(x, y, col)
					}
				}
			}
		}
	}
}

func drawCoverInset(img *image.RGBA, cover []byte, rect image.Rectangle) bool {
	if len(cover) == 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return false
	}
	dec, _, err := image.Decode(bytes.NewReader(cover))
	if err != nil {
		return false
	}
	src := imageToRGBA(dec)
	dstW, dstH := rect.Dx(), rect.Dy()
	scaled := resizeCover(src, dstW, dstH)
	draw.Draw(img, rect, scaled, image.Point{}, draw.Over)
	return true
}

func imageToRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok {
		return r
	}
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	return out
}

func resizeCover(src *image.RGBA, w, h int) *image.RGBA {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw <= 0 || sh <= 0 {
		return out
	}
	scale := math.Max(float64(w)/float64(sw), float64(h)/float64(sh))
	nw := int(math.Ceil(float64(sw) * scale))
	nh := int(math.Ceil(float64(sh) * scale))
	tmp := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := int(float64(y) / scale)
		if sy >= sh {
			sy = sh - 1
		}
		for x := 0; x < nw; x++ {
			sx := int(float64(x) / scale)
			if sx >= sw {
				sx = sw - 1
			}
			tmp.Set(x, y, src.At(src.Bounds().Min.X+sx, src.Bounds().Min.Y+sy))
		}
	}
	offX := (nw - w) / 2
	offY := (nh - h) / 2
	draw.Draw(out, out.Bounds(), tmp, image.Point{offX, offY}, draw.Src)
	return out
}

func decodeCoverBlob(m map[string]any) []byte {
	cover, _ := m["cover"].(map[string]any)
	if cover == nil {
		return nil
	}
	b64, _ := cover["data_b64"].(string)
	if b64 == "" {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil
	}
	return raw
}

// renderMissionBriefMap draws the mission-brief regional map: gray overview,
// past missions B&W, current mission colored with cover inset (or color bathy fallback in previews).
func renderMissionBriefMap(missions []missionPreview, curIdx int, cover []byte, path string) map[string]any {
	const imgW, imgH = 880, 1040
	const headerH = 0
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{8, 18, 32, 255}}, image.Point{}, draw.Src)
	if curIdx < 0 || curIdx >= len(missions) {
		writePNG(path, img)
		return map[string]any{"kind": "brief_map", "geo": false}
	}
	cur := missions[curIdx]
	lay := buildOverviewLayout(missions, imgW, imgH, headerH)
	if !lay.geoOK {
		drawString(img, 12, 12, "no geo catalog — brief map skipped", color.RGBA{255, 180, 80, 255})
		writePNG(path, img)
		return map[string]any{"kind": "brief_map", "geo": false, "current": cur.id}
	}
	drawOverviewGrayBathy(img, lay)
	for i := 0; i <= curIdx; i++ {
		m := missions[i]
		if !m.geoOK {
			continue
		}
		rect := missionPreviewRect(m, lay.latLonToPx)
		if rect.Dx() < 4 || rect.Dy() < 4 {
			continue
		}
		if i < curIdx {
			drawBathyInsetGray(img, m.bathy, rect)
			drawRectBorder(img, rect, color.RGBA{210, 210, 210, 255}, 1)
		} else {
			if len(cover) > 0 {
				if !drawCoverInset(img, cover, rect) {
					drawBathyInset(img, m.bathy, rect)
				}
			} else {
				drawBathyInset(img, m.bathy, rect)
			}
			drawRectBorder(img, rect, briefMapAmberBorder, 1)
		}
	}
	writePNG(path, img)
	return map[string]any{
		"kind": "brief_map", "geo": true, "current": cur.id,
		"past_missions": curIdx, "has_cover": len(cover) > 0,
	}
}

func overviewGeoBounds(missions []missionPreview) (latMin, latMax, lonMin, lonMax float64, ok bool) {
	first := true
	for _, m := range missions {
		if !m.geoOK {
			continue
		}
		ok = true
		if first {
			latMin, latMax = m.latMin, m.latMax
			lonMin, lonMax = m.lonMin, m.lonMax
			first = false
			continue
		}
		if m.latMin < latMin {
			latMin = m.latMin
		}
		if m.latMax > latMax {
			latMax = m.latMax
		}
		if m.lonMin < lonMin {
			lonMin = m.lonMin
		}
		if m.lonMax > lonMax {
			lonMax = m.lonMax
		}
	}
	return latMin, latMax, lonMin, lonMax, ok
}

// overviewViewBounds uses the regional overview chart when available so gray
// bathy fills the entire frame; otherwise falls back to mission theater union.
func overviewViewBounds(missions []missionPreview, regional *world.Bathymetry) (latMin, latMax, lonMin, lonMax float64, ok bool) {
	if regional != nil {
		if latMin, latMax, lonMin, lonMax, ok = bathyLatLonBounds("taiwan_overview", regional); ok {
			return latMin, latMax, lonMin, lonMax, true
		}
	}
	return overviewGeoBounds(missions)
}

func uniqueTheaterBathies(missions []missionPreview) map[string]*world.Bathymetry {
	out := map[string]*world.Bathymetry{}
	for _, m := range missions {
		if m.bathy != nil && m.bathy.Valid() {
			out[m.theater] = m.bathy
		}
	}
	return out
}

func loadRegionalOverviewBathy() *world.Bathymetry {
	for _, name := range []string{"taiwan_overview.bin", "scenario_overview.bin"} {
		path := filepath.Join(defaultBathyDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		b, err := world.LoadBathymetry(raw)
		if err != nil {
			continue
		}
		return &b
	}
	return nil
}

func latLonToTheaterYards(geo theaterGeo, lat, lon float64) (xYd, yYd float64) {
	mPerDegLon := metersPerDegLat * math.Cos(geo.centerLat*math.Pi/180)
	xYd = (lon - geo.centerLon) * mPerDegLon * yardsPerMeter
	yYd = (lat - geo.centerLat) * metersPerDegLat * yardsPerMeter
	return xYd, yYd
}

func sampleTheaterDepth(tid string, bathy *world.Bathymetry, lat, lon float64) (depth float64, ok bool) {
	geo, ok := theaterGeoCatalog[tid]
	if !ok || bathy == nil || !bathy.Valid() {
		return 0, false
	}
	xYd, yYd := latLonToTheaterYards(geo, lat, lon)
	if !bathy.OnChart(xYd, yYd) {
		return 0, false
	}
	return bathy.DepthAtFt(xYd, yYd), true
}

func sampleMergedTheaterDepth(lat, lon float64, theaters map[string]*world.Bathymetry, offChartDepth float64) float64 {
	maxDepth := offChartDepth
	hit := false
	for tid, bathy := range theaters {
		d, ok := sampleTheaterDepth(tid, bathy, lat, lon)
		if !ok {
			continue
		}
		hit = true
		if d <= 0 {
			return d
		}
		if d > maxDepth {
			maxDepth = d
		}
	}
	if !hit {
		return offChartDepth
	}
	return maxDepth
}

func overviewGrayColor(depthFt float64) color.RGBA {
	if depthFt <= 0 {
		return color.RGBA{108, 106, 100, 255}
	}
	depth := math.Min(depthFt, 8000)
	t := math.Log1p(depth) / math.Log1p(8000)
	v := 42.0 + (1-t)*78.0
	return color.RGBA{uint8(v), uint8(v), uint8(v + 6), 255}
}

func drawRegionalGrayBathy(img *image.RGBA, bathy *world.Bathymetry, geo theaterGeo, headerH, imgW, imgH, pxStep int, pxToLatLon func(int, int) (float64, float64), offChartDepth float64) {
	if bathy == nil || !bathy.Valid() || pxStep < 1 {
		return
	}
	for py := headerH; py < imgH; py += pxStep {
		for px := 0; px < imgW; px += pxStep {
			lat, lon := pxToLatLon(px, py)
			xYd, yYd := latLonToTheaterYards(geo, lat, lon)
			depth := offChartDepth
			if bathy.OnChart(xYd, yYd) {
				depth = bathy.DepthAtFt(xYd, yYd)
			}
			col := overviewGrayColor(depth)
			for dy := 0; dy < pxStep; dy++ {
				for dx := 0; dx < pxStep; dx++ {
					x, y := px+dx, py+dy
					if image.Pt(x, y).In(img.Bounds()) {
						img.Set(x, y, col)
					}
				}
			}
		}
	}
}

func drawMergedTheaterGrayBathy(img *image.RGBA, theaters map[string]*world.Bathymetry, headerH, imgW, imgH, pxStep int, pxToLatLon func(int, int) (float64, float64), offChartDepth float64) {
	if len(theaters) == 0 || pxStep < 1 {
		return
	}
	for py := headerH; py < imgH; py += pxStep {
		for px := 0; px < imgW; px += pxStep {
			lat, lon := pxToLatLon(px, py)
			col := overviewGrayColor(sampleMergedTheaterDepth(lat, lon, theaters, offChartDepth))
			for dy := 0; dy < pxStep; dy++ {
				for dx := 0; dx < pxStep; dx++ {
					x, y := px+dx, py+dy
					if image.Pt(x, y).In(img.Bounds()) {
						img.Set(x, y, col)
					}
				}
			}
		}
	}
}

func drawBathyInset(img *image.RGBA, bathy *world.Bathymetry, rect image.Rectangle) {
	if bathy == nil || !bathy.Valid() {
		return
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	w := rect.Dx()
	h := rect.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	stepX := int(math.Max(1, math.Ceil(float64(w)/80)))
	stepY := int(math.Max(1, math.Ceil(float64(h)/80)))
	for py := 0; py < h; py += stepY {
		for px := 0; px < w; px += stepX {
			wx := minX + (maxX-minX)*float64(px)/float64(w)
			wy := maxY - (maxY-minY)*float64(py)/float64(h)
			col := depthColor(bathy.DepthAtFt(wx, wy))
			for dy := 0; dy < stepY; dy++ {
				for dx := 0; dx < stepX; dx++ {
					x, y := rect.Min.X+px+dx, rect.Min.Y+py+dy
					if image.Pt(x, y).In(rect) {
						img.Set(x, y, col)
					}
				}
			}
		}
	}
}

func drawRectBorder(img *image.RGBA, rect image.Rectangle, c color.RGBA, thick int) {
	for t := 0; t < thick; t++ {
		r := rect.Inset(-t)
		drawHLine(img, r.Min.X, r.Max.X-1, r.Min.Y, c)
		drawHLine(img, r.Min.X, r.Max.X-1, r.Max.Y-1, c)
		drawVLine(img, r.Min.X, r.Min.Y, r.Max.Y-1, c)
		drawVLine(img, r.Max.X-1, r.Min.Y, r.Max.Y-1, c)
	}
}

func drawLatLonGrid(img *image.RGBA, latMin, latMax, lonMin, lonMax float64, headerH, imgW, mapH int, latLonToPx func(float64, float64) (int, int)) {
	gridCol := color.RGBA{90, 110, 130, 120}
	labelCol := color.RGBA{150, 170, 190, 255}
	latStep := 0.5
	if latMax-latMin > 6 {
		latStep = 1.0
	}
	lonStep := 0.5
	if lonMax-lonMin > 6 {
		lonStep = 1.0
	}
	for lat := math.Ceil(latMin/latStep) * latStep; lat <= latMax; lat += latStep {
		_, y := latLonToPx(lat, lonMin)
		drawHLine(img, 20, imgW-20, y, gridCol)
		drawString(img, 4, y-3, fmt.Sprintf("%.0fN", lat), labelCol)
	}
	for lon := math.Ceil(lonMin/lonStep) * lonStep; lon <= lonMax; lon += lonStep {
		x, _ := latLonToPx(latMax, lon)
		drawVLine(img, x, headerH+16, headerH+mapH-8, gridCol)
		drawString(img, x-10, headerH+mapH-6, fmt.Sprintf("%.0fE", lon), labelCol)
	}
}

func shortenLabel(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func writePNG(path string, img *image.RGBA) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func subtitleFor(bathy *world.Bathymetry, routes []routeDraw) string {
	landPct := landFraction(bathy) * 100
	return fmt.Sprintf("routes %d | land %.1f%% | ★ = unit spawn | cyan = player | red = shore hit", len(routes), landPct)
}

type playerSpawn struct {
	spawn      string
	corner     string
	insetYd    float64
	minRouteYd float64
	maxRouteYd float64
	depthFt    float64
	routeID    string
	routeFrac  float64
}

type unitSpawn struct {
	id         string
	side       string
	spawn      string
	corner     string
	insetYd    float64
	minRouteYd float64
	maxRouteYd float64
	routeID    string
	routeFrac  float64
	depthFt    float64
	kind       world.EntityKind
}

func parsePlayerSpawn(m map[string]any) *playerSpawn {
	p, ok := m["player"].(map[string]any)
	if !ok || p == nil {
		return nil
	}
	spawn, _ := p["spawn"].(string)
	ps := &playerSpawn{
		spawn:      spawn,
		corner:     "SW",
		insetYd:    1800,
		minRouteYd: 600,
	}
	if spawn == "route" {
		ps.routeID, _ = p["route_id"].(string)
		if v, ok := p["route_frac"].(float64); ok {
			ps.routeFrac = v
		}
	} else if spawn != "" && spawn != "corner" {
		return nil
	}
	if c, ok := p["corner"].(string); ok && c != "" {
		ps.corner = strings.ToUpper(c)
	}
	if v, ok := p["corner_inset_yd"].(float64); ok && v > 0 {
		ps.insetYd = v
	}
	if v, ok := p["min_route_yd"].(float64); ok && v > 0 {
		ps.minRouteYd = v
	}
	if v, ok := p["max_route_yd"].(float64); ok {
		ps.maxRouteYd = v
	}
	if v, ok := p["depth_ft"].(float64); ok && v > 0 {
		ps.depthFt = v
	} else {
		ps.depthFt = 120
	}
	return ps
}

func parseUnitSpawns(m map[string]any) []unitSpawn {
	var out []unitSpawn
	if p, ok := m["player"].(map[string]any); ok && p != nil {
		out = append(out, unitSpawnFromJSON("player", p))
	}
	for _, uraw := range m["units"].([]any) {
		u, ok := uraw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := u["id"].(string)
		if id == "" {
			continue
		}
		out = append(out, unitSpawnFromJSON(id, u))
	}
	return out
}

func unitSpawnFromJSON(id string, u map[string]any) unitSpawn {
	us := unitSpawn{
		id:      id,
		side:    "player",
		spawn:   "corner",
		corner:  "SW",
		insetYd: 1800,
	}
	if s, ok := u["side"].(string); ok && s != "" {
		us.side = s
	}
	if spawn, ok := u["spawn"].(string); ok && spawn != "" {
		us.spawn = spawn
	}
	if c, ok := u["corner"].(string); ok && c != "" {
		us.corner = strings.ToUpper(c)
	}
	if v, ok := u["corner_inset_yd"].(float64); ok && v > 0 {
		us.insetYd = v
	}
	if v, ok := u["min_route_yd"].(float64); ok && v > 0 {
		us.minRouteYd = v
	}
	if v, ok := u["max_route_yd"].(float64); ok {
		us.maxRouteYd = v
	}
	if us.minRouteYd == 0 {
		us.minRouteYd = 600
	}
	if v, ok := u["route_frac"].(float64); ok {
		us.routeFrac = v
	}
	us.routeID, _ = u["route_id"].(string)
	if v, ok := u["depth_ft"].(float64); ok && v > 0 {
		us.depthFt = v
	} else {
		us.depthFt = 120
	}
	switch u["kind"] {
	case "submarine":
		us.kind = world.KindSubmarine
	default:
		us.kind = world.KindSurfaceShip
	}
	return us
}

func playerSpawnCenter(bathy *world.Bathymetry, ps *playerSpawn) (float64, float64) {
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := ps.insetYd
	tx, ty := minX+m, minY+m
	switch ps.corner {
	case "NW":
		tx, ty = minX+m, maxY-m
	case "NE":
		tx, ty = maxX-m, maxY-m
	case "SE":
		tx, ty = maxX-m, minY+m
	}
	return tx, ty
}

func landFraction(bathy *world.Bathymetry) float64 {
	if bathy == nil || !bathy.Valid() {
		return 0
	}
	land := 0
	for j := 0; j < bathy.Height; j++ {
		for i := 0; i < bathy.Width; i++ {
			cx := bathy.OriginX + (float64(i)+0.5)*bathy.CellSize
			cy := bathy.OriginY + (float64(j)+0.5)*bathy.CellSize
			if bathy.DepthAtFt(cx, cy) <= 0 {
				land++
			}
		}
	}
	return float64(land) / float64(bathy.Width*bathy.Height)
}

func loadTheater(th map[string]any) *world.Bathymetry {
	b := th["bathy"].(map[string]any)
	raw, _ := base64.StdEncoding.DecodeString(b["data_b64"].(string))
	bathy, err := world.LoadBathymetry(raw)
	if err != nil {
		panic(err)
	}
	return &bathy
}

type routeDraw struct {
	id              string
	mode            string
	wps             []world.Waypoint
	playerClearance bool
}

// engagementZone marks a predicted weapons-release area (route CPA + range analysis).
type engagementZone struct {
	x, y, radiusYd float64
	label            string
}

// twAttributionEngagementZones — DEFCON 3, patrol speeds, weapon envelopes (Aug 2026 routes).
var twAttributionEngagementZones = []engagementZone{
	{x: 10820, y: -7860, radiusYd: 1800, label: "1 SE X-ing ~28m"},
	{x: 7680, y: -11020, radiusYd: 2200, label: "2 W detect ~32m"},
	{x: 9750, y: -7630, radiusYd: 2800, label: "3 688 vs escorts ~44m"},
	{x: 11320, y: -6510, radiusYd: 1600, label: "4 688 merge ~56m"},
	{x: 11590, y: -7900, radiusYd: 2200, label: "5 Kilo ~61m"},
}

func parseRoutes(raw []any) []routeDraw {
	out := make([]routeDraw, 0, len(raw))
	for _, rraw := range raw {
		rm := rraw.(map[string]any)
		id, _ := rm["id"].(string)
		mode, _ := rm["mode"].(string)
		pc, _ := rm["player_clearance"].(bool)
		var wps []world.Waypoint
		for _, wraw := range rm["waypoints"].([]any) {
			wm := wraw.(map[string]any)
			wps = append(wps, world.Waypoint{X: wm["x"].(float64), Y: wm["y"].(float64)})
		}
		out = append(out, routeDraw{id: id, mode: mode, wps: wps, playerClearance: pc})
	}
	return out
}

func routeLabelsFromUnits(m map[string]any) map[string]string {
	labels := map[string]string{}
	units, _ := m["units"].([]any)
	for _, uraw := range units {
		u, ok := uraw.(map[string]any)
		if !ok {
			continue
		}
		rid, _ := u["route_id"].(string)
		uid, _ := u["id"].(string)
		if rid == "" || uid == "" {
			continue
		}
		if prev, ok := labels[rid]; ok {
			if len(prev)+len(uid)+1 < 24 {
				labels[rid] = prev + "+" + uid
			}
			continue
		}
		labels[rid] = uid
	}
	for _, rraw := range m["routes"].([]any) {
		rm, _ := rraw.(map[string]any)
		rid, _ := rm["id"].(string)
		if rid == "" || labels[rid] != "" {
			continue
		}
		labels[rid] = strings.TrimPrefix(rid, "route_")
	}
	return labels
}

func renderPreview(bathy *world.Bathymetry, routes []routeDraw, labels map[string]string, spawns []unitSpawn, title, subtitle, path string, zones []engagementZone) map[string]any {
	const imgW, imgH = 900, 940
	const headerH = 40
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{12, 14, 18, 255}}, image.Point{}, draw.Src)
	drawHeader(img, title, subtitle)

	minX, minY, maxX, maxY := bathy.BoundsYards()
	mapH := imgH - headerH
	for j := 0; j < bathy.Height; j += 2 {
		for i := 0; i < bathy.Width; i += 2 {
			cx := bathy.OriginX + (float64(i)+0.5)*bathy.CellSize
			cy := bathy.OriginY + (float64(j)+0.5)*bathy.CellSize
			d := bathy.DepthAtFt(cx, cy)
			col := depthColor(d)
			sx, sy := worldToPx(cx, cy, minX, minY, maxX, maxY, imgW, mapH, headerH)
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					if sx+dx < imgW && sy+dy < imgH {
						img.Set(sx+dx, sy+dy, col)
					}
				}
			}
		}
	}

	drawCoastline(img, bathy, minX, minY, maxX, maxY, imgW, mapH, headerH)
	if len(zones) > 0 {
		drawEngagementZones(img, zones, minX, minY, maxX, maxY, imgW, mapH, headerH)
	}

	totalHits := 0
	routeStats := make([]map[string]any, 0, len(routes))
	for ri, r := range routes {
		col := routeColors[ri%len(routeColors)]
		pts := make([]image.Point, len(r.wps))
		for i, wp := range r.wps {
			px, py := worldToPx(wp.X, wp.Y, minX, minY, maxX, maxY, imgW, mapH, headerH)
			pts[i] = image.Point{X: px, Y: py}
		}
		routeHits := 0
		for i := 1; i < len(r.wps); i++ {
			drawLine(img, pts[i-1], pts[i], col)
			hits := shoreHitPoints(bathy, r.wps[i-1], r.wps[i])
			routeHits += len(hits)
			for _, hp := range hits {
				sx, sy := worldToPx(hp.X, hp.Y, minX, minY, maxX, maxY, imgW, mapH, headerH)
				drawHitMarker(img, sx, sy)
			}
		}
		if r.mode == "pingpong" && len(r.wps) >= 2 {
			hits := shoreHitPoints(bathy, r.wps[len(r.wps)-1], r.wps[len(r.wps)-2])
			routeHits += len(hits)
			for _, hp := range hits {
				sx, sy := worldToPx(hp.X, hp.Y, minX, minY, maxX, maxY, imgW, mapH, headerH)
				drawHitMarker(img, sx, sy)
			}
		}
		if r.mode == "loop" && len(r.wps) >= 2 {
			drawLine(img, pts[len(pts)-1], pts[0], col)
			hits := shoreHitPoints(bathy, r.wps[len(r.wps)-1], r.wps[0])
			routeHits += len(hits)
			for _, hp := range hits {
				sx, sy := worldToPx(hp.X, hp.Y, minX, minY, maxX, maxY, imgW, mapH, headerH)
				drawHitMarker(img, sx, sy)
			}
		}
		totalHits += routeHits
		routeStats = append(routeStats, map[string]any{"id": r.id, "hits": routeHits, "wps": len(r.wps)})
		for _, p := range pts {
			rect := image.Rect(p.X-2, p.Y-2, p.X+3, p.Y+3)
			draw.Draw(img, rect, &image.Uniform{col}, image.Point{}, draw.Src)
		}
		label := labels[r.id]
		if label == "" {
			label = strings.TrimPrefix(r.id, "route_")
		}
		if len(label) > 18 {
			label = label[:18]
		}
		if len(pts) > 0 {
			lx, ly := pts[0].X+4, pts[0].Y-10
			if ly < headerH+2 {
				ly = pts[0].Y + 6
			}
			drawLabel(img, lx, ly, label, col)
		}
	}

	if len(spawns) > 0 {
		drawUnitSpawns(img, bathy, spawns, routes, minX, minY, maxX, maxY, imgW, mapH, headerH)
		drawPlayerSpawnCircle(img, bathy, spawns, routes, minX, minY, maxX, maxY, imgW, mapH, headerH)
	}
	if len(zones) > 0 {
		drawEngagementZoneLabels(img, zones, minX, minY, maxX, maxY, imgW, mapH, headerH)
	}

	drawScaleBar(img, minX, minY, maxX, maxY, imgW, mapH, headerH)

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	return map[string]any{
		"land_pct": landFraction(bathy) * 100,
		"hits":     totalHits,
		"routes":   routeStats,
	}
}

// renderCoastTemplate writes a blank chart (ocean + shoreline) for hand-drawn route markup.
func renderCoastTemplate(bathy *world.Bathymetry, title, path string) {
	const imgW, imgH = 900, 940
	const headerH = 40
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	ocean := color.RGBA{14, 22, 34, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{ocean}, image.Point{}, draw.Src)
	drawHeader(img, title, "coast template — draw routes here | yards | same scale as mission preview")

	minX, minY, maxX, maxY := bathy.BoundsYards()
	mapH := imgH - headerH
	drawCoastline(img, bathy, minX, minY, maxX, maxY, imgW, mapH, headerH)
	drawWorldCornerLabels(img, minX, minY, maxX, maxY, imgW, mapH, headerH)
	drawScaleBar(img, minX, minY, maxX, maxY, imgW, mapH, headerH)

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func drawWorldCornerLabels(img *image.RGBA, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	col := color.RGBA{120, 150, 180, 255}
	drawString(img, 10, headerH+6, fmt.Sprintf("NW %.0f,%.0f", minX, maxY), col)
	drawString(img, imgW-170, headerH+6, fmt.Sprintf("NE %.0f,%.0f", maxX, maxY), col)
	drawString(img, 10, headerH+mapH-18, fmt.Sprintf("SW %.0f,%.0f", minX, minY), col)
	drawString(img, imgW-170, headerH+mapH-18, fmt.Sprintf("SE %.0f,%.0f", maxX, minY), col)
}

// drawScaleBar draws a true-scale bar: full width = 1 nm with a 1000 yd tick.
func drawScaleBar(img *image.RGBA, minX, _, maxX, _ float64, imgW, mapH, headerH int) {
	const pad = 8.0
	worldSpan := maxX - minX
	if worldSpan <= 0 {
		return
	}
	pxPerYd := (float64(imgW) - 2*pad) / worldSpan
	nmLen := int(math.Round(world.YardsPerNM * pxPerYd))
	yd1000 := int(math.Round(1000 * pxPerYd))
	if nmLen < 8 {
		nmLen = 8
	}
	if yd1000 < 1 {
		yd1000 = 1
	}
	if yd1000 > nmLen-2 {
		yd1000 = nmLen - 2
	}

	marginX := 12
	barY := headerH + mapH - 14
	x0 := marginX
	x1 := x0 + nmLen
	tick1000 := x0 + yd1000

	barCol := color.RGBA{235, 240, 248, 255}
	dimCol := color.RGBA{170, 180, 195, 255}

	// Semi-transparent backing for labels + bar.
	bg := image.Rect(x0-4, barY-16, x1+44, barY+5)
	draw.Draw(img, bg, &image.Uniform{color.RGBA{8, 10, 14, 210}}, image.Point{}, draw.Src)

	drawHLine(img, x0, x1, barY, barCol)
	drawVLine(img, x0, barY-4, barY+4, barCol)
	drawVLine(img, tick1000, barY-3, barY+3, dimCol)
	drawVLine(img, x1, barY-4, barY+4, barCol)

	drawString(img, x0, barY-14, "1000 yd", dimCol)
	drawString(img, x1-18, barY-14, "1 nm", barCol)
}

func drawHLine(img *image.RGBA, x0, x1, y int, c color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	for x := x0; x <= x1; x++ {
		if image.Pt(x, y).In(img.Bounds()) {
			img.Set(x, y, c)
		}
	}
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.RGBA) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		if image.Pt(x, y).In(img.Bounds()) {
			img.Set(x, y, c)
		}
	}
}

func drawLabel(img *image.RGBA, x0, y0 int, text string, c color.RGBA) {
	if text == "" {
		return
	}
	w := len(text)*6 + 4
	h := 9
	bg := image.Rect(x0-2, y0-1, x0+w, y0+h)
	draw.Draw(img, bg, &image.Uniform{color.RGBA{8, 10, 14, 220}}, image.Point{}, draw.Src)
	drawString(img, x0, y0, text, c)
}

type coastSegment struct {
	x0, y0, x1, y1 float64
}

func drawUnitSpawns(img *image.RGBA, bathy *world.Bathymetry, spawns []unitSpawn, routes []routeDraw, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	for _, us := range spawns {
		if us.id == "player" {
			continue
		}
		x, y, ok := placeUnitSpawn(bathy, us, routes)
		if !ok {
			continue
		}
		col := spawnStarColor(us.id, us.side)
		drawSpawnStar(img, x, y, minX, minY, maxX, maxY, imgW, mapH, headerH, col)
		px, py := worldToPx(x, y, minX, minY, maxX, maxY, imgW, mapH, headerH)
		label := us.id
		if len(label) > 14 {
			label = label[:14]
		}
		drawLabel(img, px+8, py-6, label, col)
	}
}

func drawPlayerSpawnCircle(img *image.RGBA, bathy *world.Bathymetry, spawns []unitSpawn, routes []routeDraw, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	for _, us := range spawns {
		if us.id != "player" {
			continue
		}
		x, y, ok := placeUnitSpawn(bathy, us, routes)
		if !ok {
			continue
		}
		label := "PLAYER " + us.corner
		if us.spawn == "route" {
			label = "PLAYER EX"
		}
		spawnCol := color.RGBA{80, 220, 255, 255}
		fillCol := color.RGBA{80, 220, 255, 40}
		drawWorldCircle(img, x, y, 500, minX, minY, maxX, maxY, imgW, mapH, headerH, fillCol, true)
		drawWorldCircle(img, x, y, 500, minX, minY, maxX, maxY, imgW, mapH, headerH, spawnCol, false)
		px, py := worldToPx(x, y, minX, minY, maxX, maxY, imgW, mapH, headerH)
		drawLabel(img, px+6, py-4, label, spawnCol)
		return
	}
}

func placeUnitSpawn(bathy *world.Bathymetry, us unitSpawn, routes []routeDraw) (x, y float64, ok bool) {
	routeByID := map[string]*world.Route{}
	for _, r := range routes {
		if len(r.wps) < 1 {
			continue
		}
		routeByID[r.id] = &world.Route{
			ID: r.id, Waypoints: r.wps, PingPong: r.mode == "pingpong",
		}
	}
	clearance := make([]*world.Route, 0)
	for _, r := range routes {
		if !r.playerClearance || len(r.wps) < 2 {
			continue
		}
		clearance = append(clearance, &world.Route{
			ID: r.id, Waypoints: r.wps, PingPong: r.mode == "pingpong",
		})
	}
	ent := &world.Entity{Kind: us.kind, DepthFt: us.depthFt}
	if us.spawn == "route" && us.routeID != "" {
		if route := routeByID[us.routeID]; route != nil {
			if world.PlaceOnRouteFraction(ent, route, us.routeFrac, bathy) {
				return ent.X, ent.Y, true
			}
			// Preview: show intended route waypoint even when depth/bathy rejects placement.
			if wx, wy, ok2 := routeWaypointAtFraction(route, us.routeFrac); ok2 {
				return wx, wy, true
			}
		}
	}
	if us.spawn == "corner" {
		if world.PlaceNearChartCorner(ent, bathy, us.corner, clearance, us.minRouteYd, us.maxRouteYd, us.insetYd) {
			return ent.X, ent.Y, true
		}
		return chartCornerCenter(bathy, us.corner, us.insetYd)
	}
	return 0, 0, false
}

func routeWaypointAtFraction(r *world.Route, t float64) (x, y float64, ok bool) {
	if r == nil {
		return 0, 0, false
	}
	n := r.UniqueCount()
	if n == 0 {
		return 0, 0, false
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	idx := int(t*float64(n-1) + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	wp := r.Waypoints[idx]
	return wp.X, wp.Y, true
}

func chartCornerCenter(bathy *world.Bathymetry, corner string, insetYd float64) (float64, float64, bool) {
	if bathy == nil || !bathy.Valid() {
		return 0, 0, false
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := insetYd
	if m <= 0 {
		m = 1800
	}
	switch strings.ToUpper(corner) {
	case "NW":
		return minX + m, maxY - m, true
	case "NE":
		return maxX - m, maxY - m, true
	case "SE":
		return maxX - m, minY + m, true
	default:
		return minX + m, minY + m, true
	}
}

func spawnStarColor(id, side string) color.RGBA {
	if id == "player" {
		return color.RGBA{80, 220, 255, 255}
	}
	switch side {
	case "enemy":
		return color.RGBA{255, 90, 70, 255}
	case "neutral":
		return color.RGBA{120, 220, 120, 255}
	default:
		return color.RGBA{255, 220, 80, 255}
	}
}

func drawSpawnStar(img *image.RGBA, wx, wy, minX, minY, maxX, maxY float64, imgW, mapH, headerH int, col color.RGBA) {
	cx, cy := worldToPx(wx, wy, minX, minY, maxX, maxY, imgW, mapH, headerH)
	outline := color.RGBA{12, 14, 18, 255}
	const outerR, innerR = 9, 4
	drawStarAt(img, cx, cy, outerR+1, innerR+1, outline)
	drawStarAt(img, cx, cy, outerR, innerR, col)
}

func drawStarAt(img *image.RGBA, cx, cy, outerR, innerR int, col color.RGBA) {
	for i := 0; i < 10; i++ {
		ang := float64(i)*math.Pi/5 - math.Pi/2
		r := float64(outerR)
		if i%2 == 1 {
			r = float64(innerR)
		}
		x := cx + int(math.Round(math.Cos(ang)*r))
		y := cy + int(math.Round(math.Sin(ang)*r))
		j := (i + 1) % 10
		ang2 := float64(j)*math.Pi/5 - math.Pi/2
		r2 := float64(outerR)
		if j%2 == 1 {
			r2 = float64(innerR)
		}
		x2 := cx + int(math.Round(math.Cos(ang2)*r2))
		y2 := cy + int(math.Round(math.Sin(ang2)*r2))
		drawLine(img, image.Point{X: x, Y: y}, image.Point{X: x2, Y: y2}, col)
	}
	if image.Pt(cx, cy).In(img.Bounds()) {
		img.Set(cx, cy, col)
	}
}

func drawPlayerSpawn(img *image.RGBA, bathy *world.Bathymetry, ps *playerSpawn, routes []routeDraw, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	player := &world.Entity{Kind: world.KindSubmarine, DepthFt: ps.depthFt}
	label := "PLAYER " + ps.corner
	placed := false
	if ps.spawn == "route" && ps.routeID != "" {
		for _, r := range routes {
			if r.id != ps.routeID || len(r.wps) < 2 {
				continue
			}
			route := &world.Route{ID: r.id, Waypoints: r.wps, PingPong: r.mode == "pingpong"}
			if world.PlaceOnRouteFraction(player, route, ps.routeFrac, bathy) {
				placed = true
				label = "PLAYER EX"
				break
			}
		}
	}
	if !placed {
		clearance := make([]*world.Route, 0)
		for _, r := range routes {
			if !r.playerClearance || len(r.wps) < 2 {
				continue
			}
			clearance = append(clearance, &world.Route{
				ID: r.id, Waypoints: r.wps, PingPong: r.mode == "pingpong",
			})
		}
		if !world.PlaceNearChartCorner(player, bathy, ps.corner, clearance, ps.minRouteYd, ps.maxRouteYd, ps.insetYd) {
			cx, cy := playerSpawnCenter(bathy, ps)
			player.X, player.Y = cx, cy
		}
	}
	spawnCol := color.RGBA{80, 220, 255, 255}
	fillCol := color.RGBA{80, 220, 255, 40}
	drawWorldCircle(img, player.X, player.Y, 500, minX, minY, maxX, maxY, imgW, mapH, headerH, fillCol, true)
	drawWorldCircle(img, player.X, player.Y, 500, minX, minY, maxX, maxY, imgW, mapH, headerH, spawnCol, false)
	px, py := worldToPx(player.X, player.Y, minX, minY, maxX, maxY, imgW, mapH, headerH)
	drawLabel(img, px+6, py-4, label, spawnCol)
}

func drawWorldCircle(img *image.RGBA, cx, cy, radiusYd, minX, minY, maxX, maxY float64, imgW, mapH, headerH int, col color.RGBA, fill bool) {
	const steps = 180
	pts := make([]image.Point, 0, steps)
	for a := 0; a < steps; a++ {
		rad := float64(a) * 2 * math.Pi / float64(steps)
		wx := cx + math.Sin(rad)*radiusYd
		wy := cy + math.Cos(rad)*radiusYd
		px, py := worldToPx(wx, wy, minX, minY, maxX, maxY, imgW, mapH, headerH)
		pts = append(pts, image.Point{X: px, Y: py})
	}
	if fill && len(pts) >= 3 {
		fillPolygon(img, pts, col)
	}
	for i := range pts {
		j := (i + 1) % len(pts)
		drawLine(img, pts[i], pts[j], col)
	}
}

func fillPolygon(img *image.RGBA, pts []image.Point, col color.RGBA) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := range pts {
			j := (i + 1) % len(pts)
			y0, y1 := pts[i].Y, pts[j].Y
			if y0 == y1 {
				continue
			}
			if (y < int(math.Min(float64(y0), float64(y1)))) || (y >= int(math.Max(float64(y0), float64(y1)))) {
				continue
			}
			t := (float64(y) - float64(y0)) / float64(y1-y0)
			x := int(float64(pts[i].X) + t*float64(pts[j].X-pts[i].X))
			xs = append(xs, x)
		}
		if len(xs) < 2 {
			continue
		}
		for i := 0; i < len(xs)-1; i++ {
			for j := i + 1; j < len(xs); j++ {
				if xs[j] < xs[i] {
					xs[i], xs[j] = xs[j], xs[i]
				}
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			x0, x1 := xs[i], xs[i+1]
			if x0 > x1 {
				x0, x1 = x1, x0
			}
			for x := x0; x <= x1; x++ {
				if image.Pt(x, y).In(img.Bounds()) {
					img.Set(x, y, col)
				}
			}
		}
	}
}

func drawCoastline(img *image.RGBA, bathy *world.Bathymetry, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	coastCol := color.RGBA{210, 200, 150, 255}
	for _, seg := range buildCoastSegments(bathy) {
		a := worldToPxPt(seg.x0, seg.y0, minX, minY, maxX, maxY, imgW, mapH, headerH)
		b := worldToPxPt(seg.x1, seg.y1, minX, minY, maxX, maxY, imgW, mapH, headerH)
		drawLine(img, a, b, coastCol)
	}
}

func buildCoastSegments(bathy *world.Bathymetry) []coastSegment {
	if bathy == nil || !bathy.Valid() || bathy.Width < 2 || bathy.Height < 2 {
		return nil
	}
	segments := make([]coastSegment, 0, bathy.Width*bathy.Height/4)
	for j := 0; j < bathy.Height-1; j++ {
		y0 := bathy.OriginY + (float64(j)+0.5)*bathy.CellSize
		y1 := bathy.OriginY + (float64(j+1)+0.5)*bathy.CellSize
		for i := 0; i < bathy.Width-1; i++ {
			x0 := bathy.OriginX + (float64(i)+0.5)*bathy.CellSize
			x1 := bathy.OriginX + (float64(i+1)+0.5)*bathy.CellSize
			dBL := float64(bathy.Depths[j*bathy.Width+i])
			dBR := float64(bathy.Depths[j*bathy.Width+i+1])
			dTL := float64(bathy.Depths[(j+1)*bathy.Width+i])
			dTR := float64(bathy.Depths[(j+1)*bathy.Width+i+1])
			mask := 0
			if dBL > 0 {
				mask |= 1
			}
			if dBR > 0 {
				mask |= 2
			}
			if dTR > 0 {
				mask |= 4
			}
			if dTL > 0 {
				mask |= 8
			}
			if mask == 0 || mask == 15 {
				continue
			}
			bottomX, bottomY := interpZero(x0, y0, dBL, x1, y0, dBR)
			rightX, rightY := interpZero(x1, y0, dBR, x1, y1, dTR)
			topX, topY := interpZero(x0, y1, dTL, x1, y1, dTR)
			leftX, leftY := interpZero(x0, y0, dBL, x0, y1, dTL)
			switch mask {
			case 1, 14:
				segments = append(segments, coastSegment{leftX, leftY, bottomX, bottomY})
			case 2, 13:
				segments = append(segments, coastSegment{bottomX, bottomY, rightX, rightY})
			case 3, 12:
				segments = append(segments, coastSegment{leftX, leftY, rightX, rightY})
			case 4, 11:
				segments = append(segments, coastSegment{rightX, rightY, topX, topY})
			case 5:
				segments = append(segments,
					coastSegment{leftX, leftY, topX, topY},
					coastSegment{bottomX, bottomY, rightX, rightY},
				)
			case 6, 9:
				segments = append(segments, coastSegment{bottomX, bottomY, topX, topY})
			case 7, 8:
				segments = append(segments, coastSegment{leftX, leftY, topX, topY})
			case 10:
				segments = append(segments,
					coastSegment{leftX, leftY, bottomX, bottomY},
					coastSegment{rightX, rightY, topX, topY},
				)
			}
		}
	}
	return segments
}

func interpZero(x0, y0, d0, x1, y1, d1 float64) (float64, float64) {
	den := d0 - d1
	if math.Abs(den) < 1e-6 {
		return (x0 + x1) * 0.5, (y0 + y1) * 0.5
	}
	t := d0 / den
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return x0 + (x1-x0)*t, y0 + (y1-y0)*t
}

func worldToPxPt(x, y, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) image.Point {
	px, py := worldToPx(x, y, minX, minY, maxX, maxY, imgW, mapH, headerH)
	return image.Point{X: px, Y: py}
}

func drawHeader(img *image.RGBA, title, subtitle string) {
	hdr := image.Rect(0, 0, img.Bounds().Dx(), 40)
	draw.Draw(img, hdr, &image.Uniform{color.RGBA{20, 24, 30, 255}}, image.Point{}, draw.Src)
	drawString(img, 8, 6, title, color.RGBA{220, 230, 240, 255})
	drawString(img, 8, 22, subtitle, color.RGBA{140, 160, 180, 255})
}

// Minimal 5x7 bitmap font for ASCII preview headers.
func drawString(img *image.RGBA, x0, y0 int, s string, c color.RGBA) {
	x := x0
	for _, ch := range s {
		glyph, ok := ascii5x7[ch]
		if !ok {
			x += 6
			continue
		}
		for row, bits := range glyph {
			for col := 0; col < 5; col++ {
				if bits&(1<<uint(4-col)) != 0 {
					img.Set(x+col, y0+row, c)
				}
			}
		}
		x += 6
	}
}

func drawHitMarker(img *image.RGBA, x, y int) {
	rect := image.Rect(x-3, y-3, x+4, y+4)
	draw.Draw(img, rect, &image.Uniform{color.RGBA{255, 40, 40, 255}}, image.Point{}, draw.Src)
}

func drawEngagementZones(img *image.RGBA, zones []engagementZone, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	fillCol := color.RGBA{255, 140, 40, 48}
	strokeCol := color.RGBA{255, 140, 40, 110}
	for _, z := range zones {
		cx, cy := worldToPx(z.x, z.y, minX, minY, maxX, maxY, imgW, mapH, headerH)
		const pad = 8.0
		pxPerYd := (float64(imgW) - 2*pad) / (maxX - minX)
		rPx := int(z.radiusYd * pxPerYd)
		if rPx < 6 {
			rPx = 6
		}
		drawCircle(img, cx, cy, rPx, fillCol, strokeCol)
	}
}

func drawEngagementZoneLabels(img *image.RGBA, zones []engagementZone, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) {
	zoneCol := color.RGBA{255, 180, 100, 255}
	for _, z := range zones {
		cx, cy := worldToPx(z.x, z.y, minX, minY, maxX, maxY, imgW, mapH, headerH)
		const pad = 8.0
		pxPerYd := (float64(imgW) - 2*pad) / (maxX - minX)
		rPx := int(z.radiusYd * pxPerYd)
		if rPx < 6 {
			rPx = 6
		}
		lx, ly := cx+rPx+4, cy-4
		if lx > imgW-120 {
			lx = cx - rPx - 80
		}
		if ly < headerH+4 {
			ly = cy + rPx + 10
		}
		drawLabel(img, lx, ly, z.label, zoneCol)
	}
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	if c.A == 255 {
		img.Set(x, y, c)
		return
	}
	dst := img.RGBAAt(x, y)
	a := float64(c.A) / 255
	inv := 1 - a
	img.Set(x, y, color.RGBA{
		R: uint8(float64(c.R)*a + float64(dst.R)*inv),
		G: uint8(float64(c.G)*a + float64(dst.G)*inv),
		B: uint8(float64(c.B)*a + float64(dst.B)*inv),
		A: 255,
	})
}

func drawCircle(img *image.RGBA, cx, cy, r int, fill, stroke color.RGBA) {
	if r < 1 {
		return
	}
	r2 := r * r
	inner := (r - 1) * (r - 1)
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			d2 := dx*dx + dy*dy
			if d2 > r2 {
				continue
			}
			x, y := cx+dx, cy+dy
			if d2 >= inner {
				blendPixel(img, x, y, stroke)
			} else {
				blendPixel(img, x, y, fill)
			}
		}
	}
}

func shoreHitPoints(bathy *world.Bathymetry, a, b world.Waypoint) []world.Waypoint {
	var hits []world.Waypoint
	dist := math.Hypot(b.X-a.X, b.Y-a.Y)
	steps := int(dist/200) + 1
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		x := a.X + (b.X-a.X)*t
		y := a.Y + (b.Y-a.Y)*t
		if bathy.IsShoreBlocked(x, y) {
			hits = append(hits, world.Waypoint{X: x, Y: y})
		}
	}
	return hits
}

func worldToPx(x, y, minX, minY, maxX, maxY float64, imgW, mapH, headerH int) (int, int) {
	const pad = 8.0
	sx := pad + (x-minX)/(maxX-minX)*(float64(imgW)-2*pad)
	sy := float64(headerH) + pad + (maxY-y)/(maxY-minY)*(float64(mapH)-2*pad)
	return int(sx), int(sy)
}

func depthColor(d float64) color.RGBA {
	if d <= 0 {
		return color.RGBA{48, 72, 40, 255}
	}
	depth := math.Min(d, 6000)
	t := math.Log1p(depth) / math.Log1p(6000)
	return color.RGBA{
		R: uint8(30 + (1-t)*90),
		G: uint8(60 + (1-t)*100),
		B: uint8(40 + t*180),
		A: 255,
	}
}

func drawLine(img *image.RGBA, a, b image.Point, c color.RGBA) {
	dx := math.Abs(float64(b.X - a.X))
	dy := math.Abs(float64(b.Y - a.Y))
	steps := int(math.Max(dx, dy))
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(float64(a.X) + float64(b.X-a.X)*t)
		y := int(float64(a.Y) + float64(b.Y-a.Y)*t)
		if image.Pt(x, y).In(img.Bounds()) {
			img.Set(x, y, c)
		}
	}
}

// ascii5x7 is a tiny subset for headers (A-Z a-z 0-9 punctuation).
var ascii5x7 = map[rune][7]byte{
	' ': {0, 0, 0, 0, 0, 0, 0},
	'_': {0, 0, 0, 0, 0, 0, 0x1F},
	'-': {0, 0, 0, 0x1F, 0, 0, 0},
	'.': {0, 0, 0, 0, 0, 0x0C, 0x0C},
	'/': {0x01, 0x02, 0x04, 0x08, 0x10, 0, 0},
	':': {0, 0x0C, 0, 0, 0x0C, 0, 0},
	'|': {0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'+': {0, 0x04, 0x04, 0x1F, 0x04, 0x04, 0},
	'q': {0, 0, 0x0D, 0x13, 0x0D, 0x01, 0x0E},
	'0': {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2': {0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F},
	'3': {0x1F, 0x02, 0x04, 0x02, 0x01, 0x11, 0x0E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	'A': {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1E},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0F},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'I': {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M': {0x11, 0x1B, 0x15, 0x11, 0x11, 0x11, 0x11},
	'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	'O': {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0F, 0x10, 0x10, 0x0E, 0x01, 0x01, 0x1E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'a': {0, 0, 0x0E, 0x01, 0x0F, 0x11, 0x0F},
	'b': {0x10, 0x10, 0x16, 0x19, 0x11, 0x11, 0x1E},
	'c': {0, 0, 0x0E, 0x10, 0x10, 0x11, 0x0E},
	'd': {0x01, 0x01, 0x0D, 0x13, 0x11, 0x11, 0x0F},
	'e': {0, 0, 0x0E, 0x11, 0x1F, 0x10, 0x0E},
	'f': {0x06, 0x09, 0x08, 0x1C, 0x08, 0x08, 0x08},
	'g': {0, 0, 0x0F, 0x11, 0x0F, 0x01, 0x0E},
	'h': {0x10, 0x10, 0x16, 0x19, 0x11, 0x11, 0x11},
	'i': {0x04, 0, 0x0C, 0x04, 0x04, 0x04, 0x0E},
	'k': {0x10, 0x10, 0x12, 0x14, 0x18, 0x14, 0x12},
	'l': {0x0C, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'm': {0, 0, 0x1A, 0x15, 0x15, 0x11, 0x11},
	'n': {0, 0, 0x16, 0x19, 0x11, 0x11, 0x11},
	'o': {0, 0, 0x0E, 0x11, 0x11, 0x11, 0x0E},
	'p': {0, 0, 0x1E, 0x11, 0x1E, 0x10, 0x10},
	'r': {0, 0, 0x16, 0x19, 0x10, 0x10, 0x10},
	's': {0, 0, 0x0F, 0x10, 0x0E, 0x01, 0x1E},
	't': {0x08, 0x08, 0x1C, 0x08, 0x08, 0x09, 0x06},
	'u': {0, 0, 0x11, 0x11, 0x11, 0x13, 0x0D},
	'v': {0, 0, 0x11, 0x11, 0x11, 0x0A, 0x04},
	'w': {0, 0, 0x11, 0x11, 0x15, 0x15, 0x0A},
	'x': {0, 0, 0x11, 0x0A, 0x04, 0x0A, 0x11},
	'y': {0, 0, 0x11, 0x11, 0x0F, 0x01, 0x0E},
	'z': {0, 0, 0x1F, 0x02, 0x04, 0x08, 0x1F},
}
