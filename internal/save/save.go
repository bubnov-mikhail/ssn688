package save

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/sim"
	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

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
	fmt.Fprintf(w, "format=1\n")
	fmt.Fprintf(w, "scenario=%s\n", engine.Scenario.Name)
	fmt.Fprintf(w, "game_time=%.3f\n", engine.Clock.GameTime)
	fmt.Fprintf(w, "time_scale=%.3f\n", engine.Clock.TimeScale)
	fmt.Fprintf(w, "paused=%t\n", engine.Clock.Paused)
	fmt.Fprintf(w, "\n")

	writeEntity(w, engine.Scenario.Player)
	for _, e := range engine.Scenario.Entities {
		writeEntity(w, e)
	}

	fmt.Fprintf(w, "\n[sonar]\n")
	fmt.Fprintf(w, "passive_enabled=%t\n", engine.Sonar.PassiveEnabled)
	fmt.Fprintf(w, "active_enabled=%t\n", engine.Sonar.ActiveEnabled)
	fmt.Fprintf(w, "active_power=%.3f\n", engine.Sonar.ActivePower)
	fmt.Fprintf(w, "ping_interval=%.3f\n", engine.Sonar.PingInterval)
	fmt.Fprintf(w, "spectrum_bearing=%.3f\n", engine.Sonar.SpectrumBearing)
	fmt.Fprintf(w, "passive_array=%d\n", engine.Sonar.PassiveArray)
	fmt.Fprintf(w, "towed_cable_pct=%.3f\n", engine.Sonar.TowedCablePct)
	fmt.Fprintf(w, "towed_cable_rate=%.3f\n", engine.Sonar.TowedCableRate)
	for _, c := range engine.Sonar.Contacts {
		fmt.Fprintf(w, "contact=%s|%.3f|%.3f|%.3f|%s|%s|%.3f|%s|%s|%d|%s|%s\n",
			c.ID, c.BearingDeg, c.EstimatedRangeYd, c.SNR,
			c.BestMatchID, c.BestMatchName, c.Confidence,
			c.SourceEntityID, c.DetectedBy, c.Kind,
			c.ConfirmedID, c.ConfirmedClass)
	}

	fmt.Fprintf(w, "\n[fire_control]\n")
	fmt.Fprintf(w, "selected_tube=%d\n", engine.FireControl.SelectedTube)
	fmt.Fprintf(w, "gyro_angle=%.3f\n", engine.FireControl.GyroAngleDeg)
	fmt.Fprintf(w, "run_depth=%.3f\n", engine.FireControl.RunDepthFt)
	fmt.Fprintf(w, "speed_setting=%s\n", engine.FireControl.SpeedSetting)
	fmt.Fprintf(w, "seeker_enabled=%t\n", engine.FireControl.SeekerEnabled)
	for _, t := range engine.FireControl.Tubes {
		fmt.Fprintf(w, "tube=%d|%d|%s|%t\n", t.Number, t.State, t.TorpedoType, t.WireIntact)
	}
	for _, torp := range engine.FireControl.ActiveTorpedoes {
		fmt.Fprintf(w, "torpedo=%s|%s|%s|%d|%.3f|%.3f|%.3f|%.3f|%.3f|%.3f|%t|%t|%t|%t|%d|%.3f\n",
			torp.ID, torp.ParentSubID, torp.TargetID, torp.Side,
			torp.X, torp.Y, torp.DepthFt, torp.HeadingDeg, torp.SpeedKts, torp.RunDepthFt,
			torp.SeekerOn, torp.WireCut, torp.Armed, torp.Alive, torp.Mode, torp.Age)
	}

	fmt.Fprintf(w, "\n[objectives]\n")
	for _, o := range engine.Scenario.Objectives {
		fmt.Fprintf(w, "objective=%s|%s|%t|%s\n", o.ID, o.Description, o.Complete, o.TargetID)
	}
	return w.Flush()
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
	fmt.Fprintf(w, "active_sonar=%t\n", e.ActiveSonar)
	fmt.Fprintf(w, "last_ping_time=%.3f\n", e.LastPingTime)
	fmt.Fprintf(w, "last_ping_power=%.3f\n", e.LastPingPower)
	fmt.Fprintf(w, "ai_state=%s\n", e.AIState)
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

	sc := world.NewTrainingScenario()
	engine := sim.NewEngine(sc)
	engine.Scenario.Entities = nil

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
		case "sonar":
			switch key {
			case "passive_enabled":
				engine.Sonar.PassiveEnabled, _ = strconv.ParseBool(val)
			case "active_enabled":
				engine.Sonar.ActiveEnabled, _ = strconv.ParseBool(val)
			case "active_power":
				engine.Sonar.ActivePower, _ = strconv.ParseFloat(val, 64)
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
			case "tube":
				parseTube(&engine.FireControl, val)
			case "torpedo":
				parseTorpedo(&engine.FireControl, val)
			}
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
		default:
			switch key {
			case "game_time":
				engine.Clock.GameTime, _ = strconv.ParseFloat(val, 64)
			case "time_scale":
				engine.Clock.TimeScale, _ = strconv.ParseFloat(val, 64)
			case "paused":
				engine.Clock.Paused, _ = strconv.ParseBool(val)
			}
		}
	}

	if len(engine.Scenario.Objectives) == 0 {
		engine.Scenario.Objectives = world.NewTrainingScenario().Objectives
	}
	return engine, nil
}

// LoadClean is an alias for Load.
func LoadClean(path string) (*sim.Engine, error) {
	return loadClean(path)
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
	case "active_sonar":
		e.ActiveSonar, _ = strconv.ParseBool(val)
	case "last_ping_time":
		e.LastPingTime, _ = strconv.ParseFloat(val, 64)
	case "last_ping_power":
		e.LastPingPower, _ = strconv.ParseFloat(val, 64)
	case "ai_state":
		e.AIState = val
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
		fc.Tubes[num-1] = weapons.Tube{
			Number: num, State: weapons.TubeState(state),
			TorpedoType: parts[2], WireIntact: wire,
		}
	}
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
	fc.ActiveTorpedoes = append(fc.ActiveTorpedoes, torp)
}

func parseObjective(val string) (world.Objective, bool) {
	parts := strings.Split(val, "|")
	if len(parts) < 4 {
		return world.Objective{}, false
	}
	done, _ := strconv.ParseBool(parts[2])
	return world.Objective{
		ID: parts[0], Description: parts[1], Complete: done, TargetID: parts[3],
	}, true
}
