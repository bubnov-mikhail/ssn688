package campaign

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const (
	maxScenarioJSONBytes = 50 << 20 // 50 MiB raw JSON
	maxCoverBytes        = 5 << 20
	maxBathyBytes        = 20 << 20
)

var bathyMagic = []byte("BATH")

// ParseScenarioJSON validates and converts a scenario document to ScenarioDef.
func ParseScenarioJSON(data []byte, source string) (ScenarioDef, error) {
	if len(data) == 0 {
		return ScenarioDef{}, fmt.Errorf("empty scenario file")
	}
	if len(data) > maxScenarioJSONBytes {
		return ScenarioDef{}, fmt.Errorf("scenario file too large (%d bytes)", len(data))
	}
	if err := scanUnsafeText(string(data)); err != nil {
		return ScenarioDef{}, err
	}

	var doc scenarioFile
	if err := decodeJSONStrict(data, &doc); err != nil {
		return ScenarioDef{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := validateScenarioDoc(&doc); err != nil {
		return ScenarioDef{}, err
	}

	sc := ScenarioDef{
		ID:                ScenarioID(doc.ID),
		Title:             doc.Title,
		Backstory:         doc.Backstory,
		PostscriptSuccess: doc.PostscriptSuccess,
		PostscriptFailure: doc.PostscriptFailure,
		FormatVersion:     ParseSemVer(doc.FormatVersion),
		Version:           ParseSemVer(doc.Version),
		MinGameVersion:    ParseSemVer(doc.MinGameVersion),
		SourcePath:        source,
		Events:            doc.Events,
	}
	if cover, key, err := resolveCover(doc.Cover, "scenario:"+doc.ID); err != nil {
		return ScenarioDef{}, err
	} else if len(cover) > 0 {
		sc.CoverData = cover
		sc.CoverCacheKey = key
	}

	for _, t := range doc.Theaters {
		tid := TheaterID(t.ID)
		if t.Bathy == nil || t.Bathy.DataB64 == "" {
			return ScenarioDef{}, fmt.Errorf("theater %q: bathy data_b64 required", t.ID)
		}
		raw, err := resolveBathy(t.Bathy)
		if err != nil {
			return ScenarioDef{}, fmt.Errorf("theater %q bathy: %w", t.ID, err)
		}
		b, err := world.LoadBathymetry(raw)
		if err != nil {
			return ScenarioDef{}, fmt.Errorf("theater %q bathy: %w", t.ID, err)
		}
		sc.Theaters = append(sc.Theaters, TheaterDef{ID: tid, Chart: &b})
	}

	for _, mj := range doc.Missions {
		m, err := convertMissionJSON(mj)
		if err != nil {
			return ScenarioDef{}, fmt.Errorf("mission %q: %w", mj.ID, err)
		}
		sc.Missions = append(sc.Missions, m)
	}
	sc.ApplyCompatibility()
	return sc, nil
}

func validateScenarioDoc(doc *scenarioFile) error {
	if doc.FormatVersion == "" {
		return fmt.Errorf("missing format_version")
	}
	if doc.Version == "" {
		return fmt.Errorf("missing version")
	}
	if doc.MinGameVersion == "" {
		return fmt.Errorf("missing min_game_version")
	}
	if doc.ID == "" || !validID(doc.ID) {
		return fmt.Errorf("invalid id %q", doc.ID)
	}
	if locTextEmpty(doc.Title) {
		return fmt.Errorf("missing title")
	}
	if len(doc.Theaters) == 0 {
		return fmt.Errorf("at least one theater required")
	}
	if len(doc.Missions) == 0 {
		return fmt.Errorf("at least one mission required")
	}
	for _, t := range doc.Theaters {
		if !validID(t.ID) {
			return fmt.Errorf("invalid theater id %q", t.ID)
		}
		if t.Bathy == nil || t.Bathy.DataB64 == "" {
			return fmt.Errorf("theater %q: bathy data_b64 required", t.ID)
		}
		if err := validateAssetBlob(t.Bathy, maxBathyBytes, true); err != nil {
			return fmt.Errorf("theater %q: %w", t.ID, err)
		}
	}
	if doc.Cover != nil {
		if err := validateAssetBlob(doc.Cover, maxCoverBytes, false); err != nil {
			return fmt.Errorf("cover: %w", err)
		}
	}
	for _, mj := range doc.Missions {
		if !validID(mj.ID) {
			return fmt.Errorf("invalid mission id %q", mj.ID)
		}
		if mj.TheaterID == "" {
			return fmt.Errorf("mission %q: missing theater_id", mj.ID)
		}
		if mj.Player.ID == "" {
			return fmt.Errorf("mission %q: missing player", mj.ID)
		}
		if mj.Cover != nil {
			if err := validateAssetBlob(mj.Cover, maxCoverBytes, false); err != nil {
				return fmt.Errorf("mission %q cover: %w", mj.ID, err)
			}
		}
		if mj.BriefMap != nil {
			if err := validateAssetBlob(mj.BriefMap, maxCoverBytes, false); err != nil {
				return fmt.Errorf("mission %q brief_map: %w", mj.ID, err)
			}
		}
	}
	return nil
}

func convertMissionJSON(mj missionJSON) (MissionDef, error) {
	m := MissionDef{
		ID:           MissionID(mj.ID),
		Title:        mj.Title,
		Description:  mj.Description,
		TheaterID:    TheaterID(mj.TheaterID),
		CommBriefing: mj.CommBriefing,
		DebriefLead:  mj.DebriefLead,
		Events:       mj.Events,
		EndAfterEvent: mj.EndAfterEvent,
	}
	startSec, err := world.ParseStartTimeHHMM(mj.StartTime)
	if err != nil {
		return MissionDef{}, fmt.Errorf("start_time: %w", err)
	}
	m.StartTimeSec = startSec
	if cover, key, err := resolveCover(mj.Cover, "mission:"+mj.ID); err != nil {
		return MissionDef{}, err
	} else if len(cover) > 0 {
		m.CoverData = cover
		m.CoverCacheKey = key
	}
	if briefMap, key, err := resolveCover(mj.BriefMap, "brief_map:"+mj.ID); err != nil {
		return MissionDef{}, err
	} else if len(briefMap) > 0 {
		m.BriefMapData = briefMap
		m.BriefMapCacheKey = key
	}
	for _, r := range mj.Routes {
		mode, err := parseRouteMode(r.Mode)
		if err != nil {
			return MissionDef{}, fmt.Errorf("route %q: %w", r.ID, err)
		}
		if len(r.Waypoints) < 2 {
			return MissionDef{}, fmt.Errorf("route %q: need at least 2 waypoints", r.ID)
		}
		wps := make([]world.Waypoint, len(r.Waypoints))
		for i, wp := range r.Waypoints {
			wps[i] = world.Waypoint{X: wp.X, Y: wp.Y}
		}
		m.Routes = append(m.Routes, RouteSpec{
			ID: r.ID, Mode: mode, Waypoints: wps, PlayerClearance: r.PlayerClearance,
		})
	}
	player, err := convertUnitJSON(mj.Player)
	if err != nil {
		return MissionDef{}, fmt.Errorf("player: %w", err)
	}
	m.Player = player
	for _, u := range mj.Units {
		unit, err := convertUnitJSON(u)
		if err != nil {
			return MissionDef{}, fmt.Errorf("unit %q: %w", u.ID, err)
		}
		m.Units = append(m.Units, unit)
	}
	for _, o := range mj.Objectives {
		m.Objectives = append(m.Objectives, ObjectiveTemplate{
			ID: o.ID, Description: o.Description, TargetID: o.TargetID,
			Primary: o.Primary, NeedIdentify: o.NeedIdentify, NeedDestroy: o.NeedDestroy, Hidden: o.Hidden,
			RequireVar: o.RequireVar, UnlessVar: o.UnlessVar,
		})
	}
	for _, o := range mj.Outputs {
		m.Outputs = append(m.Outputs, OutputRule{
			Key: o.Key, Value: o.Value,
			WhenPrimaryComplete: o.WhenPrimaryComplete,
			WhenObjectiveID:     o.WhenObjectiveID,
		})
	}
	for _, d := range mj.DebriefLines {
		m.DebriefLines = append(m.DebriefLines, DebriefLine{
			ObjectiveID: d.ObjectiveID, OnSuccess: d.OnSuccess, OnFail: d.OnFail,
		})
	}
	for _, c := range mj.CommSchedule {
		m.CommSchedule = append(m.CommSchedule, world.CommScheduledMessage{
			ID: c.ID, AtSec: c.AtSec, Text: c.Text.TT(),
		})
	}
	// Time-based COMM from events is merged at Instantiate with campaign vars.
	return m, nil
}

func convertUnitJSON(u unitJSON) (UnitSpec, error) {
	kind, err := parseEntityKind(u.Kind)
	if err != nil {
		return UnitSpec{}, err
	}
	side, err := parseSide(u.Side)
	if err != nil {
		return UnitSpec{}, err
	}
	spawn := SpawnMode(u.Spawn)
	if spawn == "" {
		spawn = SpawnOnRoute
	}
	return UnitSpec{
		ID: u.ID, Name: u.Name, Kind: kind, Side: side, SignatureID: u.SignatureID,
		LengthFt: u.LengthFt, SpeedKts: u.SpeedKts, DepthFt: u.DepthFt, DepthJitter: u.DepthJitter,
		HeadingDeg: u.HeadingDeg, AIState: u.AIState, Defcon: u.Defcon,
		CrewSkill: u.CrewSkill, CrewJitter: u.CrewJitter, Combatant: u.Combatant,
		Spawn: spawn, Corner: u.Corner, CornerInsetYd: u.CornerInsetYd,
		MinRouteYd: u.MinRouteYd, MaxRouteYd: u.MaxRouteYd,
		RouteID: u.RouteID, RouteFrac: u.RouteFrac,
		FallbackCorner: u.FallbackCorner, FallbackMinYd: u.FallbackMinYd, FallbackMaxYd: u.FallbackMaxYd,
		RequireVar: u.RequireVar, UnlessVar: u.UnlessVar, AllyIgnore: u.AllyIgnore, ExerciseTarget: u.ExerciseTarget,
		Payload: convertPayloadJSON(u.Payload),
	}, nil
}

func convertPayloadJSON(p *unitPayloadJSON) *UnitPayload {
	if p == nil {
		return nil
	}
	return &UnitPayload{
		Torpedoes: p.Torpedoes, Harpoons: p.Harpoons, CruiseMissiles: p.CruiseMissiles,
		ASWRockets: p.ASWRockets, ShipTubes: p.ShipTubes, ExerciseTorpedoes: p.ExerciseTorpedoes,
		RBU: p.RBU, SAM: p.SAM, CIWS: p.CIWS,
	}
}

func resolveCover(blob *assetBlobJSON, cacheKey string) ([]byte, string, error) {
	if blob == nil {
		return nil, "", nil
	}
	data, err := decodeAssetBlob(blob, maxCoverBytes, false)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", nil
	}
	return data, cacheKey, nil
}

func resolveBathy(blob *assetBlobJSON) ([]byte, error) {
	if blob == nil {
		return nil, fmt.Errorf("no bathy data")
	}
	data, err := decodeAssetBlob(blob, maxBathyBytes, true)
	if err != nil {
		return nil, err
	}
	if len(data) < 4 || !bytes.Equal(data[:4], bathyMagic) {
		return nil, fmt.Errorf("invalid BATH header")
	}
	return data, nil
}

func decodeAssetBlob(blob *assetBlobJSON, maxDecoded int, bathy bool) ([]byte, error) {
	if blob == nil {
		return nil, nil
	}
	if blob.UseDefault {
		return nil, fmt.Errorf("use_game_default is not supported; inline bathy with data_b64")
	}
	if blob.AssetRef != "" {
		return nil, fmt.Errorf("asset_ref is not supported; inline assets with data_b64")
	}
	if blob.DataB64 != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(blob.DataB64))
		if err != nil {
			return nil, fmt.Errorf("invalid base64: %w", err)
		}
		if len(raw) > maxDecoded {
			return nil, fmt.Errorf("decoded asset too large (%d bytes)", len(raw))
		}
		if bathy && (len(raw) < 4 || !bytes.Equal(raw[:4], bathyMagic)) {
			return nil, fmt.Errorf("invalid BATH header")
		}
		if !bathy && blob.Mime != "" && !allowedCoverMime(blob.Mime) {
			return nil, fmt.Errorf("unsupported cover mime %q", blob.Mime)
		}
		return raw, nil
	}
	return nil, nil
}

