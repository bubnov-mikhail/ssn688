package save

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bubnov-mikhail/ssn688/internal/acoustics"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/weapons"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const saveFormat = 15

// Save writes the simulation state to a plain-text file.
func Save(path string, engine *sim.Engine) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "# SSN688 Save File\n")
	fmt.Fprintf(w, "format=%d\n", saveFormat)
	fmt.Fprintf(w, "scenario=%s\n", engine.Scenario.Name)
	fmt.Fprintf(w, "fail_reason=%s\n", engine.Scenario.FailReason)
	fmt.Fprintf(w, "game_time=%.3f\n", engine.Clock.GameTime)
	fmt.Fprintf(w, "time_scale=%.3f\n", engine.Clock.TimeScale)
	fmt.Fprintf(w, "paused=%t\n", engine.Clock.Paused)
	fmt.Fprintf(w, "\n")

	writeEntity(w, engine.Scenario.Player)
	for _, e := range engine.Scenario.Entities {
		writeEntity(w, e)
	}

	env := engine.Acoustics.Env
	fmt.Fprintf(w, "\n[environment]\n")
	fmt.Fprintf(w, "weather=%d\n", int(engine.Scenario.Weather))
	fmt.Fprintf(w, "layer_survey_known=%t\n", env.LayerSurveyKnown)
	fmt.Fprintf(w, "layer_survey_start=%.3f\n", env.LayerSurveyStartAt)
	fmt.Fprintf(w, "layer_survey_end=%.3f\n", env.LayerSurveyEndAt)

	fmt.Fprintf(w, "\n[esm]\n")
	fmt.Fprintf(w, "order=%d\n", int(engine.ESM.Order))
	fmt.Fprintf(w, "extension=%.3f\n", engine.ESM.Extension)
	fmt.Fprintf(w, "sheared=%t\n", engine.ESM.Sheared)

	fmt.Fprintf(w, "\n[comm]\n")
	fmt.Fprintf(w, "order=%d\n", int(engine.COMM.Order))
	fmt.Fprintf(w, "extension=%.3f\n", engine.COMM.Extension)
	fmt.Fprintf(w, "sheared=%t\n", engine.COMM.Sheared)
	for id := range engine.COMM.DeliveredIDs {
		fmt.Fprintf(w, "delivered=%s\n", id)
	}
	for _, msg := range engine.COMM.Inbox {
		en := msg.Body[i18n.LangEN]
		if en == "" {
			en = msg.Body.GetText(i18n.LangEN)
		}
		ru := msg.Body[i18n.LangRU]
		fmt.Fprintf(w, "inbox=%.3f|%s|%s|%s\n", msg.TimeSec, msg.SourceID, escapeCommField(en), escapeCommField(ru))
	}

	fmt.Fprintf(w, "\n[peri]\n")
	fmt.Fprintf(w, "order=%d\n", int(engine.Periscope.Order))
	fmt.Fprintf(w, "extension=%.3f\n", engine.Periscope.Extension)
	fmt.Fprintf(w, "sheared=%t\n", engine.Periscope.Sheared)
	fmt.Fprintf(w, "train_rel=%.3f\n", engine.Periscope.TrainRelDeg)
	fmt.Fprintf(w, "zoom=%d\n", engine.Periscope.Zoom)
	fmt.Fprintf(w, "lock_id=%s\n", engine.Periscope.LockEntityID)

	fmt.Fprintf(w, "\n[sonar]\n")
	fmt.Fprintf(w, "passive_enabled=%t\n", engine.Sonar.PassiveEnabled)
	fmt.Fprintf(w, "active_enabled=%t\n", engine.Sonar.ActiveEnabled)
	fmt.Fprintf(w, "active_power=%.3f\n", engine.Sonar.ActivePower)
	fmt.Fprintf(w, "last_ping_time=%.3f\n", engine.Sonar.LastPingTime)
	fmt.Fprintf(w, "ping_interval=%.3f\n", engine.Sonar.PingInterval)
	fmt.Fprintf(w, "spectrum_bearing=%.3f\n", engine.Sonar.SpectrumBearing)
	fmt.Fprintf(w, "passive_array=%d\n", engine.Sonar.PassiveArray)
	fmt.Fprintf(w, "towed_cable_pct=%.3f\n", engine.Sonar.TowedCablePct)
	fmt.Fprintf(w, "towed_cable_rate=%.3f\n", engine.Sonar.TowedCableRate)
	fmt.Fprintf(w, "towed_damaged=%t\n", engine.Sonar.TowedDamaged)
	fmt.Fprintf(w, "listen_band=%d\n", engine.Sonar.ListenBand)
	fmt.Fprintf(w, "sonar_deaf_until=%.3f\n", engine.Sonar.SonarDeafUntil)
	fmt.Fprintf(w, "last_blast_detonate_at=%.3f\n", engine.Sonar.LastBlastDetonateAt)
	fmt.Fprintf(w, "last_blast_at=%.3f\n", engine.Sonar.LastBlastAt)
	fmt.Fprintf(w, "last_blast_x=%.3f\n", engine.Sonar.LastBlastX)
	fmt.Fprintf(w, "last_blast_y=%.3f\n", engine.Sonar.LastBlastY)
	fmt.Fprintf(w, "last_blast_range=%.3f\n", engine.Sonar.LastBlastRangeYd)
	fmt.Fprintf(w, "last_blast_flash=%.3f\n", engine.Sonar.LastBlastFlashSec)
	fmt.Fprintf(w, "last_blast_entity=%s\n", engine.Sonar.LastBlastEntityID)
	fmt.Fprintf(w, "contact_seq=%d\n", engine.Sonar.ContactSeq())
	for _, c := range engine.Sonar.Contacts {
		fmt.Fprintf(w, "contact=%s|%.3f|%.3f|%.3f|%s|%s|%.3f|%s|%s|%d|%s|%s|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%t|%s|%.3f|%.3f|%.3f\n",
			c.ID, c.BearingDeg, c.EstimatedRangeYd, c.SNR,
			c.BestMatchID, c.BestMatchName, c.Confidence,
			c.SourceEntityID, c.DetectedBy, c.Kind,
			c.ConfirmedID, c.ConfirmedClass,
			c.UncBearingDeg, c.UncRangeYd, c.LastUpdate, c.FirstSeen, c.ListenTime,
			c.LastActiveBearingDeg, c.LastActiveRangeYd, c.LastActiveFixAt,
			c.TMACourseDeg, c.TMASpeedKts, c.TMAAccuracy,
			c.Identified, c.IdentifiedBy, c.HarmonicMatch, c.HarmonicHoldSec, c.IdentifiedAt)
	}

	fmt.Fprintf(w, "\n[fire_control]\n")
	fmt.Fprintf(w, "selected_tube=%d\n", engine.FireControl.SelectedTube)
	fmt.Fprintf(w, "gyro_angle=%.3f\n", engine.FireControl.GyroAngleDeg)
	fmt.Fprintf(w, "run_depth=%.3f\n", engine.FireControl.RunDepthFt)
	fmt.Fprintf(w, "speed_setting=%s\n", engine.FireControl.SpeedSetting)
	fmt.Fprintf(w, "seeker_enabled=%t\n", engine.FireControl.SeekerEnabled)
	fmt.Fprintf(w, "magazine_left=%d\n", engine.FireControl.MagazineLeft)
	fmt.Fprintf(w, "harpoon_mag_left=%d\n", engine.FireControl.HarpoonMagLeft)
	fmt.Fprintf(w, "harpoon_radar_beam=%s\n", engine.FireControl.HarpoonRadarBeam)
	fmt.Fprintf(w, "harpoon_radar_range=%s\n", engine.FireControl.HarpoonRadarRange)
	fmt.Fprintf(w, "harpoon_destruct_range=%s\n", engine.FireControl.HarpoonDestructRange)
	fmt.Fprintf(w, "torpedo_seq=%d\n", engine.FireControl.TorpedoSeq())
	for id, n := range engine.FireControl.EnemyMagazine {
		fmt.Fprintf(w, "enemy_mag=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.AllyHarpoonMag {
		fmt.Fprintf(w, "ally_harpoon=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemyASCMMag {
		fmt.Fprintf(w, "enemy_ascm=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemyRastrub {
		fmt.Fprintf(w, "enemy_rastrub=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemyShipTube {
		fmt.Fprintf(w, "enemy_ship_tube=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemyExerciseTube {
		fmt.Fprintf(w, "enemy_exercise_tube=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemyRBU {
		fmt.Fprintf(w, "enemy_rbu=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemySAM {
		fmt.Fprintf(w, "enemy_sam=%s|%d\n", id, n)
	}
	for id, n := range engine.FireControl.EnemyCIWS {
		fmt.Fprintf(w, "enemy_ciws=%s|%d\n", id, n)
	}
	for id, t := range engine.FireControl.EnemyPDEngageAt {
		fmt.Fprintf(w, "enemy_pd_engage=%s|%.3f\n", id, t)
	}
	for id, t := range engine.FireControl.EnemyTubeOpenAt {
		fmt.Fprintf(w, "enemy_tube_open=%s|%.3f\n", id, t)
	}
	for _, t := range engine.FireControl.Tubes {
		fmt.Fprintf(w, "tube=%d|%d|%s|%t|%s|%.3f|%s|%s|%s\n",
			t.Number, t.State, t.TorpedoType, t.WireIntact, t.TorpedoID, t.ReloadEnds,
			t.TargetContactID, t.ReloadOrdnance, t.LastOrdnance)
	}
	for _, torp := range engine.FireControl.ActiveTorpedoes {
		fmt.Fprintf(w, "torpedo=%s|%s|%s|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%t|%t|%t|%t|%d|%.3f|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%t|%.3f|%t|%d|%s|%d|%s|%t\n",
			torp.ID, torp.ParentSubID, torp.TargetID, torp.Side,
			torp.X, torp.Y, torp.DepthFt, torp.HeadingDeg, torp.SpeedKts, torp.RunDepthFt,
			torp.SeekerOn, torp.WireCut, torp.Armed, torp.Alive, torp.Mode, torp.Age,
			torp.TubeNumber, torp.OrderedHead, torp.CruiseKts,
			torp.LaunchHeadDeg, torp.GyroCourseDeg, torp.ClearDistYd, torp.EnableSearchAfterClear,
			torp.LastPingTime, torp.GyroEnabled(), torp.Class, torp.OrdnanceType,
			torp.TerminalMode, torp.AcousticSig, torp.DisableSearch)
	}
	for _, a := range engine.FireControl.ActiveRastrub {
		if a == nil || !a.Alive {
			continue
		}
		fmt.Fprintf(w, "rastrub=%s|%s|%s|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%s\n",
			a.ID, a.ParentID, a.TargetID, a.Side,
			a.X0, a.Y0, a.X1, a.Y1, a.LaunchAt, a.FlightSec, a.RunDepthFt, a.ParentSig)
	}
	for _, a := range engine.FireControl.ActiveRBU {
		if a == nil || !a.Alive {
			continue
		}
		fmt.Fprintf(w, "rbu=%s|%s|%s|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f\n",
			a.ID, a.ParentID, a.TargetID, a.Side,
			a.X0, a.Y0, a.X1, a.Y1, a.LaunchAt, a.FlightSec)
	}
	for _, h := range engine.FireControl.ActiveHarpoons {
		if h == nil || (!h.Alive && !h.VisibleOnWEPS) {
			continue
		}
		fmt.Fprintf(w, "harpoon=%s|%s|%s|%d|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%d|%t|%t|%.3f|%.3f|%.3f|%.3f|%.3f|%t|%.3f|%.3f|%d\n",
			h.ID, h.ParentSubID, h.TargetContactID, h.Side, h.TubeNumber,
			h.LaunchX, h.LaunchY, h.X, h.Y, h.HeadingDeg, h.SpeedKts, h.DistanceYd, int(h.Phase),
			h.RadarOn, h.Alive, h.BeamHalfDeg, h.RadarRangeYd, h.DestructRangeYd,
			h.UnderwaterLeft, h.Age, h.VisibleOnWEPS, h.ProgrammedHead, h.AssumedDistanceYd, int(h.Variant))
	}

	fmt.Fprintf(w, "\n[cm]\n")
	for id, n := range engine.CM.Magazine {
		fmt.Fprintf(w, "cm_mag=%s|%d\n", id, n)
	}
	for id, n := range engine.CM.JitterMagazine {
		fmt.Fprintf(w, "jitter_mag=%s|%d\n", id, n)
	}
	for id, on := range engine.CM.NixieEnabled {
		fmt.Fprintf(w, "nixie=%s|%t\n", id, on)
	}
	for id, t := range engine.CM.LastDeployAt {
		fmt.Fprintf(w, "cm_deploy_at=%s|%.3f\n", id, t)
	}
	for id, t := range engine.CM.LastJitterAt {
		fmt.Fprintf(w, "jitter_deploy_at=%s|%.3f\n", id, t)
	}
	for _, cm := range engine.CM.Active {
		if cm == nil {
			continue
		}
		fmt.Fprintf(w, "cm=%s|%s|%d|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%t|%.3f|%.3f|%.3f\n",
			cm.ID, cm.ParentID, cm.Side, cm.Kind,
			cm.X, cm.Y, cm.DepthFt, cm.HeadingDeg, cm.SpeedKts,
			cm.Alive, cm.Age, cm.TTL, cm.NoiseBoostDB)
	}

	fmt.Fprintf(w, "\n[campaign]\n")
	writeCampaign(w, &engine.Campaign)

	fmt.Fprintf(w, "\n[objectives]\n")
	for _, o := range engine.Scenario.Objectives {
		desc := strings.ReplaceAll(o.Description.GetText(i18n.LangEN), "|", "/")
		fmt.Fprintf(w, "objective=%s|%s|%t|%s|%t|%t|%t|%t|%t\n",
			o.ID, desc, o.Complete, o.TargetID,
			o.Primary, o.NeedIdentify, o.NeedDestroy, o.Identified, o.Hidden)
	}

	fmt.Fprintf(w, "\n[plot_markers]\n")
	fmt.Fprintf(w, "marker_seq=%d\n", engine.PlotMarkerSeq())
	for _, m := range engine.PlotMarkers {
		fmt.Fprintf(w, "marker=%s|%.3f|%.3f\n", m.ID, m.X, m.Y)
	}
	return w.Flush()
}

func writeCampaign(w *bufio.Writer, meta *campaign.RuntimeMeta) {
	if meta == nil || meta.ScenarioID == "" {
		return
	}
	fmt.Fprintf(w, "scenario_id=%s\n", meta.ScenarioID)
	fmt.Fprintf(w, "mission_id=%s\n", meta.MissionID)
	fmt.Fprintf(w, "mission_hash=%s\n", meta.MissionHash)
	fmt.Fprintf(w, "loadout_mix=%.3f\n", meta.LoadoutMix)
	fmt.Fprintf(w, "report_eligible=%t\n", meta.ReportEligible)
	fmt.Fprintf(w, "between_missions=%t\n", meta.BetweenMissions)
	fmt.Fprintf(w, "debrief_pending=%t\n", meta.DebriefPending)
	fmt.Fprintf(w, "debrief_mission=%s\n", meta.DebriefMission)
	for _, o := range meta.DebriefOutcomes {
		fmt.Fprintf(w, "debrief_obj=%s|%t|%t|%t\n", o.ID, o.Identified, o.Destroyed, o.Complete)
	}
	var done []string
	for id, ok := range meta.Completed {
		if ok {
			done = append(done, string(id))
		}
	}
	sort.Strings(done)
	fmt.Fprintf(w, "completed=%s\n", strings.Join(done, ","))
	for k, v := range meta.Vars {
		fmt.Fprintf(w, "var=%s:%s\n", k, v)
	}
}

func applyCampaignField(meta *campaign.RuntimeMeta, key, val string) {
	if meta == nil {
		return
	}
	if meta.Completed == nil {
		meta.Completed = map[campaign.MissionID]bool{}
	}
	if meta.Vars == nil {
		meta.Vars = map[string]string{}
	}
	switch key {
	case "scenario_id":
		meta.ScenarioID = campaign.ScenarioID(val)
	case "mission_id":
		meta.MissionID = campaign.MissionID(val)
	case "mission_hash":
		meta.MissionHash = val
	case "loadout_mix":
		meta.LoadoutMix, _ = strconv.ParseFloat(val, 64)
	case "report_eligible":
		meta.ReportEligible, _ = strconv.ParseBool(val)
	case "between_missions":
		meta.BetweenMissions, _ = strconv.ParseBool(val)
	case "debrief_pending":
		meta.DebriefPending, _ = strconv.ParseBool(val)
	case "debrief_mission":
		meta.DebriefMission = campaign.MissionID(val)
	case "debrief_obj":
		if o, ok := parseDebriefObj(val); ok {
			meta.DebriefOutcomes = append(meta.DebriefOutcomes, o)
		}
	case "completed":
		for _, id := range strings.Split(val, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				meta.Completed[campaign.MissionID(id)] = true
			}
		}
	case "var":
		parts := strings.SplitN(val, ":", 2)
		if len(parts) == 2 {
			meta.Vars[parts[0]] = parts[1]
		}
	}
}

func parseDebriefObj(val string) (campaign.ObjectiveOutcome, bool) {
	parts := strings.Split(val, "|")
	if len(parts) < 4 || parts[0] == "" {
		return campaign.ObjectiveOutcome{}, false
	}
	id := parts[0]
	ident, _ := strconv.ParseBool(parts[1])
	dest, _ := strconv.ParseBool(parts[2])
	done, _ := strconv.ParseBool(parts[3])
	return campaign.ObjectiveOutcome{
		ID: id, Identified: ident, Destroyed: dest, Complete: done,
	}, true
}

func parseTrailingInt(id string) int {
	i := strings.LastIndexByte(id, '-')
	if i < 0 || i+1 >= len(id) {
		return 0
	}
	n, err := strconv.Atoi(id[i+1:])
	if err != nil {
		return 0
	}
	return n
}

// parseContactSeqNum extracts the numeric suffix from sonar IDs like C03 / E12.
func parseContactSeqNum(id string) int {
	i := 0
	for i < len(id) && (id[i] < '0' || id[i] > '9') {
		i++
	}
	if i >= len(id) {
		return 0
	}
	n, err := strconv.Atoi(id[i:])
	if err != nil {
		return 0
	}
	return n
}

func escapeCommField(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

func unescapeCommField(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	esc := false
	for _, r := range s {
		if esc {
			switch r {
			case 'n':
				b.WriteByte('\n')
			case '|', '\\':
				b.WriteRune(r)
			default:
				b.WriteRune(r)
			}
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		b.WriteRune(r)
	}
	if esc {
		b.WriteByte('\\')
	}
	return b.String()
}

// splitCommInboxFields splits on unescaped '|'.
func splitCommInboxFields(s string) []string {
	var parts []string
	var cur strings.Builder
	esc := false
	for _, r := range s {
		if esc {
			cur.WriteByte('\\')
			cur.WriteRune(r)
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		if r == '|' {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	if esc {
		cur.WriteByte('\\')
	}
	parts = append(parts, cur.String())
	return parts
}

func parseCommInboxLine(val string) (world.CommInboxEntry, bool) {
	parts := splitCommInboxFields(val)
	if len(parts) < 2 {
		return world.CommInboxEntry{}, false
	}
	tsec, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return world.CommInboxEntry{}, false
	}
	// Legacy: time|body (EN only, mirrored to RU)
	if len(parts) == 2 {
		txt := unescapeCommField(parts[1])
		// Old saves replaced '|' with '/' — leave as-is.
		return world.CommInboxEntry{TimeSec: tsec, Body: i18n.T(txt, txt)}, true
	}
	// Current: time|sourceID|en|ru
	if len(parts) >= 4 {
		en := unescapeCommField(parts[2])
		ru := unescapeCommField(parts[3])
		if ru == "" {
			ru = en
		}
		return world.CommInboxEntry{
			TimeSec:  tsec,
			SourceID: parts[1],
			Body:     i18n.T(en, ru),
		}, true
	}
	// time|sourceID|en
	en := unescapeCommField(parts[2])
	return world.CommInboxEntry{
		TimeSec:  tsec,
		SourceID: parts[1],
		Body:     i18n.T(en, en),
	}, true
}

func restoreMissionComm(engine *sim.Engine, m *campaign.MissionDef) {
	if engine == nil || engine.Scenario == nil || m == nil {
		return
	}
	briefing, schedule := campaign.RuntimeComm(m, engine.Scenario.Player, engine.Scenario.Entities, engine.Campaign.Vars)
	if engine.Scenario.CommBriefing == nil || engine.Scenario.CommBriefing.GetText(i18n.LangEN) == "" {
		engine.Scenario.CommBriefing = briefing
	}
	if len(engine.Scenario.CommSchedule) == 0 {
		schedule = campaign.AppendFiredEventComm(
			schedule, engine.Scenario.MissionEvents, engine.Scenario.FiredEventIDs,
			engine.Scenario.Player, engine.Scenario.Entities, engine.Scenario.StartTimeSec,
		)
		engine.Scenario.CommSchedule = schedule
	}
}

// rehydrateCommInboxLocales replaces EN-only inbox bodies with mission TT when
// the source (or time+EN match) is known — fixes old saves that stored English only.
func rehydrateCommInboxLocales(engine *sim.Engine) {
	if engine == nil || engine.Scenario == nil {
		return
	}
	bySrc := map[string]i18n.TranslatedText{}
	if engine.Scenario.CommBriefing.GetText(i18n.LangEN) != "" {
		bySrc["briefing"] = engine.Scenario.CommBriefing
	}
	for _, m := range engine.Scenario.CommSchedule {
		if m.ID != "" {
			bySrc[m.ID] = m.Text
		}
	}
	for i := range engine.COMM.Inbox {
		e := &engine.COMM.Inbox[i]
		if e.SourceID != "" {
			if tt, ok := bySrc[e.SourceID]; ok && tt.GetText(i18n.LangRU) != "" {
				e.Body = tt
				continue
			}
		}
		en := e.Body.GetText(i18n.LangEN)
		ru := e.Body[i18n.LangRU]
		if ru != "" && ru != en {
			continue // already bilingual
		}
		if e.TimeSec == 0 {
			if tt := bySrc["briefing"]; tt != nil && tt.GetText(i18n.LangEN) == en {
				e.Body = tt
				e.SourceID = "briefing"
				continue
			}
		}
		for _, m := range engine.Scenario.CommSchedule {
			if absFloat(m.AtSec-e.TimeSec) > 0.05 {
				continue
			}
			if m.Text.GetText(i18n.LangEN) == en && m.Text.GetText(i18n.LangRU) != "" {
				e.Body = m.Text
				e.SourceID = m.ID
				break
			}
		}
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func writeEntity(w *bufio.Writer, e *world.Entity) {
	fmt.Fprintf(w, "[entity:%s]\n", e.ID)
	fmt.Fprintf(w, "name=%s\n", e.Name)
	fmt.Fprintf(w, "kind=%d\n", e.Kind)
	fmt.Fprintf(w, "side=%d\n", e.Side)
	fmt.Fprintf(w, "status=%d\n", e.Status)
	fmt.Fprintf(w, "signature=%s\n", e.SignatureID)
	fmt.Fprintf(w, "x=%.3f\n", e.X)
	fmt.Fprintf(w, "y=%.3f\n", e.Y)
	fmt.Fprintf(w, "depth_ft=%.3f\n", e.DepthFt)
	fmt.Fprintf(w, "heading_deg=%.3f\n", e.HeadingDeg)
	fmt.Fprintf(w, "speed_kts=%.3f\n", e.SpeedKts)
	fmt.Fprintf(w, "ordered_speed=%.3f\n", e.OrderedSpeed)
	fmt.Fprintf(w, "ordered_depth=%.3f\n", e.OrderedDepth)
	fmt.Fprintf(w, "ordered_heading=%.3f\n", e.OrderedHead)
	fmt.Fprintf(w, "length_ft=%.3f\n", e.LengthFt)
	fmt.Fprintf(w, "active_sonar=%t\n", e.ActiveSonar)
	fmt.Fprintf(w, "last_ping_time=%.3f\n", e.LastPingTime)
	fmt.Fprintf(w, "last_ping_power=%.3f\n", e.LastPingPower)
	fmt.Fprintf(w, "ai_state=%s\n", e.AIState)
	fmt.Fprintf(w, "route_id=%s\n", e.RouteID)
	fmt.Fprintf(w, "route_wp=%d\n", e.RouteWP)
	fmt.Fprintf(w, "route_dir=%d\n", e.RouteDir)
	fmt.Fprintf(w, "route_need_resume=%t\n", e.RouteNeedResume)
	fmt.Fprintf(w, "defcon=%d\n", e.Defcon)
	fmt.Fprintf(w, "crew_skill=%.3f\n", e.CrewSkill)
	fmt.Fprintf(w, "ai_prosecuting=%t\n", e.AIProsecuting)
	fmt.Fprintf(w, "ai_lost_contact_sec=%.3f\n", e.AILostContactSec)
	fmt.Fprintf(w, "ai_engage_cooldown_until=%.3f\n", e.AIEngageCooldownUntil)
	fmt.Fprintf(w, "sink_rate_fpm=%.3f\n", e.SinkRateFPM)
	fmt.Fprintf(w, "wreck_noise_until=%.3f\n", e.WreckNoiseUntil)
	fmt.Fprintf(w, "cook_off_left=%d\n", e.CookOffLeft)
	fmt.Fprintf(w, "next_cook_off_at=%.3f\n", e.NextCookOffAt)
	fmt.Fprintf(w, "hull_fire_until=%.3f\n", e.HullFireUntil)
	fmt.Fprintf(w, "transient_until=%.3f\n", e.TransientUntil)
	fmt.Fprintf(w, "transient_freq=%.3f\n", e.TransientFreqHz)
	fmt.Fprintf(w, "transient_level=%.3f\n", e.TransientLevelDB)
	fmt.Fprintf(w, "torpedo_variant=%s\n", e.TorpedoVariant)
	fmt.Fprintf(w, "damage_init=%t\n", e.Damage.Initialized)
	fmt.Fprintf(w, "damage_repairing=%d\n", e.Damage.Repairing)
	fmt.Fprintf(w, "damage_runaway_fpm=%.3f\n", e.Damage.DepthRunawayFPM)
	fmt.Fprintf(w, "damage_steer_jam=%t\n", e.Damage.SteeringJammed)
	fmt.Fprintf(w, "damage_steer_deg=%.3f\n", e.Damage.SteeringJamDeg)
	for i := 0; i < world.SysCount; i++ {
		fmt.Fprintf(w, "damage_eff_%d=%.3f\n", i, e.Damage.Eff[i])
	}
	fmt.Fprintf(w, "\n")
}

// Load restores simulation state from a plain-text save file.
func Load(path string) (*sim.Engine, error) {
	return loadClean(path)
}

func loadClean(path string) (*sim.Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	sc := &world.Scenario{
		Name: "loaded",
		Player: &world.Entity{
			ID: "player", Kind: world.KindSubmarine, Side: world.SidePlayer,
			Status: world.StatusActive, SignatureID: "los_angeles", LengthFt: 360,
		},
	}
	engine := sim.NewEngine(sc)
	engine.Scenario.Entities = nil
	engine.Sonar.Contacts = nil
	engine.FireControl.ActiveTorpedoes = nil
	engine.FireControl.ActiveHarpoons = nil
	engine.FireControl.ActiveRastrub = nil
	engine.FireControl.ActiveRBU = nil
	engine.FireControl.EnemyMagazine = map[string]int{}
	engine.FireControl.EnemyASCMMag = map[string]int{}
	engine.FireControl.AllyHarpoonMag = map[string]int{}
	engine.FireControl.EnemyRastrub = map[string]int{}
	engine.FireControl.EnemyShipTube = map[string]int{}
	engine.FireControl.EnemyExerciseTube = map[string]int{}
	engine.FireControl.EnemyRBU = map[string]int{}
	engine.FireControl.EnemySAM = map[string]int{}
	engine.FireControl.EnemyCIWS = map[string]int{}
	engine.FireControl.EnemyPDEngageAt = map[string]float64{}
	engine.FireControl.EnemyTubeOpenAt = map[string]float64{}
	engine.CM = weapons.NewCountermeasureSystem()
	engine.CM.Active = nil
	engine.PlotMarkers = nil
	engine.COMM = acoustics.COMMState{}
	engine.ESM = acoustics.ESMState{}

	lines := strings.Split(string(data), "\n")
	section := ""
	var current *world.Entity

	objectivesLoaded := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			if strings.HasPrefix(section, "entity:") {
				id := strings.TrimPrefix(section, "entity:")
				if id == "player" {
					current = engine.Scenario.Player
				} else {
					current = &world.Entity{ID: id}
					engine.Scenario.Entities = append(engine.Scenario.Entities, current)
				}
			} else {
				current = nil
			}
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		if strings.HasPrefix(section, "entity:") && current != nil {
			applyEntityField(current, key, val)
			continue
		}

		switch section {
		case "environment":
			switch key {
			case "weather":
				n, _ := strconv.Atoi(val)
				engine.Scenario.Weather = world.Weather(n)
				engine.Acoustics.Env.SeaState = engine.Scenario.Weather.SeaStateInt()
			case "layer_survey_known":
				engine.Acoustics.Env.LayerSurveyKnown, _ = strconv.ParseBool(val)
			case "layer_survey_start":
				engine.Acoustics.Env.LayerSurveyStartAt, _ = strconv.ParseFloat(val, 64)
			case "layer_survey_end":
				engine.Acoustics.Env.LayerSurveyEndAt, _ = strconv.ParseFloat(val, 64)
			}
		case "esm":
			switch key {
			case "order":
				n, _ := strconv.Atoi(val)
				engine.ESM.Order = acoustics.ESMMastOrder(n)
			case "extension":
				engine.ESM.Extension, _ = strconv.ParseFloat(val, 64)
			case "sheared":
				engine.ESM.Sheared, _ = strconv.ParseBool(val)
			}
		case "comm":
			switch key {
			case "order":
				n, _ := strconv.Atoi(val)
				engine.COMM.Order = acoustics.COMMMastOrder(n)
			case "extension":
				engine.COMM.Extension, _ = strconv.ParseFloat(val, 64)
			case "sheared":
				engine.COMM.Sheared, _ = strconv.ParseBool(val)
			case "delivered":
				if engine.COMM.DeliveredIDs == nil {
					engine.COMM.DeliveredIDs = map[string]bool{}
				}
				engine.COMM.DeliveredIDs[val] = true
			case "inbox":
				if entry, ok := parseCommInboxLine(val); ok {
					engine.COMM.Inbox = append(engine.COMM.Inbox, entry)
				}
			}
		case "peri":
			switch key {
			case "order":
				n, _ := strconv.Atoi(val)
				engine.Periscope.Order = acoustics.PeriMastOrder(n)
			case "extension":
				engine.Periscope.Extension, _ = strconv.ParseFloat(val, 64)
			case "sheared":
				engine.Periscope.Sheared, _ = strconv.ParseBool(val)
			case "train_rel":
				engine.Periscope.TrainRelDeg, _ = strconv.ParseFloat(val, 64)
			case "zoom":
				n, _ := strconv.Atoi(val)
				if n < 0 {
					n = 0
				}
				if n >= acoustics.PeriZoomCount {
					n = acoustics.PeriZoomCount - 1
				}
				engine.Periscope.Zoom = n
			case "lock_id":
				engine.Periscope.LockEntityID = val
			}
		case "sonar":
			switch key {
			case "passive_enabled":
				engine.Sonar.PassiveEnabled, _ = strconv.ParseBool(val)
			case "active_enabled":
				engine.Sonar.ActiveEnabled, _ = strconv.ParseBool(val)
			case "active_power":
				engine.Sonar.ActivePower, _ = strconv.ParseFloat(val, 64)
			case "last_ping_time":
				engine.Sonar.LastPingTime, _ = strconv.ParseFloat(val, 64)
			case "ping_interval":
				engine.Sonar.PingInterval, _ = strconv.ParseFloat(val, 64)
			case "spectrum_bearing":
				engine.Sonar.SpectrumBearing, _ = strconv.ParseFloat(val, 64)
			case "passive_array":
				n, _ := strconv.Atoi(val)
				engine.Sonar.PassiveArray = acoustics.PassiveArrayKind(n)
			case "towed_cable_pct":
				engine.Sonar.TowedCablePct, _ = strconv.ParseFloat(val, 64)
			case "towed_cable_rate":
				engine.Sonar.TowedCableRate, _ = strconv.ParseFloat(val, 64)
			case "towed_damaged":
				engine.Sonar.TowedDamaged, _ = strconv.ParseBool(val)
			case "listen_band":
				n, _ := strconv.Atoi(val)
				engine.Sonar.ListenBand = acoustics.ListenBand(n)
			case "sonar_deaf_until":
				engine.Sonar.SonarDeafUntil, _ = strconv.ParseFloat(val, 64)
			case "last_blast_detonate_at":
				engine.Sonar.LastBlastDetonateAt, _ = strconv.ParseFloat(val, 64)
			case "last_blast_at":
				engine.Sonar.LastBlastAt, _ = strconv.ParseFloat(val, 64)
			case "last_blast_x":
				engine.Sonar.LastBlastX, _ = strconv.ParseFloat(val, 64)
			case "last_blast_y":
				engine.Sonar.LastBlastY, _ = strconv.ParseFloat(val, 64)
			case "last_blast_range":
				engine.Sonar.LastBlastRangeYd, _ = strconv.ParseFloat(val, 64)
			case "last_blast_flash":
				engine.Sonar.LastBlastFlashSec, _ = strconv.ParseFloat(val, 64)
			case "last_blast_entity":
				engine.Sonar.LastBlastEntityID = val
			case "contact_seq":
				n, _ := strconv.Atoi(val)
				engine.Sonar.SetContactSeq(n)
			case "contact":
				parseContact(&engine.Sonar, val)
			}
		case "fire_control":
			switch key {
			case "selected_tube":
				engine.FireControl.SelectedTube, _ = strconv.Atoi(val)
			case "gyro_angle":
				engine.FireControl.GyroAngleDeg, _ = strconv.ParseFloat(val, 64)
			case "run_depth":
				engine.FireControl.RunDepthFt, _ = strconv.ParseFloat(val, 64)
			case "speed_setting":
				engine.FireControl.SpeedSetting = val
			case "seeker_enabled":
				engine.FireControl.SeekerEnabled, _ = strconv.ParseBool(val)
			case "magazine_left":
				engine.FireControl.MagazineLeft, _ = strconv.Atoi(val)
			case "harpoon_mag_left":
				engine.FireControl.HarpoonMagLeft, _ = strconv.Atoi(val)
			case "harpoon_radar_beam":
				engine.FireControl.HarpoonRadarBeam = val
			case "harpoon_radar_range":
				engine.FireControl.HarpoonRadarRange = val
			case "harpoon_destruct_range":
				engine.FireControl.HarpoonDestructRange = val
			case "torpedo_seq":
				n, _ := strconv.Atoi(val)
				engine.FireControl.SetTorpedoSeq(n)
			case "enemy_mag":
				parseEnemyMag(&engine.FireControl, val)
			case "ally_harpoon":
				parseAllyHarpoon(&engine.FireControl, val)
			case "enemy_ascm":
				parseEnemyASCM(&engine.FireControl, val)
			case "enemy_rastrub", "enemy_asroc": // enemy_asroc: legacy
				parseEnemyRastrub(&engine.FireControl, val)
			case "enemy_ship_tube", "enemy_mk32": // enemy_mk32: legacy
				parseEnemyShipTube(&engine.FireControl, val)
			case "enemy_exercise_tube":
				parseEnemyExerciseTube(&engine.FireControl, val)
			case "enemy_rbu":
				parseEnemyRBU(&engine.FireControl, val)
			case "enemy_sam":
				parseEnemySAM(&engine.FireControl, val)
			case "enemy_ciws":
				parseEnemyCIWS(&engine.FireControl, val)
			case "enemy_pd_engage":
				parseEnemyPDEngage(&engine.FireControl, val)
			case "enemy_tube_open":
				parseEnemyTubeOpen(&engine.FireControl, val)
			case "tube":
				parseTube(&engine.FireControl, val)
			case "torpedo":
				parseTorpedo(&engine.FireControl, val)
			case "rastrub", "asroc": // asroc: legacy
				parseRastrub(&engine.FireControl, val)
			case "rbu":
				parseRBU(&engine.FireControl, val)
			case "harpoon":
				parseHarpoon(&engine.FireControl, val)
			}
		case "cm":
			switch key {
			case "cm_mag":
				parseCMMag(&engine.CM, val)
			case "jitter_mag":
				parseJitterMag(&engine.CM, val)
			case "nixie":
				parseNixie(&engine.CM, val)
			case "cm_deploy_at":
				parseCMDeployAt(&engine.CM, val)
			case "jitter_deploy_at":
				parseJitterDeployAt(&engine.CM, val)
			case "cm":
				parseCM(&engine.CM, val)
			}
		case "campaign":
			applyCampaignField(&engine.Campaign, key, val)
		case "objectives":
			if key == "objective" {
				if !objectivesLoaded {
					engine.Scenario.Objectives = nil
					objectivesLoaded = true
				}
				if obj, ok := parseObjective(val); ok {
					engine.Scenario.Objectives = append(engine.Scenario.Objectives, obj)
				}
			}
		case "plot_markers":
			switch key {
			case "marker_seq":
				n, _ := strconv.Atoi(val)
				engine.SetPlotMarkerSeq(n)
			case "marker":
				if m, ok := parsePlotMarker(val); ok {
					engine.PlotMarkers = append(engine.PlotMarkers, m)
					engine.SetPlotMarkerSeq(parseTrailingInt(m.ID))
				}
			}
		default:
			switch key {
			case "scenario":
				engine.Scenario.Name = val
			case "fail_reason":
				engine.Scenario.FailReason = val
			case "game_time":
				engine.Clock.GameTime, _ = strconv.ParseFloat(val, 64)
			case "time_scale":
				engine.Clock.TimeScale, _ = strconv.ParseFloat(val, 64)
			case "paused":
				engine.Clock.Paused, _ = strconv.ParseBool(val)
			}
		}
	}

	finalizeLoadedEntities(engine)
	if engine.Scenario != nil && engine.Scenario.Player != nil {
		engine.CM.EnsureMagazine(engine.Scenario.Player.ID)
	}
	if bathy := campaign.ResolveMissionBathy(engine.Campaign.ScenarioID, engine.Campaign.MissionID); bathy != nil && bathy.Valid() {
		engine.Scenario.Bathy = bathy
		engine.Acoustics.Bathy = bathy
	} else {
		return nil, fmt.Errorf(
			"save requires bathymetry from scenario %q mission %q (missing or incompatible)",
			engine.Campaign.ScenarioID, engine.Campaign.MissionID,
		)
	}
	if m := campaign.MissionByID(engine.Campaign.ScenarioID, engine.Campaign.MissionID); m != nil {
		engine.Scenario.StartTimeSec = m.StartTimeSec
		if len(engine.Scenario.Objectives) == 0 {
			engine.Scenario.Objectives = campaign.RuntimeObjectives(m.Objectives, engine.Campaign.Vars)
		}
		if len(engine.Scenario.MissionEvents) == 0 {
			events := campaign.FilterEvents(m.Events, engine.Campaign.Vars)
			engine.Scenario.MissionEvents = campaign.ToWorldEvents(events)
		}
		// Route geometry is not written to .sav; rebuild from the mission def
		// so debug overlays and AI FollowRoute keep working after load.
		if len(engine.Scenario.Routes) == 0 {
			engine.Scenario.Routes, _ = campaign.RuntimeRoutes(m.Routes)
		}
		restoreMissionComm(engine, m)
	} else if m := campaign.MissionByID(campaign.DemoScenarioID, campaign.DemoMissionTraining); m != nil {
		if engine.Scenario.StartTimeSec == 0 {
			engine.Scenario.StartTimeSec = m.StartTimeSec
		}
		if len(engine.Scenario.Objectives) == 0 {
			engine.Scenario.Objectives = campaign.RuntimeObjectives(m.Objectives, engine.Campaign.Vars)
		}
		if len(engine.Scenario.Routes) == 0 {
			engine.Scenario.Routes, _ = campaign.RuntimeRoutes(m.Routes)
		}
		restoreMissionComm(engine, m)
	}
	if engine.Scenario.FiredEventIDs == nil {
		engine.Scenario.FiredEventIDs = map[string]bool{}
	}
	for _, o := range engine.Scenario.Objectives {
		if o.ID == "obj_tanker_sink_hidden" && !o.Hidden {
			engine.Scenario.FiredEventIDs["tanker_id_reveal_sink"] = true
		}
	}
	rehydrateCommInboxLocales(engine)
	if len(engine.COMM.Inbox) == 0 {
		engine.COMM.SeedBriefing(engine.Scenario.CommBriefing)
	}
	return engine, nil
}

// LoadClean is an alias for Load.
func LoadClean(path string) (*sim.Engine, error) {
	return loadClean(path)
}

func finalizeLoadedEntities(engine *sim.Engine) {
	fix := func(e *world.Entity) {
		if e == nil {
			return
		}
		if e.LengthFt <= 0 {
			e.LengthFt = defaultLengthFt(e.SignatureID, e.Kind)
		}
		if e.Side != world.SideNeutral && !e.Damage.Initialized {
			e.Damage = world.NewFullHealth()
		}
	}
	fix(engine.Scenario.Player)
	for _, e := range engine.Scenario.Entities {
		fix(e)
	}
	for _, t := range engine.FireControl.ActiveTorpedoes {
		if t == nil {
			continue
		}
		engine.FireControl.SetTorpedoSeq(parseTrailingInt(t.ID))
		// Old saves: once past tube-clear distance, gyro steering already applied.
		if !t.GyroEnabled() && t.TubeCleared() {
			t.MarkGyroEnabled(true)
		}
	}
	for _, c := range engine.Sonar.Contacts {
		// Old saves omit contact_seq — raise from C01/E02-style IDs so new
		// detections do not reuse numbers already on the board.
		engine.Sonar.SetContactSeq(parseContactSeqNum(c.ID))
	}
	for _, a := range engine.FireControl.ActiveRastrub {
		if a == nil {
			continue
		}
		engine.FireControl.SetTorpedoSeq(parseTrailingInt(a.ID))
	}
	for _, a := range engine.FireControl.ActiveRBU {
		if a == nil {
			continue
		}
		engine.FireControl.SetTorpedoSeq(parseTrailingInt(a.ID))
	}
	for _, h := range engine.FireControl.ActiveHarpoons {
		if h == nil {
			continue
		}
		engine.FireControl.SetTorpedoSeq(parseTrailingInt(h.ID))
	}
	if engine.FireControl.HarpoonRadarBeam == "" {
		engine.FireControl.HarpoonMagLeft = weapons.PlayerHarpoonMagazine
		engine.FireControl.HarpoonRadarBeam = weapons.HarpoonBeamWide
		engine.FireControl.HarpoonRadarRange = weapons.HarpoonSRCHMedium
		engine.FireControl.HarpoonDestructRange = weapons.HarpoonDSTRLong
	}
}

func defaultLengthFt(sig string, kind world.EntityKind) float64 {
	switch sig {
	case "los_angeles":
		return 360
	case "kilo":
		return 240
	case "victor_iii":
		return 335
	case "yasen_m":
		return 456
	case "foxtrot":
		return 300
	case "udaloy":
		return 535
	case "spruance":
		return 563
	case "gorshkov":
		return 443
	case "krivak":
		return 405
	case "kresta2":
		return 520
	case "grisha":
		return 235
	case "merchant":
		return 520
	case "tanker":
		return 900
	case "fishing":
		return 140
	case "mk48", "type53", "umgt1", "set40", "mk46":
		return 19
	}
	switch kind {
	case world.KindSubmarine:
		return 300
	case world.KindSurfaceShip:
		return 400
	case world.KindTorpedo:
		return 19
	default:
		return 200
	}
}

func applyEntityField(e *world.Entity, key, val string) {
	switch key {
	case "name":
		e.Name = val
	case "kind":
		n, _ := strconv.Atoi(val)
		e.Kind = world.EntityKind(n)
	case "side":
		n, _ := strconv.Atoi(val)
		e.Side = world.Side(n)
	case "status":
		n, _ := strconv.Atoi(val)
		e.Status = world.Status(n)
	case "signature":
		e.SignatureID = val
	case "x":
		e.X, _ = strconv.ParseFloat(val, 64)
	case "y":
		e.Y, _ = strconv.ParseFloat(val, 64)
	case "depth_ft":
		e.DepthFt, _ = strconv.ParseFloat(val, 64)
	case "heading_deg":
		e.HeadingDeg, _ = strconv.ParseFloat(val, 64)
	case "speed_kts":
		e.SpeedKts, _ = strconv.ParseFloat(val, 64)
	case "ordered_speed":
		e.OrderedSpeed, _ = strconv.ParseFloat(val, 64)
	case "ordered_depth":
		e.OrderedDepth, _ = strconv.ParseFloat(val, 64)
	case "ordered_heading":
		e.OrderedHead, _ = strconv.ParseFloat(val, 64)
	case "length_ft":
		e.LengthFt, _ = strconv.ParseFloat(val, 64)
	case "active_sonar":
		e.ActiveSonar, _ = strconv.ParseBool(val)
	case "last_ping_time":
		e.LastPingTime, _ = strconv.ParseFloat(val, 64)
	case "last_ping_power":
		e.LastPingPower, _ = strconv.ParseFloat(val, 64)
	case "ai_state":
		e.AIState = val
	case "route_id":
		e.RouteID = val
	case "route_wp":
		e.RouteWP, _ = strconv.Atoi(val)
	case "route_dir":
		e.RouteDir, _ = strconv.Atoi(val)
	case "route_need_resume":
		e.RouteNeedResume, _ = strconv.ParseBool(val)
	case "defcon":
		e.Defcon, _ = strconv.Atoi(val)
	case "crew_skill":
		e.CrewSkill, _ = strconv.ParseFloat(val, 64)
	case "ai_prosecuting":
		e.AIProsecuting, _ = strconv.ParseBool(val)
	case "ai_lost_contact_sec":
		e.AILostContactSec, _ = strconv.ParseFloat(val, 64)
	case "ai_engage_cooldown_until":
		e.AIEngageCooldownUntil, _ = strconv.ParseFloat(val, 64)
	case "sink_rate_fpm":
		e.SinkRateFPM, _ = strconv.ParseFloat(val, 64)
	case "wreck_noise_until":
		e.WreckNoiseUntil, _ = strconv.ParseFloat(val, 64)
	case "cook_off_left":
		e.CookOffLeft, _ = strconv.Atoi(val)
	case "next_cook_off_at":
		e.NextCookOffAt, _ = strconv.ParseFloat(val, 64)
	case "hull_fire_until":
		e.HullFireUntil, _ = strconv.ParseFloat(val, 64)
	case "transient_until":
		e.TransientUntil, _ = strconv.ParseFloat(val, 64)
	case "transient_freq":
		e.TransientFreqHz, _ = strconv.ParseFloat(val, 64)
	case "transient_level":
		e.TransientLevelDB, _ = strconv.ParseFloat(val, 64)
	case "torpedo_variant":
		e.TorpedoVariant = val
	case "damage_init":
		e.Damage.Initialized, _ = strconv.ParseBool(val)
	case "damage_repairing":
		e.Damage.Repairing, _ = strconv.Atoi(val)
	case "damage_runaway_fpm":
		e.Damage.DepthRunawayFPM, _ = strconv.ParseFloat(val, 64)
	case "damage_steer_jam":
		e.Damage.SteeringJammed, _ = strconv.ParseBool(val)
	case "damage_steer_deg":
		e.Damage.SteeringJamDeg, _ = strconv.ParseFloat(val, 64)
	default:
		if strings.HasPrefix(key, "damage_eff_") {
			idx, err := strconv.Atoi(strings.TrimPrefix(key, "damage_eff_"))
			if err == nil && idx >= 0 && idx < world.SysCount {
				e.Damage.Eff[idx], _ = strconv.ParseFloat(val, 64)
				e.Damage.Initialized = true
			}
		}
	}
}

func parseContact(sonar *acoustics.SonarState, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 9 {
		return
	}
	bear, _ := strconv.ParseFloat(parts[1], 64)
	rng, _ := strconv.ParseFloat(parts[2], 64)
	snr, _ := strconv.ParseFloat(parts[3], 64)
	conf, _ := strconv.ParseFloat(parts[6], 64)
	c := acoustics.Contact{
		ID: parts[0], BearingDeg: bear, EstimatedRangeYd: rng, SNR: snr,
		BestMatchID: parts[4], BestMatchName: parts[5], Confidence: conf,
		SourceEntityID: parts[7], DetectedBy: parts[8],
	}
	if len(parts) > 9 {
		kind, _ := strconv.Atoi(parts[9])
		c.Kind = world.EntityKind(kind)
	}
	if len(parts) > 10 {
		c.ConfirmedID = parts[10]
	}
	if len(parts) > 11 {
		c.ConfirmedClass = parts[11]
	}
	if len(parts) > 12 {
		c.UncBearingDeg, _ = strconv.ParseFloat(parts[12], 64)
	}
	if len(parts) > 13 {
		c.UncRangeYd, _ = strconv.ParseFloat(parts[13], 64)
	}
	if len(parts) > 14 {
		c.LastUpdate, _ = strconv.ParseFloat(parts[14], 64)
	}
	if len(parts) > 15 {
		c.FirstSeen, _ = strconv.ParseFloat(parts[15], 64)
	}
	if len(parts) > 16 {
		c.ListenTime, _ = strconv.ParseFloat(parts[16], 64)
	}
	if len(parts) > 17 {
		c.LastActiveBearingDeg, _ = strconv.ParseFloat(parts[17], 64)
	}
	if len(parts) > 18 {
		c.LastActiveRangeYd, _ = strconv.ParseFloat(parts[18], 64)
	}
	if len(parts) > 19 {
		c.LastActiveFixAt, _ = strconv.ParseFloat(parts[19], 64)
	}
	if len(parts) > 20 {
		c.TMACourseDeg, _ = strconv.ParseFloat(parts[20], 64)
	}
	if len(parts) > 21 {
		c.TMASpeedKts, _ = strconv.ParseFloat(parts[21], 64)
	}
	if len(parts) > 22 {
		c.TMAAccuracy, _ = strconv.ParseFloat(parts[22], 64)
	}
	if len(parts) > 23 {
		c.Identified, _ = strconv.ParseBool(parts[23])
	}
	if len(parts) > 24 {
		c.IdentifiedBy = parts[24]
	}
	if len(parts) > 25 {
		c.HarmonicMatch, _ = strconv.ParseFloat(parts[25], 64)
	}
	if len(parts) > 26 {
		c.HarmonicHoldSec, _ = strconv.ParseFloat(parts[26], 64)
	}
	if len(parts) > 27 {
		c.IdentifiedAt, _ = strconv.ParseFloat(parts[27], 64)
	}
	sonar.Contacts = append(sonar.Contacts, c)
}

func parseTube(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 4 {
		return
	}
	num, _ := strconv.Atoi(parts[0])
	state, _ := strconv.Atoi(parts[1])
	wire, _ := strconv.ParseBool(parts[3])
	if num >= 1 && num <= 4 {
		t := weapons.Tube{
			Number: num, State: weapons.TubeState(state),
			TorpedoType: parts[2], WireIntact: wire,
		}
		if len(parts) > 4 {
			t.TorpedoID = parts[4]
		}
		if len(parts) > 5 {
			t.ReloadEnds, _ = strconv.ParseFloat(parts[5], 64)
		}
		if len(parts) > 6 {
			t.TargetContactID = parts[6]
		}
		if len(parts) > 7 {
			t.ReloadOrdnance = parts[7]
		}
		if len(parts) > 8 {
			t.LastOrdnance = parts[8]
		}
		fc.Tubes[num-1] = t
	}
}

func parseEnemyMag(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyMagazine == nil {
		fc.EnemyMagazine = map[string]int{}
	}
	fc.EnemyMagazine[parts[0]] = n
}

func parseAllyHarpoon(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.AllyHarpoonMag == nil {
		fc.AllyHarpoonMag = map[string]int{}
	}
	fc.AllyHarpoonMag[parts[0]] = n
}

func parseEnemyASCM(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyASCMMag == nil {
		fc.EnemyASCMMag = map[string]int{}
	}
	fc.EnemyASCMMag[parts[0]] = n
}

func parseEnemyRastrub(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyRastrub == nil {
		fc.EnemyRastrub = map[string]int{}
	}
	fc.EnemyRastrub[parts[0]] = n
}

func parseEnemyShipTube(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyShipTube == nil {
		fc.EnemyShipTube = map[string]int{}
	}
	fc.EnemyShipTube[parts[0]] = n
}

func parseEnemyExerciseTube(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyExerciseTube == nil {
		fc.EnemyExerciseTube = map[string]int{}
	}
	fc.EnemyExerciseTube[parts[0]] = n
}

func parseEnemyRBU(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyRBU == nil {
		fc.EnemyRBU = map[string]int{}
	}
	fc.EnemyRBU[parts[0]] = n
}

func parseEnemySAM(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemySAM == nil {
		fc.EnemySAM = map[string]int{}
	}
	fc.EnemySAM[parts[0]] = n
}

func parseEnemyCIWS(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	if fc.EnemyCIWS == nil {
		fc.EnemyCIWS = map[string]int{}
	}
	fc.EnemyCIWS[parts[0]] = n
}

func parseEnemyPDEngage(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	t, _ := strconv.ParseFloat(parts[1], 64)
	if fc.EnemyPDEngageAt == nil {
		fc.EnemyPDEngageAt = map[string]float64{}
	}
	fc.EnemyPDEngageAt[parts[0]] = t
}

func parseEnemyTubeOpen(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	t, _ := strconv.ParseFloat(parts[1], 64)
	if fc.EnemyTubeOpenAt == nil {
		fc.EnemyTubeOpenAt = map[string]float64{}
	}
	fc.EnemyTubeOpenAt[parts[0]] = t
}

func parseTorpedo(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 16 {
		return
	}
	side, _ := strconv.Atoi(parts[3])
	mode, _ := strconv.Atoi(parts[14])
	torp := &weapons.Torpedo{
		ID: parts[0], ParentSubID: parts[1], TargetID: parts[2],
		Side: world.Side(side),
	}
	torp.X, _ = strconv.ParseFloat(parts[4], 64)
	torp.Y, _ = strconv.ParseFloat(parts[5], 64)
	torp.DepthFt, _ = strconv.ParseFloat(parts[6], 64)
	torp.HeadingDeg, _ = strconv.ParseFloat(parts[7], 64)
	torp.SpeedKts, _ = strconv.ParseFloat(parts[8], 64)
	torp.RunDepthFt, _ = strconv.ParseFloat(parts[9], 64)
	torp.SeekerOn, _ = strconv.ParseBool(parts[10])
	torp.WireCut, _ = strconv.ParseBool(parts[11])
	torp.Armed, _ = strconv.ParseBool(parts[12])
	torp.Alive, _ = strconv.ParseBool(parts[13])
	torp.Mode = weapons.TorpedoMode(mode)
	torp.Age, _ = strconv.ParseFloat(parts[15], 64)
	torp.OrderedHead = torp.HeadingDeg
	torp.LastPingTime = -1
	if len(parts) > 16 {
		torp.TubeNumber, _ = strconv.Atoi(parts[16])
	}
	if len(parts) > 17 {
		torp.OrderedHead, _ = strconv.ParseFloat(parts[17], 64)
	}
	if len(parts) > 18 {
		torp.CruiseKts, _ = strconv.ParseFloat(parts[18], 64)
	}
	if torp.CruiseKts <= 0 {
		torp.CruiseKts = torp.SpeedKts
	}
	if len(parts) > 19 {
		torp.LaunchHeadDeg, _ = strconv.ParseFloat(parts[19], 64)
	} else {
		torp.LaunchHeadDeg = torp.HeadingDeg
	}
	if len(parts) > 20 {
		torp.GyroCourseDeg, _ = strconv.ParseFloat(parts[20], 64)
	} else {
		torp.GyroCourseDeg = torp.OrderedHead
	}
	if len(parts) > 21 {
		torp.ClearDistYd, _ = strconv.ParseFloat(parts[21], 64)
	}
	if len(parts) > 22 {
		torp.EnableSearchAfterClear, _ = strconv.ParseBool(parts[22])
	}
	if len(parts) > 23 {
		torp.LastPingTime, _ = strconv.ParseFloat(parts[23], 64)
	}
	if len(parts) > 24 {
		gy, _ := strconv.ParseBool(parts[24])
		torp.MarkGyroEnabled(gy)
	}
	if len(parts) > 25 {
		cl, _ := strconv.Atoi(parts[25])
		torp.Class = weapons.WeaponClass(cl)
	}
	if len(parts) > 26 {
		torp.OrdnanceType = parts[26]
	}
	if len(parts) > 27 {
		mode, _ := strconv.Atoi(parts[27])
		torp.TerminalMode = weapons.TorpedoTerminalMode(mode)
	}
	if len(parts) > 28 {
		torp.AcousticSig = parts[28]
	}
	if len(parts) > 29 {
		torp.DisableSearch, _ = strconv.ParseBool(parts[29])
	}
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
}

func parseRastrub(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 11 {
		return
	}
	side, _ := strconv.Atoi(parts[3])
	a := &weapons.RastrubFlight{
		ID: parts[0], ParentID: parts[1], TargetID: parts[2],
		Side:  world.Side(side),
		Alive: true,
	}
	a.X0, _ = strconv.ParseFloat(parts[4], 64)
	a.Y0, _ = strconv.ParseFloat(parts[5], 64)
	a.X1, _ = strconv.ParseFloat(parts[6], 64)
	a.Y1, _ = strconv.ParseFloat(parts[7], 64)
	a.LaunchAt, _ = strconv.ParseFloat(parts[8], 64)
	a.FlightSec, _ = strconv.ParseFloat(parts[9], 64)
	a.RunDepthFt, _ = strconv.ParseFloat(parts[10], 64)
	if len(parts) > 11 {
		a.ParentSig = parts[11]
	}
	fc.ActiveRastrub = append(fc.ActiveRastrub, a)
	fc.SetTorpedoSeq(parseTrailingInt(a.ID))
}

func parseRBU(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 10 {
		return
	}
	side, _ := strconv.Atoi(parts[3])
	a := &weapons.RBUSalvo{
		ID: parts[0], ParentID: parts[1], TargetID: parts[2],
		Side:  world.Side(side),
		Alive: true,
	}
	a.X0, _ = strconv.ParseFloat(parts[4], 64)
	a.Y0, _ = strconv.ParseFloat(parts[5], 64)
	a.X1, _ = strconv.ParseFloat(parts[6], 64)
	a.Y1, _ = strconv.ParseFloat(parts[7], 64)
	a.LaunchAt, _ = strconv.ParseFloat(parts[8], 64)
	a.FlightSec, _ = strconv.ParseFloat(parts[9], 64)
	fc.ActiveRBU = append(fc.ActiveRBU, a)
	fc.SetTorpedoSeq(parseTrailingInt(a.ID))
}

func parseHarpoon(fc *weapons.FireControl, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 20 {
		return
	}
	side, _ := strconv.Atoi(parts[3])
	tubeNum, _ := strconv.Atoi(parts[4])
	phase, _ := strconv.Atoi(parts[12])
	h := &weapons.HarpoonMissile{
		ID: parts[0], ParentSubID: parts[1], TargetContactID: parts[2],
		Side: world.Side(side), TubeNumber: tubeNum,
		Alive: true, VisibleOnWEPS: true,
		Phase: weapons.HarpoonPhase(phase),
	}
	h.LaunchX, _ = strconv.ParseFloat(parts[5], 64)
	h.LaunchY, _ = strconv.ParseFloat(parts[6], 64)
	h.X, _ = strconv.ParseFloat(parts[7], 64)
	h.Y, _ = strconv.ParseFloat(parts[8], 64)
	h.HeadingDeg, _ = strconv.ParseFloat(parts[9], 64)
	h.SpeedKts, _ = strconv.ParseFloat(parts[10], 64)
	h.DistanceYd, _ = strconv.ParseFloat(parts[11], 64)
	h.RadarOn, _ = strconv.ParseBool(parts[13])
	h.Alive, _ = strconv.ParseBool(parts[14])
	h.BeamHalfDeg, _ = strconv.ParseFloat(parts[15], 64)
	h.RadarRangeYd, _ = strconv.ParseFloat(parts[16], 64)
	h.DestructRangeYd, _ = strconv.ParseFloat(parts[17], 64)
	h.UnderwaterLeft, _ = strconv.ParseFloat(parts[18], 64)
	h.Age, _ = strconv.ParseFloat(parts[19], 64)
	if len(parts) > 20 {
		h.VisibleOnWEPS, _ = strconv.ParseBool(parts[20])
	}
	if len(parts) > 22 {
		h.ProgrammedHead, _ = strconv.ParseFloat(parts[21], 64)
		h.AssumedDistanceYd, _ = strconv.ParseFloat(parts[22], 64)
	} else {
		h.ProgrammedHead = h.HeadingDeg
		h.AssumedDistanceYd = h.DistanceYd
	}
	if len(parts) > 23 {
		v, _ := strconv.Atoi(parts[23])
		h.Variant = weapons.ASCMVariant(v)
	}
	h.CruiseKts = weapons.ASCMCruiseKts(h.Variant)
	fc.ActiveHarpoons = append(fc.ActiveHarpoons, h)
	fc.SetTorpedoSeq(parseTrailingInt(h.ID))
}

func parseObjective(val string) (world.Objective, bool) {
	parts := strings.Split(val, "|")
	if len(parts) < 4 {
		return world.Objective{}, false
	}
	done, _ := strconv.ParseBool(parts[2])
	obj := world.Objective{
		ID: parts[0], Description: i18n.T(parts[1], parts[1]), Complete: done, TargetID: parts[3],
	}
	if len(parts) >= 8 {
		obj.Primary, _ = strconv.ParseBool(parts[4])
		obj.NeedIdentify, _ = strconv.ParseBool(parts[5])
		obj.NeedDestroy, _ = strconv.ParseBool(parts[6])
		obj.Identified, _ = strconv.ParseBool(parts[7])
	} else {
		// Legacy saves: destroy-only tasks.
		obj.NeedDestroy = true
	}
	if len(parts) >= 9 {
		obj.Hidden, _ = strconv.ParseBool(parts[8])
	}
	return obj, true
}

func parsePlotMarker(val string) (world.PlotMarker, bool) {
	parts := strings.Split(val, "|")
	if len(parts) < 3 || parts[0] == "" {
		return world.PlotMarker{}, false
	}
	x, errX := strconv.ParseFloat(parts[1], 64)
	y, errY := strconv.ParseFloat(parts[2], 64)
	if errX != nil || errY != nil {
		return world.PlotMarker{}, false
	}
	return world.PlotMarker{ID: parts[0], X: x, Y: y}, true
}

func parseCMMag(cs *weapons.CountermeasureSystem, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	cs.SetMagazine(parts[0], n)
}

func parseJitterMag(cs *weapons.CountermeasureSystem, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	n, _ := strconv.Atoi(parts[1])
	cs.SetJitterMagazine(parts[0], n)
}

func parseNixie(cs *weapons.CountermeasureSystem, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	on, _ := strconv.ParseBool(parts[1])
	cs.SetNixie(parts[0], on)
}

func parseCMDeployAt(cs *weapons.CountermeasureSystem, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	t, _ := strconv.ParseFloat(parts[1], 64)
	if cs.LastDeployAt == nil {
		cs.LastDeployAt = map[string]float64{}
	}
	cs.LastDeployAt[parts[0]] = t
}

func parseJitterDeployAt(cs *weapons.CountermeasureSystem, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 2 {
		return
	}
	t, _ := strconv.ParseFloat(parts[1], 64)
	if cs.LastJitterAt == nil {
		cs.LastJitterAt = map[string]float64{}
	}
	cs.LastJitterAt[parts[0]] = t
}

func parseCM(cs *weapons.CountermeasureSystem, val string) {
	parts := strings.Split(val, "|")
	if len(parts) < 13 {
		return
	}
	side, _ := strconv.Atoi(parts[2])
	kind, _ := strconv.Atoi(parts[3])
	cm := &weapons.Countermeasure{
		ID: parts[0], ParentID: parts[1],
		Side: world.Side(side), Kind: weapons.CMKind(kind),
	}
	cm.X, _ = strconv.ParseFloat(parts[4], 64)
	cm.Y, _ = strconv.ParseFloat(parts[5], 64)
	cm.DepthFt, _ = strconv.ParseFloat(parts[6], 64)
	cm.HeadingDeg, _ = strconv.ParseFloat(parts[7], 64)
	cm.SpeedKts, _ = strconv.ParseFloat(parts[8], 64)
	cm.Alive, _ = strconv.ParseBool(parts[9])
	cm.Age, _ = strconv.ParseFloat(parts[10], 64)
	cm.TTL, _ = strconv.ParseFloat(parts[11], 64)
	cm.NoiseBoostDB, _ = strconv.ParseFloat(parts[12], 64)
	if cm.Alive {
		cs.Active = append(cs.Active, cm)
	}
}
