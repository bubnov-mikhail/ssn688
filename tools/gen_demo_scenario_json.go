// Command gen_demo_scenario_json writes scenarios/demo_catalina.json.
// Cover/bathy are inlined from files or reused from an existing demo JSON.
//
//	go run ./tools/gen_demo_scenario_json.go [cover.jpg] [bathy.bin]
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

var scenarioRU map[string]string

func init() {
	data, err := os.ReadFile(filepath.Join("tools", "i18n_scenario_translations.json"))
	if err == nil {
		_ = json.Unmarshal(data, &scenarioRU)
	}
	if scenarioRU == nil {
		scenarioRU = map[string]string{}
	}
}

func loc(en string) map[string]any {
	ru := scenarioRU[en]
	if ru == "" {
		ru = en
	}
	return map[string]any{"en": en, "ru": ru}
}


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

func existingDemo() map[string]any {
	path := filepath.Join("scenarios", "demo_catalina.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc map[string]any
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	return doc
}

func coverAsset() map[string]any {
	if len(os.Args) > 1 {
		return encodeCoverAsset(os.Args[1])
	}
	if prev := existingDemo(); prev != nil {
		if c, ok := prev["cover"].(map[string]any); ok {
			return c
		}
	}
	return encodeCoverAsset(filepath.Join("assets", "scenarios", "demo_cover.jpg"))
}

func bathyAssetAndChart() (map[string]any, *world.Bathymetry) {
	path := filepath.Join("tools", "bathy_catalina.bin")
	if len(os.Args) > 2 {
		path = os.Args[2]
	}
	if data, err := os.ReadFile(path); err == nil {
		bathy, err := world.LoadBathymetry(data)
		if err != nil {
			panic(err)
		}
		return map[string]any{
			"mime":     "application/octet-stream",
			"data_b64": base64.StdEncoding.EncodeToString(data),
		}, &bathy
	}
	prev := existingDemo()
	if prev == nil {
		panic("need tools/bathy_catalina.bin or existing scenarios/demo_catalina.json")
	}
	theaters, _ := prev["theaters"].([]any)
	if len(theaters) == 0 {
		panic("existing demo missing theaters")
	}
	th, _ := theaters[0].(map[string]any)
	bathyObj, _ := th["bathy"].(map[string]any)
	b64, _ := bathyObj["data_b64"].(string)
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err)
	}
	bathy, err := world.LoadBathymetry(raw)
	if err != nil {
		panic(err)
	}
	return bathyObj, &bathy
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
	bathyAsset, bathy := bathyAssetAndChart()
	return map[string]any{
		"format_version":   "3.0.0",
		"version":          "1.3.0",
		"min_game_version": "1.0.0",
		"id":               "demo_catalina",
		"title":            "Shadows off Catalina",
		"backstory":        campaignBackstory(),
		"cover":            coverAsset(),
		"postscript_success": "# Campaign complete\n\nCOMSUBPAC acknowledges your report. Hostile units neutralized. " +
			"Los Angeles is released from the Catalina barrier.",
		"postscript_failure": "# Campaign failed\n\nCOMSUBPAC notes mission failure. The Catalina barrier is compromised. " +
			"A relief SSN is being routed to your datum.",
		"theaters": []any{
			map[string]any{"id": "catalina", "bathy": bathyAsset},
		},
		"missions": []any{
			demoMission1(bathy),
			demoMission2(bathy),
		},
	}
}

