package weapons

import (
	"fmt"
	"math"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// Scenario event weapon kinds (mission events fire_weapon action).
const (
	EventWeaponExerciseTorpedo = "exercise_torpedo"
	EventWeaponShipTorpedo     = "ship_torpedo"
	EventWeaponCombatTorpedo   = "combat_torpedo"
	EventWeaponRBU             = "rbu"
	EventWeaponRastrub         = "rastrub"
	EventWeaponSubTorpedo      = "sub_torpedo"
)

// FireScenarioWeapon launches a scripted weapon for mission events.
// Returns a surface torpedo when spawned immediately, or nil when the effect
// is deferred (Rastrub/RBU splash) or failed.
func (fc *FireControl) FireScenarioWeapon(shooter, target *world.Entity, weapon string, gameTime float64) (*Torpedo, error) {
	if fc == nil || shooter == nil || target == nil {
		return nil, fmt.Errorf("fire_weapon: missing shooter or target")
	}
	if !shooter.Alive() || !target.Alive() {
		return nil, fmt.Errorf("fire_weapon: shooter or target not active")
	}
	switch weapon {
	case EventWeaponExerciseTorpedo, "exercise_fish", "exercise":
		t := fc.LaunchExerciseShipTube(shooter, target)
		if t == nil {
			return nil, fmt.Errorf("fire_weapon: exercise_torpedo out of range or failed")
		}
		return t, nil
	case EventWeaponShipTorpedo, EventWeaponCombatTorpedo, "torpedo":
		t := fc.LaunchShipTube(shooter, target)
		if t == nil {
			return nil, fmt.Errorf("fire_weapon: ship_torpedo out of range or empty magazine")
		}
		return t, nil
	case EventWeaponSubTorpedo, "hostile_torpedo":
		t := fc.SpawnHostileTorpedo(shooter, target)
		if t == nil {
			return nil, fmt.Errorf("fire_weapon: sub_torpedo failed")
		}
		return t, nil
	case EventWeaponRBU:
		if fc.LaunchRBU(shooter, target, gameTime) == nil {
			return nil, fmt.Errorf("fire_weapon: rbu failed")
		}
		return nil, nil
	case EventWeaponRastrub:
		if fc.LaunchRastrub(shooter, target, gameTime) == nil {
			return nil, fmt.Errorf("fire_weapon: rastrub failed")
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("fire_weapon: unknown weapon %q", weapon)
	}
}

// LaunchExerciseShipTube fires a lightweight ASW fish that terminates with an
// exercise signal (no warhead damage). Does not consume ship-tube magazine.
func (fc *FireControl) LaunchExerciseShipTube(ship, target *world.Entity) *Torpedo {
	if ship == nil || target == nil || !ship.Alive() || !target.Alive() {
		return nil
	}
	if world.IsExerciseTarget(ship) {
		left := fc.exerciseTubeAmmo(ship)
		if left <= 0 {
			return nil
		}
		fc.EnemyExerciseTube[ship.ID] = left - 1
	}
	rangeYd := math.Hypot(target.X-ship.X, target.Y-ship.Y)
	if rangeYd < ShipTubeMinRangeYd || rangeYd > ShipTubeMaxRangeYd {
		return nil
	}
	brg := bearing(ship.X, ship.Y, target.X, target.Y)
	rad := brg * math.Pi / 180
	x := ship.X + math.Sin(rad)*40
	y := ship.Y + math.Cos(rad)*40
	torp := fc.spawnLightweight(ship.ID, ship.SignatureID, ship.Side, target.ID, x, y, brg, target.DepthFt, true)
	if torp == nil {
		return nil
	}
	torp.TerminalMode = TerminalSignal
	torp.OrdnanceType = OrdnanceMk48Exercise
	return torp
}
