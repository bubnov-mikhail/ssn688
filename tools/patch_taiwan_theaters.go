// Command patch_taiwan_theaters rebuilds routes per theater and patches
// scenarios_generated/taiwan_formosa_watch.json with multi-theater bathy.
//
//	go run ./tools/patch_taiwan_theaters.go
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	scenarioPath  = "scenarios_generated/taiwan_formosa_watch.json"
	bathyDir      = "scenarios_generated/theater_bathy"
	clearanceYd   = 2800.0
	transitMargin = 1800.0
)

var missionTheater = map[string]string{
	"tw_quiet_waters":   "taiwan_east",
	"tw_twin_exercises": "taiwan_strait",
	"tw_attribution":    "taiwan_penghu",
	"tw_contested":      "taiwan_south",
	"tw_combined_asw":   "taiwan_north",
	"tw_break_pressure": "taiwan_lanyu",
}

func main() {
	deepenOut, err := exec.Command("python3", "tools/patch_penghu_deepen.py").CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("patch_penghu_deepen: %v\n%s", err, deepenOut))
	}
	fmt.Print(string(deepenOut))

	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		panic(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		panic(err)
	}

	theaters := loadTheaters()
	doc["theaters"] = theaters

	missions, _ := doc["missions"].([]any)
	for _, raw := range missions {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		mid, _ := m["id"].(string)
		tid := missionTheater[mid]
		if tid == "" {
			continue
		}
		bathy := theaterBathy[tid]
		if bathy == nil {
			panic("missing bathy for " + tid)
		}
		m["theater_id"] = tid
		if mid == "tw_attribution" {
			patchAttributionContent(m)
			patchAttributionEnemySpawns(m)
			patchAttributionCivSpawns(m)
			patchAttributionAllySpawns(m)
		}
		if mid == "tw_twin_exercises" {
			patchTwinExercises(m)
		}
		if mid == "tw_quiet_waters" {
			continue // keep authored routes on first mission
		}
		routes := rebuildMissionRoutes(bathy, m, tid)
		m["routes"] = routes
		fmt.Printf("mission %s -> theater %s (%d routes)\n", mid, tid, len(routes))
	}

	if v, ok := doc["version"].(string); ok {
		doc["version"] = bumpPatchVersion(v)
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(scenarioPath, append(out, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("patched", scenarioPath)
}

var theaterBathy = map[string]*world.Bathymetry{}

func loadTheaters() []any {
	ids := []string{"taiwan_east", "taiwan_strait", "taiwan_penghu", "taiwan_south", "taiwan_north", "taiwan_lanyu"}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		path := filepath.Join(bathyDir, id+".bin")
		raw, err := os.ReadFile(path)
		if err != nil {
			// Mission 1: reuse existing inline bathy from scenario if bin missing.
			if id == "taiwan_east" {
				raw = loadExistingTheaterBathy(id)
			}
			if raw == nil {
				panic(fmt.Sprintf("read %s: %v (run python tools/gen_bathy_zone.py first)", path, err))
			}
		}
		bathy, err := world.LoadBathymetry(raw)
		if err != nil {
			panic(err)
		}
		theaterBathy[id] = &bathy
		out = append(out, map[string]any{
			"id": id,
			"bathy": map[string]any{
				"mime":     "application/octet-stream",
				"data_b64": base64.StdEncoding.EncodeToString(raw),
			},
		})
		fmt.Printf("theater %s land check via routes later\n", id)
	}
	return out
}

func loadExistingTheaterBathy(id string) []byte {
	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		return nil
	}
	var doc map[string]any
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	theaters, _ := doc["theaters"].([]any)
	for _, th := range theaters {
		m, _ := th.(map[string]any)
		if m["id"] != id {
			continue
		}
		b, _ := m["bathy"].(map[string]any)
		b64, _ := b["data_b64"].(string)
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil
		}
		return raw
	}
	return nil
}

func rebuildMissionRoutes(bathy *world.Bathymetry, m map[string]any, theaterID string) []any {
	mid, _ := m["id"].(string)
	oldRoutes, _ := m["routes"].([]any)
	byID := make(map[string]map[string]any, len(oldRoutes))
	order := make([]string, 0, len(oldRoutes))
	for _, raw := range oldRoutes {
		rm, _ := raw.(map[string]any)
		id, _ := rm["id"].(string)
		if id == "" {
			continue
		}
		byID[id] = rm
		order = append(order, id)
	}
	for _, rid := range routeIDsFromUnits(m) {
		if _, ok := byID[rid]; !ok {
			byID[rid] = map[string]any{"id": rid, "mode": "pingpong"}
			order = append(order, rid)
		}
	}
	out := make([]any, 0, len(order))
	for i, id := range order {
		rm := byID[id]
		mode, _ := rm["mode"].(string)
		playerClear, _ := rm["player_clearance"].(bool)
		r := routeForID(bathy, id, i, theaterID, mid)
		if r == nil || len(r.Waypoints) < 2 {
			fmt.Printf("  WARN route %s failed to build\n", id)
			continue
		}
		if bad := validateRoute(bathy, r); len(bad) > 0 {
			if mid != "tw_attribution" || (id != "route_ally_688" && id != "route_ally_edge") {
				r = repairRouteLandHits(bathy, r)
			}
			if bad2 := validateRoute(bathy, r); len(bad2) > 0 {
				fmt.Printf("  WARN route %s has %d shore hits\n", id, len(bad2))
			}
		}
		out = append(out, routeJSON(r, mode, playerClear))
	}
	return out
}

func routeIDsFromUnits(m map[string]any) []string {
	units, _ := m["units"].([]any)
	seen := make(map[string]bool)
	var ids []string
	for _, raw := range units {
		u, _ := raw.(map[string]any)
		rid, _ := u["route_id"].(string)
		if rid == "" || seen[rid] {
			continue
		}
		seen[rid] = true
		ids = append(ids, rid)
	}
	return ids
}

func routeJSON(r *world.Route, mode string, playerClear bool) map[string]any {
	wps := make([]any, 0, len(r.Waypoints))
	for _, wp := range r.Waypoints {
		wps = append(wps, map[string]any{"x": wp.X, "y": wp.Y})
	}
	switch {
	case r.Looped:
		mode = "loop"
	case r.PingPong:
		mode = "pingpong"
	default:
		mode = "open"
	}
	obj := map[string]any{
		"id":        r.ID,
		"mode":      mode,
		"waypoints": wps,
	}
	if playerClear {
		obj["player_clearance"] = true
	}
	return obj
}