func campaignBackstory() string {
	return "" +
		"# Situation\n\n" +
		"Santa Catalina sits astride the approaches to Los Angeles and the busy coastwise " +
		"lanes that feed Southern California ports. In peacetime the water is crowded and " +
		"predictable. In a crisis it becomes a choke where commercial traffic, naval " +
		"presence, and political signaling collide inside a few dozen nautical miles.\n\n" +
		"Pacific Fleet order **44-7** assigns USS Los Angeles (SSN-688) to a covert barrier " +
		"patrol south of the island: hold station, stay quiet, and be ready if the " +
		"situation hardens.\n\n" +
		"## Background\n\n" +
		"Tensions across the Pacific have climbed through a sequence of incidents short of " +
		"open war — contested EEZ claims, aggressive shadowing of merchant shipping, and " +
		"competing naval deployments framed as \"exercises.\" Neither side wants a shooting " +
		"war off a major U.S. metro area; both want the other to blink first.\n\n" +
		"Catalina is useful precisely because it is close to home. A hostile presence here " +
		"threatens coastal shipping and raises the political cost of inaction. An American " +
		"response here risks escalation in view of civilian traffic and allied units sharing " +
		"the same water.\n\n" +
		"## Flashpoints\n\n" +
		"- **Sea lines of communication** — tankers and merchants still run the Catalina " +
		"corridor; any disruption is felt ashore within days\n" +
		"- **Covert access** — diesel boats can linger on battery near shipping noise and " +
		"deny clear attribution until it is too late\n" +
		"- **ASW pressure** — surface and air ASW can \"sanitize\" the approaches, or " +
		"provoke a misread if they press too hard on the wrong contact\n" +
		"- **Blue-on-blue risk** — friendly combatants and a sister SSN may share the belt; " +
		"**positive ID before weapons**\n" +
		"- **Political optics** — a public clash in Southern California waters forces " +
		"Washington and the adversary into irreversible choices\n\n" +
		"## Intent\n\n" +
		"Your barrier is meant to keep options open: detect and report, hold fire until " +
		"tasked, and preserve acoustic discretion while the crisis is still reversible. " +
		"If the order comes to engage, identification under dense civilian traffic is the " +
		"hard problem — not finding water to shoot in."
}

