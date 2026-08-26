package campaign

import "github.com/ssn688/sim/internal/weapons"

// UnitPayload overrides default magazines for AI platforms (enemy or allied).
// Nil fields keep class defaults from weapons.EnemySubMagazineFor / Surface* helpers.
type UnitPayload struct {
	Torpedoes  *int // heavy fish (subs)
	ASWRockets *int // Rastrub / Otvet / ASROC cells
	ShipTubes  *int // lightweight tube fish
	RBU        *int // RBU-6000 salvos
	SAM        *int // point-defense SAM
	CIWS       *int // CIWS bursts
}

// ApplyUnitPayloads seeds FireControl magazines from mission unit specs.
// Call once after NewEngine when starting a mission (not on save reload).
func ApplyUnitPayloads(fc *weapons.FireControl, m *MissionDef, vars map[string]string) {
	if fc == nil || m == nil {
		return
	}
	if vars == nil {
		vars = map[string]string{}
	}
	if fc.EnemyMagazine == nil {
		fc.EnemyMagazine = map[string]int{}
	}
	if fc.EnemyRastrub == nil {
		fc.EnemyRastrub = map[string]int{}
	}
	if fc.EnemyShipTube == nil {
		fc.EnemyShipTube = map[string]int{}
	}
	if fc.EnemyRBU == nil {
		fc.EnemyRBU = map[string]int{}
	}
	if fc.EnemySAM == nil {
		fc.EnemySAM = map[string]int{}
	}
	if fc.EnemyCIWS == nil {
		fc.EnemyCIWS = map[string]int{}
	}
	applyOne := func(u UnitSpec) {
		if u.Payload == nil || !specMatchesVars(u.RequireVar, u.UnlessVar, vars) {
			return
		}
		p := u.Payload
		id := u.ID
		if p.Torpedoes != nil {
			fc.EnemyMagazine[id] = *p.Torpedoes
		}
		if p.ASWRockets != nil {
			fc.EnemyRastrub[id] = *p.ASWRockets
		}
		if p.ShipTubes != nil {
			fc.EnemyShipTube[id] = *p.ShipTubes
		}
		if p.RBU != nil {
			fc.EnemyRBU[id] = *p.RBU
		}
		if p.SAM != nil {
			fc.EnemySAM[id] = *p.SAM
		}
		if p.CIWS != nil {
			fc.EnemyCIWS[id] = *p.CIWS
		}
	}
	applyOne(m.Player)
	for _, u := range m.Units {
		applyOne(u)
	}
}