func validateAssetBlob(blob *assetBlobJSON, maxDecoded int, bathy bool) error {
	if blob == nil {
		return nil
	}
	if blob.UseDefault {
		return fmt.Errorf("use_game_default is not supported; inline bathy with data_b64")
	}
	if blob.AssetRef != "" {
		return fmt.Errorf("asset_ref is not supported; inline assets with data_b64")
	}
	if blob.DataB64 != "" {
		if len(blob.DataB64) > maxDecoded*2 {
			return fmt.Errorf("base64 payload too large")
		}
		_, err := decodeAssetBlob(blob, maxDecoded, bathy)
		return err
	}
	return nil
}

func allowedCoverMime(mime string) bool {
	switch strings.ToLower(mime) {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func validID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func scanUnsafeText(s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("invalid UTF-8")
	}
	lower := strings.ToLower(s)
	for _, needle := range []string{"<script", "javascript:", "\x00"} {
		if strings.Contains(lower, needle) {
			return fmt.Errorf("disallowed content in scenario file")
		}
	}
	return nil
}

func parseEntityKind(s string) (world.EntityKind, error) {
	switch strings.ToLower(s) {
	case "submarine":
		return world.KindSubmarine, nil
	case "surface_ship", "surface", "ship":
		return world.KindSurfaceShip, nil
	default:
		return 0, fmt.Errorf("unknown kind %q", s)
	}
}

func parseRouteMode(s string) (RouteMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "open", "one_way", "oneway":
		return RouteOpen, nil
	case "pingpong", "ping_pong", "patrol":
		return RoutePingPong, nil
	case "loop", "looped", "closed":
		return RouteLoop, nil
	default:
		return "", fmt.Errorf("unknown route mode %q (want open|pingpong|loop)", s)
	}
}

func parseSide(s string) (world.Side, error) {
	switch strings.ToLower(s) {
	case "player", "friendly", "ally":
		return world.SidePlayer, nil
	case "enemy", "hostile":
		return world.SideEnemy, nil
	case "neutral", "civilian":
		return world.SideNeutral, nil
	default:
		return 0, fmt.Errorf("unknown side %q", s)
	}
}