func routeForID(bathy *world.Bathymetry, id string, idx int, theaterID, missionID string) *world.Route {
	low := strings.ToLower(id)
	if theaterID == "taiwan_penghu" && missionID == "tw_attribution" {
		if r := penghuAttributionRoute(bathy, id, idx); r != nil {
			return r
		}
	}
	if theaterID == "taiwan_strait" {
		switch id {
		case "route_ally_ex", "route_ex_hulk_a", "route_ex_hulk_b", "route_rocn":
			return buildExerciseRoutesAtCenter(bathy, id)
		case "route_player_ex":
			return buildPlayerExerciseLane(bathy, id)
		case "route_rf_shadow":
			return buildShadowTankerStalkExercise(bathy, id)
		case "route_tanker":
			return buildStraitTankerLane(bathy, id)
		case "route_sw_patrol_a":
			return buildSWApproachCenter(bathy, id, 0)
		case "route_sw_patrol_b":
			return buildSWApproachCenter(bathy, id, 1)
		}
	}
	if strings.Contains(low, "sw_patrol") {
		slot := 0
		if strings.HasSuffix(low, "_b") {
			slot = 1
		}
		return buildCornerToCenterRoute(bathy, id, slot)
	}
	if theaterID == "taiwan_strait" {
		switch idx % 3 {
		case 0:
			return buildEastMarginPatrol(bathy, id, 14)
		case 1:
			return buildWaterBiasNSTransit(bathy, id, 0.78, float64(idx)*500, 14)
		default:
			return buildWaterBiasEWTransit(bathy, id, 0.72, float64(idx)*400, 14)
		}
	}
	if theaterID == "taiwan_penghu" {
		// Archipelago: northern open-water lanes only.
		yFrac := 0.08 + float64(idx%4)*0.03
		return buildCorridorEW(bathy, id, yFrac, float64(idx)*350, 14)
	}
	switch {
	case strings.Contains(low, "ally") || strings.Contains(low, "rocn") || strings.Contains(low, "_ex"):
		if r := buildEastMarginPatrol(bathy, id, 14); r != nil {
			return r
		}
		return world.BuildAllyEdgePatrol(bathy, id, 14)
	case strings.Contains(low, "merchant") || strings.Contains(low, "tanker") || strings.Contains(low, "trawler") || strings.Contains(low, "dawn") || strings.Contains(low, "hulk"):
		off := -3500.0 + float64(idx%7)*1100
		if idx%2 == 0 {
			if r := buildEWTransit(bathy, id, off, 16); r != nil {
				return r
			}
		}
		return world.BuildNWSETransit(bathy, id, off, 16)
	case strings.Contains(low, "coast") || strings.Contains(low, "grisha") || strings.Contains(low, "watch"):
		cw := idx%2 == 0
		dist := clearanceYd + float64(idx%4)*400
		if r := world.BuildCoastalLoop(bathy, id, dist, cw); r != nil {
			return r
		}
		return world.BuildNWSETransit(bathy, id, float64(idx)*800, 14)
	case strings.Contains(low, "kilo") || strings.Contains(low, "victor") || strings.Contains(low, "yasen") || strings.Contains(low, "gorshkov") || strings.Contains(low, "udaloy") || strings.Contains(low, "kresta") || strings.Contains(low, "shadow") || strings.Contains(low, "rf_"):
		if r := world.BuildApproachFromNE(bathy, id, 12); r != nil {
			return r
		}
		return buildNSTransit(bathy, id, float64(idx)*900, 14)
	default:
		off := -2000.0 + float64(idx%5)*900
		if r := buildNSTransit(bathy, id, off, 14); r != nil {
			return r
		}
		return world.BuildNWSETransit(bathy, id, off, 14)
	}
}

