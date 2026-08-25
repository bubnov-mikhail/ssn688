// Command gen_demo_scenario_json writes scenarios/demo_catalina.json.
// Cover/bathy are inlined from files; routes are generated as explicit waypoints
// via world.BuildNWSETransit / BuildAllyEdgePatrol (same as dump_demo_routes).
//
//	go run ./tools/gen_demo_scenario_json.go [cover.jpg] [bathy.bin]
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ssn688/sim/internal/campaign"
	"github.com/ssn688/sim/internal/world"
)

func main() {
	doc := demoDocument()
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	path := filepath.Join("scenarios", "demo_catalina.json")
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", path)
	_ = campaign.DemoScenarioID
}

func coverPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return filepath.Join("assets", "scenarios", "demo_cover.jpg")
}

func bathyPath() string {
	if len(os.Args) > 2 {
		return os.Args[2]
	}
	return filepath.Join("tools", "bathy_catalina.bin")
}

func encodeBathyAsset(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return map[string]any{
		"mime":     "application/octet-stream",
		"data_b64": base64.StdEncoding.EncodeToString(data),
	}
}

func encodeCoverAsset(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return map[string]any{
		"mime":     "image/jpeg",
		"data_b64": base64.StdEncoding.EncodeToString(data),
	}
}

func demoDocument() map[string]any {
	tasking := "" +
		"TOP SECRET // IMMEDIATE\n" +
		"FROM: COMSUBPAC\n" +
		"TO: USS LOS ANGELES (SSN-688)\n" +
		"BT\n" +
		"EXECUTE.\n" +
		"PRIMARY: LOCATE AND SINK HOSTILE DIESEL SUBMARINE IN YOUR OP AREA.\n" +
		"SECONDARY: LOCATE, POSITIVELY IDENTIFY, AND SINK HOSTILE SURFACE COMBATANT.\n" +
		"SECONDARY: LOCATE AND POSITIVELY IDENTIFY TANKER. DO NOT ENGAGE.\n" +
		"ID CRITERIA: VISUAL VIA PERISCOPE INSIDE 800 YARDS, OR ACOUSTIC FINGERPRINT — 80 PCT HARMONIC MATCH WITH LIBRARY ETALON HELD TWO MINUTES.\n" +
		"CIVILIAN SHIPPING IS NOT TO BE ENGAGED.\n" +
		"REPORT COMPLETION VIA THIS CHANNEL.\n" +
		"BT"
	briefing := "" +
		"TOP SECRET // FLASH\n" +
		"FROM: COMSUBPAC\n" +
		"TO: USS LOS ANGELES (SSN-688)\n" +
		"BT\n" +
		"PROCEED TO ASSIGNED OP AREA VICINITY SANTA CATALINA ISLAND.\n" +
		"CONDUCT COVERT PATROL. REMAIN UNDETECTED. DO NOT ENGAGE UNTIL DIRECTED.\n" +
		"COME TO COMMUNICATIONS DEPTH AND RAISE HF ANTENNA FOR FOLLOW-ON TASKING.\n" +
		"BT"
	bathyRaw, err := os.ReadFile(bathyPath())
	if err != nil {
		panic(err)
	}
	bathy, err := world.LoadBathymetry(bathyRaw)
	if err != nil {
		panic(err)
	}
	return map[string]any{
		"format_version":     "2.0.0",
		"version":            "1.1.0",
		"min_game_version":   "1.0.0",
		"id":                 "demo_catalina",
		"title":              "Shadows off Catalina",
		"backstory":          campaignBackstory(),
		"cover":              encodeCoverAsset(coverPath()),
		"postscript_success": "COMSUBPAC acknowledges your report. Hostile units neutralized without civilian losses.",
		"postscript_failure": "COMSUBPAC notes mission failure. The Catalina barrier is compromised.",
		"theaters": []any{
			map[string]any{
				"id":    "catalina",
				"bathy": encodeBathyAsset(bathyPath()),
			},
		},
		"missions": []any{demoMission(briefing, tasking, &bathy)},
	}
}

func campaignBackstory() string {
	return "Pacific Fleet operations order 44-7 places USS Los Angeles (SSN-688) on a covert " +
		"barrier patrol south of Santa Catalina Island. COMSUBPAC warns of a Foxtrot-class diesel " +
		"prowling merchant lanes and a Grisha corvette running inshore ASW sweeps. Allied Spruance " +
		"and a sister 688 boat patrol the eastern edge — identify before you shoot.\n\n" +
		"Maintain acoustic discretion until tasked. Neutral merchant traffic is dense; ROE requires " +
		"positive identification of all contacts before weapons release."
}