func demoMission1(bathy *world.Bathymetry) map[string]any {
	briefing := "" +
		"# FLASH — On station\n\n" +
		"**FROM:** COMSUBPAC  \n" +
		"**TO:** USS LOS ANGELES (SSN-688)\n\n" +
		"## Immediate\n\n" +
		"- Proceed to assigned OP AREA vicinity Santa Catalina Island\n" +
		"- Conduct covert patrol — remain undetected\n" +
		"- Do **not** engage until directed\n" +
		"- Come to communications depth and raise HF antenna for follow-on tasking"
	tasking := "" +
		"# IMMEDIATE — Execute\n\n" +
		"**FROM:** COMSUBPAC  \n" +
		"**TO:** USS LOS ANGELES (SSN-688)\n\n" +
		"## Primary\n\n" +
		"- Locate and **sink** hostile diesel submarine in your OP AREA\n\n" +
		"## Secondary\n\n" +
		"- Locate, positively identify, and **sink** hostile surface combatant\n" +
		"- Locate and positively identify tanker — **do not engage**\n\n" +
		"## ID criteria\n\n" +
		"- Visual via periscope inside **3000 yards**, or\n" +
		"- Acoustic fingerprint — **80%** harmonic match with library etalon held two minutes\n\n" +
		"## Restrictions\n\n" +
		"- Civilian shipping is **not** to be engaged\n" +
		"- Report completion via this channel"
	return map[string]any{
		"id":    "catalina_training",
		"title": "Santa Catalina Approaches",
		"description": "" +
			"# Mission\n\n" +
			"Establish covert patrol in assigned OP AREA south of Santa Catalina Island.\n\n" +
			"## Intel\n\n" +
			"- Diesel submarine activity along merchant transit lanes\n" +
			"- Inshore ASW surface units (Grisha-class expected)\n" +
			"- Dense neutral shipping — merchants, tanker, trawler\n\n" +
			"## Notes\n\n" +
			"Allied forces hold the eastern patrol belt. Remain undetected until follow-on tasking via COMM.",
		"theater_id": "catalina",
		"start_time": "04:30",
		"routes": []any{
			routeFrom(world.BuildNWSETransit(bathy, "route_grisha", -3500, 60), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_merchant", -1200, 50), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_tanker", 800, 50), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_trawler", 2200, 60), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_foxtrot", 4200, 50), true, "pingpong"),
			routeFrom(world.BuildAllyEdgePatrol(bathy, "route_ally_edge", 24), false, "pingpong"),
		},
		"player": map[string]any{
			"id": "player", "name": "USS Los Angeles", "kind": "submarine", "side": "player",
			"signature_id": "los_angeles", "length_ft": 360, "depth_ft": 60, "heading_deg": 45,
			"combatant": true, "spawn": "corner", "corner": "SW", "min_route_yd": 800, "max_route_yd": 3000,
		},
		"units": []any{
			unit("enemy_grisha", "Hostile Corvette", "surface_ship", "enemy", "grisha", 235, 14, 0, 0, "PATROL", 0, 60, 20, true, "route_grisha", 0.22, "", false, "", ""),
			unit("civ_merchant", "MV Pacific Star", "surface_ship", "neutral", "merchant", 520, 11, 0, 0, "CRUISE", 0, 0, 0, false, "route_merchant", 0.38, "", false, "", ""),
			unit("civ_tanker", "MT Horizon", "surface_ship", "neutral", "tanker", 900, 9, 0, 0, "CRUISE", 0, 0, 0, false, "route_tanker", 0.55, "", false, "", ""),
			unit("civ_trawler", "FV Northern Light", "surface_ship", "neutral", "fishing", 140, 7, 0, 0, "CRUISE", 0, 0, 0, false, "route_trawler", 0.70, "", false, "", ""),
			unit("enemy_foxtrot", "Hostile SS Foxtrot", "submarine", "enemy", "foxtrot", 300, 5, 100, 60, "PATROL", 0, 30, 10, true, "route_foxtrot", 0.45, "", false, "", ""),
			unit("ally_spruance", "USS Spruance", "surface_ship", "player", "spruance", 563, 12, 0, 0, "PATROL", 2, 70, 15, true, "route_ally_edge", 0.02, "SE", false, "", ""),
			unit("ally_688", "USS Bremerton", "submarine", "player", "los_angeles", 360, 5, 140, 40, "PATROL", 2, 75, 10, true, "route_ally_edge", 0.08, "SE", false, "", ""),
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
			map[string]any{
				"id": "tanker_cue_after_foxtrot",
				"when": map[string]any{"type": "objective_complete", "objective_id": "obj_foxtrot"},
				"actions": []any{
					map[string]any{
						"type": "comm_schedule",
						"id":   "tanker_cue",
						"text": "" +
							"# IMMEDIATE — Tanker cue\n\n" +
							"**FROM:** COMSUBPAC  \n" +
							"**TO:** USS LOS ANGELES (SSN-688)\n\n" +
							"## Foxtrot\n\n" +
							"- Hostile diesel kill **confirmed**\n\n" +
							"## Tanker track (do not engage)\n\n" +
							"- Contact **{{unit.civ_tanker.name}}** estimated at **{{unit.civ_tanker.pos}}**\n" +
							"- Course approx **{{unit.civ_tanker.course}}** true, speed ~**{{unit.civ_tanker.speed}}** kts\n" +
							"- Positively **identify** for secondary; weapons hold on merchants",
					},
				},
			},
		},
		"debrief_lead": "# After action\n\nCOMSUBPAC acknowledges your report. Hostile diesel submarine is confirmed sunk. " +
			"The Catalina barrier holds; Los Angeles remains on station.",
		"debrief_lines": []any{
			map[string]any{
				"objective_id": "obj_grisha",
				"on_success":   "## Surface combatant\n\nHostile surface combatant was positively identified and sunk. Inshore ASW sweeps in your area have ceased.",
				"on_fail":      "## Surface combatant\n\nHostile surface combatant was not both identified and sunk. The corvette remains a threat to merchant traffic along the Catalina lanes.",
			},
			map[string]any{
				"objective_id": "obj_tanker",
				"on_success":   "## Tanker search\n\nMT Horizon was located and positively identified. She was not engaged, in accordance with ROE.",
				"on_fail":      "## Tanker search\n\nTanker search incomplete — no positive identification. A large merchant contact may still be in the transit lanes.",
			},
		},
		"outputs": []any{
			map[string]any{"key": "foxtrot_neutralized", "value": "true", "when_objective_id": "obj_foxtrot"},
			map[string]any{"key": "grisha_neutralized", "value": "true", "when_objective_id": "obj_grisha"},
			map[string]any{"key": "tanker_identified", "value": "true", "when_objective_id": "obj_tanker"},
		},
	}
}