func buildEWTransit(bathy *world.Bathymetry, id string, lateralOffsetYd float64, numWP int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitMargin
	start := world.Waypoint{X: minX + m, Y: (minY + maxY) * 0.5}
	end := world.Waypoint{X: maxX - m, Y: (minY + maxY) * 0.5}
	dx, dy := end.X-start.X, end.Y-start.Y
	span := math.Hypot(dx, dy)
	if span < 1 {
		return nil
	}
	px, py := -dy/span, dx/span
	start.X += px * lateralOffsetYd
	start.Y += py * lateralOffsetYd
	end.X += px * lateralOffsetYd
	end.Y += py * lateralOffsetYd
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		x := start.X + (end.X-start.X)*t
		y := start.Y + (end.Y-start.Y)*t
		x, y = snapClear(bathy, x, y)
		wps = append(wps, world.Waypoint{X: x, Y: y})
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildNSTransit(bathy *world.Bathymetry, id string, lateralOffsetYd float64, numWP int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitMargin
	start := world.Waypoint{X: (minX + maxX) * 0.5, Y: maxY - m}
	end := world.Waypoint{X: (minX + maxX) * 0.5, Y: minY + m}
	dx, dy := end.X-start.X, end.Y-start.Y
	span := math.Hypot(dx, dy)
	if span < 1 {
		return nil
	}
	px, py := -dy/span, dx/span
	start.X += px * lateralOffsetYd
	start.Y += py * lateralOffsetYd
	end.X += px * lateralOffsetYd
	end.Y += py * lateralOffsetYd
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		x := start.X + (end.X-start.X)*t
		y := start.Y + (end.Y-start.Y)*t
		x, y = snapClear(bathy, x, y)
		wps = append(wps, world.Waypoint{X: x, Y: y})
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildFixedPatrol(id string, x0, y0, x1, y1 float64) *world.Route {
	wps := []world.Waypoint{{X: x0, Y: y0}, {X: x1, Y: y1}}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func mapCenter(bathy *world.Bathymetry) (float64, float64) {
	minX, minY, maxX, maxY := bathy.BoundsYards()
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	return snapClear(bathy, cx, cy)
}

func buildExerciseRoutesAtCenter(bathy *world.Bathymetry, id string) *world.Route {
	cx, cy := mapCenter(bathy)
	switch id {
	case "route_ally_ex":
		return buildFixedPatrol(id, cx-500, cy, cx-500, cy+800)
	case "route_ex_hulk_a":
		return buildFixedPatrol(id, cx+1100, cy, cx+2100, cy)
	case "route_ex_hulk_b":
		return buildFixedPatrol(id, cx-2100, cy, cx-1100, cy)
	case "route_rocn":
		return buildFixedPatrol(id, cx-500, cy-800, cx+500, cy-800)
	default:
		return nil
	}
}

func buildPlayerExerciseLane(bathy *world.Bathymetry, id string) *world.Route {
	cx, cy := mapCenter(bathy)
	// North of the exercise fish lane; periscope-depth ownship sits here at route_frac ~0.5.
	return buildFixedPatrol(id, cx-300, cy+650, cx+700, cy+650)
}

// buildSWApproachCenter: start SW (bottom-left), patrol toward map center.
func buildSWApproachCenter(bathy *world.Bathymetry, id string, slot int) *world.Route {
	minX, minY, maxX, maxY := bathy.BoundsYards()
	cx, cy := mapCenter(bathy)
	starts := [][2]float64{{0.10, 0.10}, {0.14, 0.08}}
	s := starts[slot%len(starts)]
	x0 := minX + (maxX-minX)*s[0]
	y0 := minY + (maxY-minY)*s[1]
	x0, y0 = snapClear(bathy, x0, y0)
	mx := x0 + (cx-x0)*0.55
	my := y0 + (cy-y0)*0.55
	mx, my = snapClear(bathy, mx, my)
	return &world.Route{
		ID:        id,
		Waypoints: []world.Waypoint{{X: x0, Y: y0}, {X: mx, Y: my}, {X: cx, Y: cy}},
		PingPong:  true,
	}
}

// buildStraitTankerLane: east–west merchant transit south of the exercise box.
func buildStraitTankerLane(bathy *world.Bathymetry, id string) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	_, cy := mapCenter(bathy)
	minX, _, maxX, _ := bathy.BoundsYards()
	m := transitMargin
	laneY := cy - 2300.0
	const numWP = 12
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		x := minX + m + (maxX-minX-2*m)*t
		wps = append(wps, snapWPNearY(bathy, x, laneY))
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

// buildShadowTankerStalkExercise: trail a southbound tanker lane, then orbit the
// exercise area from the east (right) at standoff.
func buildShadowTankerStalkExercise(bathy *world.Bathymetry, id string) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	cx, cy := mapCenter(bathy)
	minX, _, maxX, _ := bathy.BoundsYards()
	laneY := cy - 2300.0
	trailY := cy - 2800.0
	span := maxX - minX
	xWest := minX + span*0.18
	xMid := minX + span*0.30
	xTrail := minX + span*0.42
	xPeel := minX + span*0.52
	const orbitX = 6500.0
	wps := []world.Waypoint{
		snapWPNearY(bathy, xWest, trailY),
		snapWPNearY(bathy, xMid, trailY),
		snapWPNearY(bathy, xTrail, laneY),
		snapWPNearY(bathy, xPeel, laneY-600),
		snapWP(bathy, cx+orbitX*0.75, cy-3200),
		snapWP(bathy, cx+orbitX, cy-3000),
		snapWP(bathy, cx+orbitX, cy+2600),
	}
	return &world.Route{ID: id, Waypoints: wps, Looped: true}
}

// buildShadowApproachExercise is the legacy SW→E loop (unused on twin exercises).
func buildShadowApproachExercise(bathy *world.Bathymetry, id string) *world.Route {
	cx, cy := mapCenter(bathy)
	const orbitX = 6500.0
	// SW → SE → NE loop outside exercise box; wide standoff so ID takes time.
	offsets := [][2]float64{{-orbitX, -3000}, {orbitX, -3000}, {orbitX, 2600}}
	wps := make([]world.Waypoint, 0, len(offsets))
	for _, off := range offsets {
		x, y := snapClear(bathy, cx+off[0], cy+off[1])
		wps = append(wps, world.Waypoint{X: x, Y: y})
	}
	return &world.Route{ID: id, Waypoints: wps, Looped: true}
}

func buildCornerToCenterRoute(bathy *world.Bathymetry, id string, slot int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	starts := [][2]float64{{0.12, 0.12}, {0.16, 0.10}}
	s := starts[slot%len(starts)]
	x0 := minX + (maxX-minX)*s[0]
	y0 := minY + (maxY-minY)*s[1]
	x0, y0 = snapClear(bathy, x0, y0)
	cx, cy = snapClear(bathy, cx, cy)
	wps := []world.Waypoint{{X: x0, Y: y0}, {X: cx, Y: cy}}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildCorridorEW(bathy *world.Bathymetry, id string, yFrac float64, lateralOffsetYd float64, numWP int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitMargin
	y := minY + (maxY-minY)*yFrac + lateralOffsetYd
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		x := minX + m + (maxX-minX-2*m)*t
		nx, ny := snapClear(bathy, x, y)
		wps = append(wps, world.Waypoint{X: nx, Y: ny})
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildWaterBiasEWTransit(bathy *world.Bathymetry, id string, yFrac float64, lateralOffsetYd float64, numWP int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitMargin
	y := minY + (maxY-minY)*yFrac
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		x := minX + m + (maxX-minX-2*m)*t*0.55 + (maxX-minX)*0.35 // eastern band
		x += lateralOffsetYd * 0.1
		nx, ny := snapClear(bathy, x, y+lateralOffsetYd)
		wps = append(wps, world.Waypoint{X: nx, Y: ny})
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildWaterBiasNSTransit(bathy *world.Bathymetry, id string, xFrac float64, lateralOffsetYd float64, numWP int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	minX, minY, maxX, maxY := bathy.BoundsYards()
	m := transitMargin
	x := minX + (maxX-minX)*xFrac
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		y := minY + m + (maxY-minY-2*m)*t
		y += lateralOffsetYd
		nx, ny := snapClear(bathy, x, y)
		wps = append(wps, world.Waypoint{X: nx, Y: ny})
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildEastMarginPatrol(bathy *world.Bathymetry, id string, numWP int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	_, minY, maxX, maxY := bathy.BoundsYards()
	m := transitMargin
	x := maxX - m
	wps := make([]world.Waypoint, 0, numWP)
	for i := 0; i < numWP; i++ {
		t := float64(i) / float64(numWP-1)
		y := minY + m + (maxY-minY-2*m)*t
		nx, ny := snapClear(bathy, x, y)
		wps = append(wps, world.Waypoint{X: nx, Y: ny})
	}
	if len(wps) < 3 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func snapClear(bathy *world.Bathymetry, x, y float64) (float64, float64) {
	if bathy.NavigableFor(x, y, world.KindSurfaceShip, 0) && bathy.DistanceToShoreYd(x, y) >= clearanceYd {
		return x, y
	}
	bestX, bestY := x, y
	bestScore := math.MaxFloat64
	for rad := 200.0; rad <= 9000; rad += 200 {
		for brg := 0; brg < 360; brg += 15 {
			a := float64(brg) * math.Pi / 180
			nx := x + math.Sin(a)*rad
			ny := y + math.Cos(a)*rad
			if !bathy.NavigableFor(nx, ny, world.KindSurfaceShip, 0) {
				continue
			}
			d := bathy.DistanceToShoreYd(nx, ny)
			if d < clearanceYd {
				continue
			}
			score := rad - d*0.12
			if score < bestScore {
				bestScore = score
				bestX, bestY = nx, ny
			}
		}
	}
	return bestX, bestY
}

// snapClearNearY keeps a transit lane near the requested latitude (avoids snapping to map center).
func snapClearNearY(bathy *world.Bathymetry, x, wantY float64) (float64, float64) {
	if bathy.NavigableFor(x, wantY, world.KindSurfaceShip, 0) && bathy.DistanceToShoreYd(x, wantY) >= clearanceYd {
		return x, wantY
	}
	bestX, bestY := x, wantY
	bestScore := math.MaxFloat64
	for dy := -1200.0; dy <= 1200.0; dy += 150 {
		for dx := -600.0; dx <= 600.0; dx += 150 {
			nx, ny := x+dx, wantY+dy
			if !bathy.NavigableFor(nx, ny, world.KindSurfaceShip, 0) {
				continue
			}
			d := bathy.DistanceToShoreYd(nx, ny)
			if d < clearanceYd {
				continue
			}
			score := math.Abs(dy) + math.Abs(dx)*0.35 - d*0.08
			if score < bestScore {
				bestScore = score
				bestX, bestY = nx, ny
			}
		}
	}
	if bestScore < math.MaxFloat64 {
		return bestX, bestY
	}
	return snapClear(bathy, x, wantY)
}

func snapWP(bathy *world.Bathymetry, x, y float64) world.Waypoint {
	x, y = snapClear(bathy, x, y)
	return world.Waypoint{X: x, Y: y}
}

func snapWPNearY(bathy *world.Bathymetry, x, wantY float64) world.Waypoint {
	x, y := snapClearNearY(bathy, x, wantY)
	return world.Waypoint{X: x, Y: y}
}

// snapPenghu nudges onto navigable water near authored straits/channels (lower clearance).
func snapPenghu(bathy *world.Bathymetry, x, y float64) (float64, float64) {
	return snapPenghuNear(bathy, x, y, 3500)
}

func snapPenghuNear(bathy *world.Bathymetry, x, y, maxRad float64) (float64, float64) {
	const coastalClear = 450.0
	if bathy.NavigableFor(x, y, world.KindSurfaceShip, 0) && bathy.DistanceToShoreYd(x, y) >= coastalClear {
		return x, y
	}
	bestX, bestY := x, y
	bestScore := math.MaxFloat64
	for rad := 150.0; rad <= maxRad; rad += 150 {
		for brg := 0; brg < 360; brg += 12 {
			a := float64(brg) * math.Pi / 180
			nx := x + math.Sin(a)*rad
			ny := y + math.Cos(a)*rad
			if !bathy.NavigableFor(nx, ny, world.KindSurfaceShip, 0) {
				continue
			}
			d := bathy.DistanceToShoreYd(nx, ny)
			if d < coastalClear {
				continue
			}
			score := rad + math.Hypot(nx-x, ny-y)*0.45 - d*0.08
			if score < bestScore {
				bestScore = score
				bestX, bestY = nx, ny
			}
		}
	}
	return bestX, bestY
}

func snapPenghuWP(bathy *world.Bathymetry, x, y float64) world.Waypoint {
	x, y = snapPenghu(bathy, x, y)
	return world.Waypoint{X: x, Y: y}
}

func penghuAttributionRoute(bathy *world.Bathymetry, id string, idx int) *world.Route {
	switch id {
	case "route_ally_edge":
		return buildPenghuAllySurfaceTransit(bathy, id)
	case "route_ally_688":
		return buildPenghuAlly688Transit(bathy, id)
	case "route_trawler":
		return buildPenghuEnemyMarkup(bathy, id, penghuTrawlerAnchors)
	case "route_rf_victor":
		return buildPenghuEnemyMarkup(bathy, id, offsetFormationAnchors(penghuEnemyMainAnchors, 320))
	case "route_plan_extra": // Krivak — lead in column
		return buildPenghuEnemyMarkup(bathy, id, penghuEnemyMainAnchors)
	case "route_plan_grisha":
		return buildPenghuEnemyMarkup(bathy, id, offsetFormationAnchors(penghuEnemyMainAnchors, -320))
	case "route_rf_kilo":
		return buildPenghuEnemyMarkup(bathy, id, offsetFormationAnchors(penghuKiloAnchors, 320))
	default:
		return nil
	}
}

// buildPenghuSpruancePatrol starts between the two northern islands, curves through the
// central strait, and loops via the rendezvous box in the SE (enemy approach lanes).
func buildPenghuSpruancePatrol(bathy *world.Bathymetry, id string) *world.Route {
	anchors := [][2]float64{
		{5200, 5500}, {6800, 3000}, {9200, -200}, {11600, -7000},
		{11200, 1200}, {6800, 4800},
	}
	return &world.Route{ID: id, Waypoints: buildPenghuLoop(bathy, anchors), Looped: true}
}

// penghuAlly688Waypoints — north spawn → Spruance lane (wp2/wp3) → south via east channel.
var penghuAlly688Waypoints = [][2]float64{
	{10600, 5900}, {9800, 5200}, {9000, 4900}, {8100, 4900},
	{8094, 1690}, {9344, -1560}, {9800, -2400}, {10800, -4500},
	{11500, -7200}, {11900, -10400},
}

func buildPenghuAlly688Transit(bathy *world.Bathymetry, id string) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	wps := make([]world.Waypoint, len(penghuAlly688Waypoints))
	for i, a := range penghuAlly688Waypoints {
		wps[i] = world.Waypoint{X: a[0], Y: a[1]}
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: false}
}

// buildPenghu688Patrol starts in the east strait (top island vs open water to the right).
func buildPenghu688Patrol(bathy *world.Bathymetry, id string) *world.Route {
	anchors := [][2]float64{
		{10400, 6000}, {8200, 1400}, {11600, -7000}, {11200, 1500},
	}
	return &world.Route{ID: id, Waypoints: buildPenghuLoop(bathy, anchors), Looped: true}
}

func buildPenghuLoop(bathy *world.Bathymetry, anchors [][2]float64) []world.Waypoint {
	if bathy == nil || len(anchors) < 2 {
		return nil
	}
	pts := make([]world.Waypoint, 0, len(anchors))
	for _, a := range anchors {
		x, y := snapPenghuNear(bathy, a[0], a[1], 1200)
		pts = append(pts, world.Waypoint{X: x, Y: y})
	}
	return densifyPenghuLoop(bathy, pts)
}

// densifyPenghuLoop keeps smooth anchor legs; only inserts a short BFS chain when a leg crosses shore.
func densifyPenghuLoop(bathy *world.Bathymetry, anchors []world.Waypoint) []world.Waypoint {
	const step = 250.0
	var out []world.Waypoint
	n := len(anchors)
	for i := 0; i < n; i++ {
		a, b := anchors[i], anchors[(i+1)%n]
		leg := []world.Waypoint{a, b}
		if len(sampleSegmentHits(bathy, a.X, a.Y, b.X, b.Y)) > 0 {
			leg = penghuBFS(bathy, a, b, step)
		}
		if len(out) == 0 {
			out = append(out, leg...)
		} else {
			out = append(out, leg[1:]...)
		}
	}
	return sanitizePenghuLoop(bathy, out)
}

// sanitizePenghuLoop re-stitches any loop leg whose straight chord crosses shore.
func sanitizePenghuLoop(bathy *world.Bathymetry, wps []world.Waypoint) []world.Waypoint {
	const step = 250.0
	if len(wps) < 2 {
		return wps
	}
	for pass := 0; pass < 8; pass++ {
		changed := false
		var out []world.Waypoint
		for i := 0; i < len(wps); i++ {
			a, b := wps[i], wps[(i+1)%len(wps)]
			leg := []world.Waypoint{a, b}
			if len(sampleSegmentHits(bathy, a.X, a.Y, b.X, b.Y)) > 0 {
				if path := penghuBFS(bathy, a, b, step); len(path) >= 2 {
					leg = path
					changed = true
				}
			}
			if len(out) == 0 {
				out = append(out, leg...)
			} else {
				out = append(out, leg[1:]...)
			}
		}
		if !changed {
			return out
		}
		wps = out
	}
	return wps
}

func buildPenghuStraitTransit(bathy *world.Bathymetry, id string, xOff float64) *world.Route {
	anchors := [][2]float64{
		{5400 + xOff, 2700}, {6400 + xOff, 2300}, {7400 + xOff, 1700}, {8600 + xOff, 1000}, {9400 + xOff, 750},
	}
	wps := stitchPenghuAnchors(bathy, anchors, 0, 0)
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func stitchPenghuAnchors(bathy *world.Bathymetry, anchors [][2]float64, latOff, simplifyYd float64) []world.Waypoint {
	if bathy == nil || len(anchors) < 2 {
		return nil
	}
	const step = 250.0
	var chain []world.Waypoint
	for i := 1; i < len(anchors); i++ {
		a := world.Waypoint{X: anchors[i-1][0], Y: anchors[i-1][1] + latOff}
		b := world.Waypoint{X: anchors[i][0], Y: anchors[i][1] + latOff}
		seg := penghuBFS(bathy, a, b, step)
		if len(seg) < 2 {
			seg = []world.Waypoint{a, b}
		}
		if len(chain) == 0 {
			chain = append(chain, seg...)
		} else {
			chain = append(chain, seg[1:]...)
		}
	}
	if simplifyYd > 0 {
		chain = simplifyPenghuWaypoints(chain, simplifyYd)
	}
	return chain
}

func penghuBFS(bathy *world.Bathymetry, a, b world.Waypoint, step float64) []world.Waypoint {
	type node struct {
		x, y float64
	}
	start := node{a.X, a.Y}
	goal := node{b.X, b.Y}
	if !penghuNav(bathy, start.x, start.y) {
		return []world.Waypoint{a, b}
	}
	parent := map[node]node{start: {}}
	q := []node{start}
	for len(q) > 0 && len(q) < 12000 {
		cur := q[0]
		q = q[1:]
		if math.Hypot(cur.x-goal.x, cur.y-goal.y) < step*1.5 {
			// reconstruct
			var rev []node
			for n := cur; n != start; n = parent[n] {
				rev = append(rev, n)
			}
			out := []world.Waypoint{{X: start.x, Y: start.y}}
			for i := len(rev) - 1; i >= 0; i-- {
				out = append(out, world.Waypoint{X: rev[i].x, Y: rev[i].y})
			}
			out = append(out, b)
			return out
		}
		for _, d := range [][2]float64{{0, -step}, {step, 0}, {-step, 0}, {0, step}} {
			nx, ny := cur.x+d[0], cur.y+d[1]
			n := node{nx, ny}
			if _, ok := parent[n]; ok {
				continue
			}
			if !penghuNav(bathy, nx, ny) {
				continue
			}
			parent[n] = cur
			q = append(q, n)
		}
	}
	return []world.Waypoint{a, b}
}

func penghuNav(bathy *world.Bathymetry, x, y float64) bool {
	if bathy == nil {
		return false
	}
	fx := (x - bathy.OriginX) / bathy.CellSize
	fy := (y - bathy.OriginY) / bathy.CellSize
	if fx < 0 || fy < 0 || fx >= float64(bathy.Width) || fy >= float64(bathy.Height) {
		return false
	}
	return !bathy.IsShoreBlocked(x, y)
}

func simplifyPenghuWaypoints(wps []world.Waypoint, minSpan float64) []world.Waypoint {
	if len(wps) < 2 {
		return wps
	}
	out := []world.Waypoint{wps[0]}
	for _, wp := range wps[1:] {
		last := out[len(out)-1]
		if math.Hypot(wp.X-last.X, wp.Y-last.Y) >= minSpan {
			out = append(out, wp)
		}
	}
	if out[len(out)-1] != wps[len(wps)-1] {
		out = append(out, wps[len(wps)-1])
	}
	return out
}

func buildPenghuWestEdge(bathy *world.Bathymetry, id string) *world.Route {
	minX, minY, _, maxY := bathy.BoundsYards()
	x := minX + 2200
	wps := []world.Waypoint{
		snapPenghuWP(bathy, x, minY+2400),
		snapPenghuWP(bathy, x, maxY-3200),
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

func buildPenghuEnemyTransit(bathy *world.Bathymetry, id string, slot int) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	lanes := []float64{-9700, -9600, -9500}
	laneY := lanes[slot%len(lanes)]
	minX, _, maxX, _ := bathy.BoundsYards()
	m := transitMargin
	x0, _ := snapClearNearY(bathy, minX+m, laneY)
	x1, _ := snapClearNearY(bathy, maxX-m, laneY)
	wps := []world.Waypoint{
		{X: x0, Y: laneY},
		{X: x1, Y: laneY},
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: true}
}

// penghuEnemyMainAnchors — player markup on coast template (southwest → along bottom → NE).
var penghuEnemyMainAnchors = [][2]float64{
	{-9200, -6800}, {-4500, -11800}, {-800, -12600},
	{5500, -12200}, {9500, -11500}, {11500, -6000}, {10800, 1500},
}

// penghuKiloAnchors — quieter/deeper southern lane for the interloper.
var penghuKiloAnchors = [][2]float64{
	{-9000, -9500}, {-3000, -12800}, {4000, -12900},
	{9500, -11800}, {11800, -7500}, {11000, 500},
}

// penghuTrawlerAnchors — player markup: west edge into the southern confrontation lane.
var penghuTrawlerAnchors = [][2]float64{
	{-11800, -1200}, {-7500, -5000}, {-2500, -9500},
	{4000, -9800}, {6500, -9700}, {9000, -9400}, {12000, -8500},
}

const penghuFormationMaxOffsetYd = 500.0

// offsetFormationAnchors shifts each anchor perpendicular to the local leg (tight column/echelon).
func offsetFormationAnchors(anchors [][2]float64, offsetYd float64) [][2]float64 {
	if len(anchors) == 0 || offsetYd == 0 {
		return anchors
	}
	out := make([][2]float64, len(anchors))
	for i, a := range anchors {
		dx, dy := 0.0, 1.0
		if i+1 < len(anchors) {
			dx = anchors[i+1][0] - a[0]
			dy = anchors[i+1][1] - a[1]
		} else if i > 0 {
			dx = a[0] - anchors[i-1][0]
			dy = a[1] - anchors[i-1][1]
		}
		if mag := math.Hypot(dx, dy); mag > 1 {
			dx /= mag
			dy /= mag
		}
		px := -dy * offsetYd
		py := dx * offsetYd
		if mag := math.Hypot(px, py); mag > penghuFormationMaxOffsetYd {
			s := penghuFormationMaxOffsetYd / mag
			px *= s
			py *= s
		}
		out[i] = [2]float64{a[0] + px, a[1] + py}
	}
	return out
}

// penghuAllySurfaceAnchors — player markup: north strait → south along east channel.
var penghuAllySurfaceAnchors = [][2]float64{
	{8100, 5200}, {9200, 4800}, {10200, 6000}, {10400, 4500},
	{10000, 2500}, {9800, -1800}, {10800, -7800}, {11800, -10800},
}

func stitchPenghuMarkupShort(bathy *world.Bathymetry, anchors [][2]float64) []world.Waypoint {
	const step = 250.0
	pts := make([]world.Waypoint, 0, len(anchors))
	for _, a := range anchors {
		x, y := snapPenghuNear(bathy, a[0], a[1], 800)
		pts = append(pts, world.Waypoint{X: x, Y: y})
	}
	var out []world.Waypoint
	for i := 0; i < len(pts)-1; i++ {
		a, b := pts[i], pts[i+1]
		leg := []world.Waypoint{a, b}
		if len(sampleSegmentHits(bathy, a.X, a.Y, b.X, b.Y)) > 0 {
			if path := penghuBFS(bathy, a, b, step); len(path) >= 2 {
				if len(path) > 3 {
					mid := path[len(path)/2]
					leg = []world.Waypoint{a, mid, b}
				} else {
					leg = path
				}
			}
		}
		if len(out) == 0 {
			out = append(out, leg...)
		} else {
			out = append(out, leg[1:]...)
		}
	}
	return out
}

func buildPenghuEnemyMarkup(bathy *world.Bathymetry, id string, anchors [][2]float64) *world.Route {
	if bathy == nil || !bathy.Valid() || len(anchors) < 2 {
		return nil
	}
	wps := stitchPenghuMarkupShort(bathy, anchors)
	if len(wps) < 2 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: false}
}

func buildPenghuAllySurfaceTransit(bathy *world.Bathymetry, id string) *world.Route {
	if bathy == nil || !bathy.Valid() {
		return nil
	}
	pts := make([]world.Waypoint, 0, len(penghuAllySurfaceAnchors))
	for _, a := range penghuAllySurfaceAnchors {
		x, y := snapPenghuNear(bathy, a[0], a[1], 800)
		pts = append(pts, world.Waypoint{X: x, Y: y})
	}
	if len(pts) < 2 {
		return nil
	}
	wps := stitchPenghuMarkupShort(bathy, penghuAllySurfaceAnchors)
	if openLegHits(bathy, wps) > 0 {
		wps = densifyPenghuOpen(bathy, pts)
	}
	short := simplifyPenghuWaypoints(wps, 3200)
	if openLegHits(bathy, short) == 0 && len(short) >= 2 {
		wps = short
	}
	if len(wps) < 2 {
		return nil
	}
	return &world.Route{ID: id, Waypoints: wps, PingPong: false}
}

func openLegHits(bathy *world.Bathymetry, wps []world.Waypoint) int {
	n := 0
	for i := 0; i < len(wps)-1; i++ {
		n += len(sampleSegmentHits(bathy, wps[i].X, wps[i].Y, wps[i+1].X, wps[i+1].Y))
	}
	return n
}

func densifyPenghuOpen(bathy *world.Bathymetry, anchors []world.Waypoint) []world.Waypoint {
	const step = 250.0
	if len(anchors) < 2 {
		return anchors
	}
	var out []world.Waypoint
	for i := 0; i < len(anchors)-1; i++ {
		a, b := anchors[i], anchors[i+1]
		leg := []world.Waypoint{a, b}
		if len(sampleSegmentHits(bathy, a.X, a.Y, b.X, b.Y)) > 0 {
			leg = penghuBFS(bathy, a, b, step)
		}
		if len(out) == 0 {
			out = append(out, leg...)
		} else {
			out = append(out, leg[1:]...)
		}
	}
	return sanitizePenghuOpen(bathy, out)
}

func sanitizePenghuOpen(bathy *world.Bathymetry, wps []world.Waypoint) []world.Waypoint {
	const step = 250.0
	if len(wps) < 2 {
		return wps
	}
	for pass := 0; pass < 8; pass++ {
		changed := false
		var out []world.Waypoint
		for i := 0; i < len(wps)-1; i++ {
			a, b := wps[i], wps[i+1]
			leg := []world.Waypoint{a, b}
			if len(sampleSegmentHits(bathy, a.X, a.Y, b.X, b.Y)) > 0 {
				if path := penghuBFS(bathy, a, b, step); len(path) >= 2 {
					leg = path
					changed = true
				}
			}
			if len(out) == 0 {
				out = append(out, leg...)
			} else {
				out = append(out, leg[1:]...)
			}
		}
		if !changed {
			return out
		}
		wps = out
	}
	return wps
}

func patchAttributionEnemySpawns(m map[string]any) {
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		side, _ := u["side"].(string)
		if side != "enemy" {
			continue
		}
		spawn, _ := u["spawn"].(string)
		if spawn != "route" {
			continue
		}
		id, _ := u["id"].(string)
		if id == "rf_kilo_quiet" {
			u["route_id"] = "route_rf_kilo"
		}
		u["route_frac"] = 0.0
	}
}

func patchAttributionCivSpawns(m map[string]any) {
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := u["id"].(string)
		if id != "civ_trawler" {
			continue
		}
		spawn, _ := u["spawn"].(string)
		if spawn != "route" {
			continue
		}
		u["route_frac"] = 0.0
	}
}

func patchAttributionAllySpawns(m map[string]any) {
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := u["id"].(string)
		spawn, _ := u["spawn"].(string)
		if spawn != "route" {
			continue
		}
		switch id {
		case "ally_spruance":
			u["route_frac"] = 0.0
		case "ally_rocn":
			u["route_frac"] = 0.05
		case "ally_688":
			u["route_frac"] = 0.0
		}
	}
}

func patchAttributionContent(m map[string]any) {
	patchAttributionCrewSkills(m)
	patchAttributionPayloads(m)
	patchAttributionSubDepths(m)
	stripUnitIDs := map[string]bool{"civ_merchant": true, "civ_tanker": true}
	stripRouteIDs := map[string]bool{"route_merchant": true, "route_tanker": true}
	if units, ok := m["units"].([]any); ok {
		filtered := make([]any, 0, len(units))
		for _, raw := range units {
			u, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := u["id"].(string)
			if stripUnitIDs[id] {
				continue
			}
			if combat, _ := u["combatant"].(bool); combat {
				u["defcon"] = 3 // weapons free from mission start
			}
			filtered = append(filtered, u)
		}
		m["units"] = filtered
	}
	if routes, ok := m["routes"].([]any); ok {
		filtered := make([]any, 0, len(routes))
		for _, raw := range routes {
			r, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := r["id"].(string)
			if stripRouteIDs[id] {
				continue
			}
			filtered = append(filtered, r)
		}
		m["routes"] = filtered
	}
	if objs, ok := m["objectives"].([]any); ok {
		filtered := make([]any, 0, len(objs))
		for _, raw := range objs {
			o, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := o["id"].(string)
			if id == "obj_tanker_id" {
				continue
			}
			filtered = append(filtered, o)
		}
		m["objectives"] = filtered
	}
	m["events"] = patchAttributionEvents(m)
}

// patchTwinExercises keeps exercise geometry non-escalatory: hulks are targets only;
// allied escorts and PLAN screen hold fire (empty magazines, passive DEFCON).
func patchTwinExercises(m map[string]any) {
	empty := map[string]any{
		"torpedoes": 0, "harpoons": 0, "cruise_missiles": 0,
		"asw_rockets": 0, "ship_tubes": 0, "rbu": 0, "sam": 0, "ciws": 0,
	}
	planScreen := []string{
		"plan_watch", "plan_watch_extra", "plan_sw_patrol_a", "plan_sw_patrol_b",
	}
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := u["id"].(string)
		switch id {
		case "ex_hulk_a", "ex_hulk_b":
			u["signature_id"] = "exercise_hulk"
			u["exercise_target"] = true
			u["ally_ignore"] = true
			u["defcon"] = 0
			u["payload"] = map[string]any{
				"torpedoes": 0, "harpoons": 0, "cruise_missiles": 0,
				"asw_rockets": 0, "ship_tubes": 0, "rbu": 0, "sam": 0, "ciws": 0,
				"exercise_torpedoes": 2,
			}
		case "ally_spruance", "ally_rocn":
			u["defcon"] = 0
			u["payload"] = empty
		case "rf_shadow":
			u["defcon"] = 1
			u["payload"] = map[string]any{"torpedoes": 0, "cruise_missiles": 0}
		default:
			for _, pid := range planScreen {
				if id == pid {
					u["defcon"] = 0
					u["payload"] = empty
					break
				}
			}
		}
	}
}

// patchAttributionCrewSkills — allies need player help; enemies are veterans.
func patchAttributionCrewSkills(m map[string]any) {
	skills := map[string][2]float64{
		"ally_spruance":   {50, 14},
		"ally_rocn":       {46, 16},
		"ally_688":        {48, 12},
		"rf_victor":       {92, 4},
		"plan_grisha":     {88, 5},
		"plan_krivak":     {90, 4},
		"rf_kilo_quiet":   {90, 4},
	}
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := u["id"].(string)
		if sk, ok := skills[id]; ok {
			u["crew_skill"] = sk[0]
			u["crew_jitter"] = sk[1]
		}
	}
}

// patchAttributionPayloads seeds full class magazines for every combatant.
func patchAttributionPayloads(m map[string]any) {
	type payload struct {
		Torpedoes      int `json:"torpedoes,omitempty"`
		Harpoons       int `json:"harpoons,omitempty"`
		CruiseMissiles int `json:"cruise_missiles,omitempty"`
		ASWRockets     int `json:"asw_rockets,omitempty"`
		ShipTubes  int `json:"ship_tubes,omitempty"`
		RBU        int `json:"rbu,omitempty"`
		SAM        int `json:"sam,omitempty"`
		CIWS       int `json:"ciws,omitempty"`
	}
	specs := map[string]payload{
		"rf_victor":     {Torpedoes: 18},
		"rf_kilo_quiet": {Torpedoes: 14, CruiseMissiles: 4},
		"plan_grisha":   {RBU: 10, ShipTubes: 4, SAM: 4, CIWS: 6},
		"plan_krivak":   {ASWRockets: 8, ShipTubes: 6, SAM: 8, CIWS: 12},
		"ally_spruance": {ASWRockets: 8, ShipTubes: 6, SAM: 24, CIWS: 12},
		"ally_rocn":     {ASWRockets: 8, ShipTubes: 6, SAM: 8, CIWS: 12},
		"ally_688":      {Torpedoes: 14, Harpoons: 8},
	}
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := u["id"].(string)
		p, ok := specs[id]
		if !ok {
			continue
		}
		pm := map[string]any{}
		if p.Torpedoes > 0 {
			pm["torpedoes"] = p.Torpedoes
		}
		if p.Harpoons > 0 {
			pm["harpoons"] = p.Harpoons
		}
		if p.CruiseMissiles > 0 {
			pm["cruise_missiles"] = p.CruiseMissiles
		}
		if p.ASWRockets > 0 {
			pm["asw_rockets"] = p.ASWRockets
		}
		if p.ShipTubes > 0 {
			pm["ship_tubes"] = p.ShipTubes
		}
		if p.RBU > 0 {
			pm["rbu"] = p.RBU
		}
		if p.SAM > 0 {
			pm["sam"] = p.SAM
		}
		if p.CIWS > 0 {
			pm["ciws"] = p.CIWS
		}
		u["payload"] = pm
	}
	if p, ok := m["player"].(map[string]any); ok {
		p["payload"] = map[string]any{"torpedoes": 22}
	}
}

// patchAttributionSubDepths — periscope-depth start for all subs (matches player).
func patchAttributionSubDepths(m map[string]any) {
	if p, ok := m["player"].(map[string]any); ok {
		if kind, _ := p["kind"].(string); kind == "" || kind == "submarine" {
			p["depth_ft"] = 60.0
		}
	}
	units, _ := m["units"].([]any)
	for _, raw := range units {
		u, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if kind, _ := u["kind"].(string); kind != "submarine" {
			continue
		}
		u["depth_ft"] = 60.0
		delete(u, "depth_jitter")
	}
}

func patchAttributionEvents(m map[string]any) []any {
	events, _ := m["events"].([]any)
	byID := map[string]map[string]any{}
	order := make([]string, 0, len(events)+3)
	for _, raw := range events {
		ev, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := ev["id"].(string)
		if id == "intel_rendezvous_comm" || id == "intel_rendezvous_marker" || id == "enemy_contact_strait_marker" {
			continue
		}
		byID[id] = ev
		order = append(order, id)
	}
	byID["enemy_contact_comm"] = map[string]any{
		"id": "enemy_contact_comm",
		"when": map[string]any{
			"type": "enemy_prosecutes_allies",
		},
		"actions": []any{
			map[string]any{
				"type": "comm_schedule",
				"id":   "enemy_contact_intel",
				"text": map[string]string{
					"en": "INTEL: Hostile units have gained sonar contact on the coalition group and are closing. Rendezvous in the central strait.",
					"ru": "РАЗВЕДКА: Противник засёк группу коалиции и идёт на сближение. Рандеву — в центре пролива.",
				},
			},
			map[string]any{
				"type": "ally_sub_assist",
				"x":    7500.0,
				"y":    500.0,
			},
		},
	}
	if ev, ok := byID["tasking_attr"]; ok {
		actions, _ := ev["actions"].([]any)
		actions = append(actions, map[string]any{
			"type": "plot_marker",
			"id":   "strait_rendezvous",
			"name": map[string]string{
				"en": "RENDEZVOUS",
				"ru": "РАНДЕВУ",
			},
			"x": 7500.0,
			"y": 500.0,
		})
		ev["actions"] = actions
	}
	delete(byID, "enemy_contact_strait_marker")
	byID["enemy_contact_group_marker"] = map[string]any{
		"id": "enemy_contact_group_marker",
		"when": map[string]any{
			"type": "enemy_prosecutes_allies",
		},
		"actions": []any{
			map[string]any{
				"type": "plot_marker",
				"id":   "enemy_group",
				"name": map[string]string{
					"en": "HOSTILE GROUP",
					"ru": "ГРУППА ПРОТИВНИКА",
				},
				"x": 9200.0,
				"y": -600.0,
			},
		},
	}
	for _, id := range []string{"enemy_contact_comm", "enemy_contact_group_marker"} {
		if !containsStr(order, id) {
			order = append(order, id)
		}
	}
	out := make([]any, 0, len(order))
	for _, id := range order {
		if ev := byID[id]; ev != nil {
			out = append(out, ev)
		}
	}
	return out
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func bumpPatchVersion(v string) string {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return v
	}
	n := 0
	fmt.Sscanf(parts[2], "%d", &n)
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], n+1)
}