func demoMission(briefing, tasking string, bathy *world.Bathymetry) map[string]any {
	return map[string]any{
		"id":    "catalina_training",
		"title": "Santa Catalina Approaches",
		"description": "Establish covert patrol in assigned OP AREA south of Santa Catalina Island. " +
			"Intelligence indicates diesel submarine activity and inshore ASW surface units along merchant transit lanes.",
		"theater_id": "catalina",
		"routes": []any{
			routeFrom(world.BuildNWSETransit(bathy, "route_grisha", -3500, 60), true),
			routeFrom(world.BuildNWSETransit(bathy, "route_merchant", -1200, 50), true),
			routeFrom(world.BuildNWSETransit(bathy, "route_tanker", 800, 50), true),
			routeFrom(world.BuildNWSETransit(bathy, "route_trawler", 2200, 60), true),
			routeFrom(world.BuildNWSETransit(bathy, "route_foxtrot", 4200, 50), true),
			routeFrom(world.BuildAllyEdgePatrol(bathy, "route_ally_edge", 24), false),
		},
		"player": map[string]any{
			"id": "player", "name": "USS Los Angeles", "kind": "submarine", "side": "player",
			"signature_id": "los_angeles", "length_ft": 360, "depth_ft": 60, "heading_deg": 45,
			"combatant": true, "spawn": "corner", "corner": "SW", "min_route_yd": 800, "max_route_yd": 3000,
		},
		"units": []any{
			unit("enemy_grisha", "Hostile Corvette", "surface_ship", "enemy", "grisha", 235, 14, 0, 0, "PATROL", 0, 60, 20, true, "route_grisha", 0.22, "", ""),
			unit("civ_merchant", "MV Pacific Star", "surface_ship", "neutral", "merchant", 520, 11, 0, 0, "CRUISE", 0, 0, 0, false, "route_merchant", 0.38, "", ""),
			unit("civ_tanker", "MT Horizon", "surface_ship", "neutral", "tanker", 900, 9, 0, 0, "CRUISE", 0, 0, 0, false, "route_tanker", 0.55, "", ""),
			unit("civ_trawler", "FV Northern Light", "surface_ship", "neutral", "fishing", 140, 7, 0, 0, "CRUISE", 0, 0, 0, false, "route_trawler", 0.70, "", ""),
			unit("enemy_foxtrot", "Hostile SS Foxtrot", "submarine", "enemy", "foxtrot", 300, 5, 100, 60, "PATROL", 0, 30, 10, true, "route_foxtrot", 0.45, "", ""),
			unit("ally_spruance", "USS Spruance", "surface_ship", "player", "spruance", 563, 12, 0, 0, "PATROL", 2, 70, 15, true, "route_ally_edge", 0.02, "SE", ""),
			unit("ally_688", "USS Bremerton", "submarine", "player", "los_angeles", 360, 5, 140, 40, "PATROL", 2, 75, 10, true, "route_ally_edge", 0.08, "SE", ""),
		},
		"objectives": []any{
			map[string]any{"id": "obj_foxtrot", "description": "Locate and sink hostile diesel submarine", "target_id": "enemy_foxtrot", "primary": true, "need_destroy": true},
			map[string]any{"id": "obj_grisha", "description": "Identify and sink hostile surface combatant", "target_id": "enemy_grisha", "need_identify": true, "need_destroy": true},
			map[string]any{"id": "obj_tanker", "description": "Locate and identify tanker (do not engage)", "target_id": "civ_tanker", "need_identify": true},
		},
		"comm_briefing": briefing,
		"events": []any{
			map[string]any{
				"id": "tasking_engage",
				"when": map[string]any{"type": "time", "at_sec": 20},
				"actions": []any{
					map[string]any{"type": "comm_schedule", "id": "tasking_engage", "text": tasking},
				},
			},
		},
		"debrief_lead": "COMSUBPAC acknowledges your report. Hostile diesel submarine is confirmed sunk. The Catalina barrier holds; Los Angeles remains on station.",
		"debrief_lines": []any{
			map[string]any{"objective_id": "obj_grisha", "on_success": "Hostile surface combatant was positively identified and sunk. Inshore ASW sweeps in your area have ceased.", "on_fail": "Hostile surface combatant was not both identified and sunk. The corvette remains a threat to merchant traffic along the Catalina lanes."},
			map[string]any{"objective_id": "obj_tanker", "on_success": "Tanker search complete: MT Horizon was located and positively identified. She was not engaged, in accordance with ROE.", "on_fail": "Tanker search incomplete — no positive identification. A large merchant contact may still be in the transit lanes."},
		},
		"outputs": []any{
			map[string]any{"key": "foxrot_neutralized", "value": "true", "when_primary_complete": true},
			map[string]any{"key": "grisha_neutralized", "value": "true", "when_primary_complete": true},
		},
	}
}

func routeFrom(r *world.Route, playerClearance bool) map[string]any {
	if r == nil {
		panic("nil route")
	}
	wps := make([]any, 0, len(r.Waypoints))
	for _, wp := range r.Waypoints {
		wps = append(wps, map[string]any{"x": float64(int(wp.X*10+0.5)) / 10, "y": float64(int(wp.Y*10+0.5)) / 10})
	}
	m := map[string]any{"id": r.ID, "mode": "pingpong", "waypoints": wps}
	if playerClearance {
		m["player_clearance"] = true
	}
	return m
}

func unit(id, name, kind, side, sig string, len, spd, depth, jitter float64, ai string, defcon int, skill, jitterC float64, combat bool, route string, frac float64, fbCorner, _ string) map[string]any {
	m := map[string]any{
		"id": id, "name": name, "kind": kind, "side": side, "signature_id": sig,
		"length_ft": len, "speed_kts": spd, "ai_state": ai, "spawn": "route",
		"route_id": route, "route_frac": frac,
	}
	if depth > 0 {
		m["depth_ft"] = depth
	}
	if jitter > 0 {
		m["depth_jitter"] = jitter
	}
	if defcon > 0 {
		m["defcon"] = defcon
	}
	if skill > 0 {
		m["crew_skill"] = skill
		m["crew_jitter"] = jitterC
	}
	if combat {
		m["combatant"] = true
	}
	if fbCorner != "" {
		m["fallback_corner"] = fbCorner
	}
	return m
}