func demoMission2(bathy *world.Bathymetry) map[string]any {
	briefing := "" +
		"# FLASH — Barrier hold\n\n" +
		"**FROM:** COMSUBPAC  \n" +
		"**TO:** USS LOS ANGELES (SSN-688)\n\n" +
		"## Status\n\n" +
		"- Remain on Catalina barrier\n" +
		"- Foxtrot kill **confirmed**\n" +
		"- Expect hostile reaction force\n" +
		"- Raise COMM for follow-on tasking"
	taskingCore := "" +
		"# IMMEDIATE — Counterstroke\n\n" +
		"**FROM:** COMSUBPAC  \n" +
		"**TO:** USS LOS ANGELES (SSN-688)\n\n" +
		"## Primary\n\n" +
		"- Locate and **sink** hostile Kilo-class diesel entering from NW\n" +
		"- Locate, positively identify, and **sink** hostile Udaloy DDG approaching from east\n"
	taskingGrisha := "" +
		"# IMMEDIATE — Surviving Grisha\n\n" +
		"## Secondary\n\n" +
		"- Hostile Grisha corvette still active — identify and **sink**"
	taskingTankerSink := "" +
		"\n## Tanker (revised ROE)\n\n" +
		"- **Sink** tanker MT Horizon — now assessed hostile support\n" +
		"- Approx position **{{unit.civ_tanker.pos}}**, course **{{unit.civ_tanker.course}}** true\n"
	taskingTankerID := "" +
		"\n## Tanker\n\n" +
		"- Locate and positively **identify** tanker MT Horizon\n" +
		"- Stand by for weapons release\n"
	taskingTail := "" +
		"\n## ID criteria\n\n" +
		"- Visual via periscope inside **3000 yards**, or\n" +
		"- Acoustic fingerprint — **80%** harmonic match held two minutes\n\n" +
		"## Support\n\n" +
		"- USS Spruance holds eastern edge; will support on **hostile surface fire**\n" +
		"- Report completion via this channel"
	sinkOrder := "" +
		"# IMMEDIATE — Weapons free\n\n" +
		"**FROM:** COMSUBPAC  \n" +
		"**TO:** USS LOS ANGELES (SSN-688)\n\n" +
		"## Tanker\n\n" +
		"- MT Horizon positively identified\n" +
		"- **PRIMARY:** sink MT Horizon — weapons free on that contact only"

	return map[string]any{
		"id":    "catalina_counterstroke",
		"title": "Catalina Counterstroke",
		"description": "" +
			"# Mission\n\n" +
			"Hostile reaction after the Foxtrot kill.\n\n" +
			"## Threats\n\n" +
			"- Quieter **Kilo** probes the barrier from NW\n" +
			"- **Udaloy** closes from the east to pincer Los Angeles\n" +
			"- Surviving **Grisha** (if not sunk in mission 1) returns as a hardened threat\n\n" +
			"## Shipping picture\n\n" +
			"Dense merchant traffic remains on the Catalina lanes. Prior tanker tasking from " +
			"mission 1 still matters — expect further COMM on contacts of interest once you are " +
			"on station.\n\n" +
			"## Friendly\n\n" +
			"- Spruance remains eastern support (assists on hostile surface fire)\n" +
			"- Bremerton has been redirected",
		"theater_id": "catalina",
		"start_time": "18:45",
		"routes": []any{
			routeFrom(world.BuildNWSETransit(bathy, "route_grisha", -3500, 60), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_merchant", -1200, 50), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_tanker", 800, 50), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_trawler", 2200, 60), true, "pingpong"),
			routeFrom(world.BuildNWSETransit(bathy, "route_kilo", 3200, 50), true, "pingpong"),
			routeFrom(world.BuildApproachFromNE(bathy, "route_udaloy", 20), true, "open"),
			routeFrom(world.BuildAllyEdgePatrol(bathy, "route_ally_edge", 24), false, "pingpong"),
		},
		"player": map[string]any{
			"id": "player", "name": "USS Los Angeles", "kind": "submarine", "side": "player",
			"signature_id": "los_angeles", "length_ft": 360, "depth_ft": 160, "heading_deg": 45,
			"combatant": true, "spawn": "corner", "corner": "SW", "min_route_yd": 800, "max_route_yd": 3000,
		},
		"units": []any{
			unit("enemy_kilo", "Hostile SS Kilo", "submarine", "enemy", "kilo", 230, 4.5, 220, 40, "PATROL", 0, 55, 12, true, "route_kilo", 0.12, "", false, "", ""),
			unit("enemy_udaloy", "Hostile DDG Udaloy", "surface_ship", "enemy", "udaloy", 535, 16, 0, 0, "PATROL", 1, 70, 15, true, "route_udaloy", 0.08, "NE", false, "", ""),
			unit("enemy_grisha", "Hostile Corvette", "surface_ship", "enemy", "grisha", 235, 16, 0, 0, "PATROL", 2, 85, 8, true, "route_grisha", 0.22, "", false, "", "grisha_neutralized"),
			unit("civ_merchant", "MV Pacific Star", "surface_ship", "neutral", "merchant", 520, 11, 0, 0, "CRUISE", 0, 0, 0, false, "route_merchant", 0.38, "", false, "", ""),
			unit("civ_tanker", "MT Horizon", "surface_ship", "neutral", "tanker", 900, 9, 0, 0, "CRUISE", 0, 0, 0, false, "route_tanker", 0.55, "", true, "", ""),
			unit("civ_trawler", "FV Northern Light", "surface_ship", "neutral", "fishing", 140, 7, 0, 0, "CRUISE", 0, 0, 0, false, "route_trawler", 0.70, "", false, "", ""),
			unit("ally_spruance", "USS Spruance", "surface_ship", "player", "spruance", 563, 14, 0, 0, "PATROL", 2, 75, 12, true, "route_ally_edge", 0.02, "SE", false, "", ""),
		},
		"objectives": []any{
			map[string]any{"id": "obj_kilo", "description": "Locate and sink hostile Kilo diesel", "target_id": "enemy_kilo", "primary": true, "need_destroy": true},
			map[string]any{"id": "obj_udaloy", "description": "Identify and sink hostile Udaloy DDG", "target_id": "enemy_udaloy", "primary": true, "need_identify": true, "need_destroy": true},
			map[string]any{"id": "obj_grisha", "description": "Identify and sink surviving Grisha corvette", "target_id": "enemy_grisha", "need_identify": true, "need_destroy": true, "unless_var": "grisha_neutralized"},
			map[string]any{"id": "obj_tanker_sink_known", "description": "Sink tanker MT Horizon", "target_id": "civ_tanker", "primary": true, "need_destroy": true, "require_var": "tanker_identified"},
			map[string]any{"id": "obj_tanker_id", "description": "Locate and identify tanker MT Horizon", "target_id": "civ_tanker", "primary": true, "need_identify": true, "unless_var": "tanker_identified"},
			map[string]any{"id": "obj_tanker_sink_hidden", "description": "Sink tanker MT Horizon", "target_id": "civ_tanker", "primary": true, "need_destroy": true, "hidden": true, "unless_var": "tanker_identified"},
		},
		"comm_briefing": briefing,
		"events": []any{
			map[string]any{
				"id": "tasking_m2_core_sink",
				"when": map[string]any{"type": "time", "at_sec": 20, "require_var": "tanker_identified"},
				"actions": []any{
					map[string]any{"type": "comm_schedule", "id": "tasking_m2", "text": taskingCore + taskingTankerSink + taskingTail},
				},
			},
			map[string]any{
				"id": "tasking_m2_core_id",
				"when": map[string]any{"type": "time", "at_sec": 20, "unless_var": "tanker_identified"},
				"actions": []any{
					map[string]any{"type": "comm_schedule", "id": "tasking_m2", "text": taskingCore + taskingTankerID + taskingTail},
				},
			},
			map[string]any{
				"id": "tasking_m2_grisha_note",
				"when": map[string]any{"type": "time", "at_sec": 21, "unless_var": "grisha_neutralized"},
				"actions": []any{
					map[string]any{"type": "comm_schedule", "id": "tasking_m2_grisha", "text": taskingGrisha},
				},
			},
			map[string]any{
				"id": "tanker_id_reveal_sink",
				"when": map[string]any{"type": "objective_identified", "objective_id": "obj_tanker_id", "unless_var": "tanker_identified"},
				"actions": []any{
					map[string]any{"type": "reveal_objective", "objective_id": "obj_tanker_sink_hidden"},
					map[string]any{"type": "comm_schedule", "id": "tanker_sink_order", "text": sinkOrder},
				},
			},
		},
		"debrief_lead": "# After action\n\nCOMSUBPAC acknowledges your report. The Catalina barrier held against the hostile counterstroke.",
		"debrief_lines": []any{
			map[string]any{"objective_id": "obj_kilo", "on_success": "## Kilo\n\nHostile Kilo neutralized. Second diesel threat removed from the barrier.", "on_fail": "## Kilo\n\nHostile Kilo was not sunk. A diesel still threatens the Catalina lanes."},
			map[string]any{"objective_id": "obj_udaloy", "on_success": "## Udaloy\n\nUdaloy DDG identified and sunk. Eastern pincer broken.", "on_fail": "## Udaloy\n\nUdaloy was not both identified and sunk. Hostile surface ASW remains in the OP AREA."},
			map[string]any{"objective_id": "obj_grisha", "on_success": "## Grisha\n\nSurviving Grisha finally silenced.", "on_fail": "## Grisha\n\nGrisha corvette escaped or was not finished."},
			map[string]any{"objective_id": "obj_tanker_sink_known", "on_success": "## Tanker\n\nMT Horizon destroyed per revised ROE.", "on_fail": "## Tanker\n\nMT Horizon was not sunk."},
			map[string]any{"objective_id": "obj_tanker_id", "on_success": "## Tanker ID\n\nMT Horizon located and identified.", "on_fail": "## Tanker ID\n\nTanker still unidentified."},
			map[string]any{"objective_id": "obj_tanker_sink_hidden", "on_success": "## Tanker\n\nMT Horizon destroyed after positive ID.", "on_fail": "## Tanker\n\nMT Horizon was not sunk after identification."},
		},
	}
}

func routeFrom(r *world.Route, playerClearance bool, mode string) map[string]any {
	if r == nil {
		panic("nil route")
	}
	wps := make([]any, 0, len(r.Waypoints))
	for _, wp := range r.Waypoints {
		wps = append(wps, map[string]any{"x": float64(int(wp.X*10+0.5)) / 10, "y": float64(int(wp.Y*10+0.5)) / 10})
	}
	m := map[string]any{"id": r.ID, "mode": mode, "waypoints": wps}
	if playerClearance {
		m["player_clearance"] = true
	}
	return m
}

func unit(id, name, kind, side, sig string, lenFt, spd, depth, jitter float64, ai string, defcon int, skill, jitterC float64, combat bool, route string, frac float64, fbCorner string, allyIgnore bool, requireVar, unlessVar string) map[string]any {
	m := map[string]any{
		"id": id, "name": name, "kind": kind, "side": side, "signature_id": sig,
		"length_ft": lenFt, "speed_kts": spd, "ai_state": ai, "spawn": "route",
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
	if allyIgnore {
		m["ally_ignore"] = true
	}
	if requireVar != "" {
		m["require_var"] = requireVar
	}
	if unlessVar != "" {
		m["unless_var"] = unlessVar
	}
	return m
}