type landHit struct {
	x, y float64
	seg  int
}

func validateRoute(bathy *world.Bathymetry, r *world.Route) []landHit {
	if bathy == nil || r == nil {
		return nil
	}
	var bad []landHit
	n := r.UniqueCount()
	if n < 2 {
		return bad
	}
	for i := 1; i < n; i++ {
		a, b := r.Waypoints[i-1], r.Waypoints[i]
		sampleSegment(bathy, a.X, a.Y, b.X, b.Y, i-1, &bad)
	}
	if r.Looped {
		a, b := r.Waypoints[n-1], r.Waypoints[0]
		sampleSegment(bathy, a.X, a.Y, b.X, b.Y, n, &bad)
	} else if r.PingPong {
		a, b := r.Waypoints[n-1], r.Waypoints[n-2]
		sampleSegment(bathy, a.X, a.Y, b.X, b.Y, n-1, &bad)
	}
	return bad
}

func repairRouteLandHits(bathy *world.Bathymetry, r *world.Route) *world.Route {
	if bathy == nil || r == nil || len(r.Waypoints) < 2 {
		return r
	}
	const maxWP = 24
	for pass := 0; pass < 3 && len(r.Waypoints) < maxWP; pass++ {
		wps := append([]world.Waypoint(nil), r.Waypoints...)
		changed := false
		var out []world.Waypoint
		out = append(out, wps[0])
		for i := 1; i < len(wps); i++ {
			a, b := wps[i-1], wps[i]
			if len(sampleSegmentHits(bathy, a.X, a.Y, b.X, b.Y)) > 0 && len(out)+len(wps)-i < maxWP {
				mx, my := (a.X+b.X)/2, (a.Y+b.Y)/2
				mx, my = snapClear(bathy, mx, my)
				out = append(out, world.Waypoint{X: mx, Y: my})
				changed = true
			}
			out = append(out, b)
		}
		r.Waypoints = out
		if !changed {
			break
		}
	}
	for i := range r.Waypoints {
		wp := &r.Waypoints[i]
		wp.X, wp.Y = snapClear(bathy, wp.X, wp.Y)
	}
	return r
}

func sampleSegmentHits(bathy *world.Bathymetry, x0, y0, x1, y1 float64) []landHit {
	var bad []landHit
	sampleSegment(bathy, x0, y0, x1, y1, 0, &bad)
	return bad
}

func sampleSegment(bathy *world.Bathymetry, x0, y0, x1, y1 float64, seg int, bad *[]landHit) {
	dist := math.Hypot(x1-x0, y1-y0)
	steps := int(dist/200) + 1
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		if bathy.IsShoreBlocked(x, y) {
			*bad = append(*bad, landHit{x: x, y: y, seg: seg})
		}
	}
}
