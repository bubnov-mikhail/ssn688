//go:build ignore

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/sim"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

const maxSec = 5400.0 // 90 min mission time

func main() {
	data, err := os.ReadFile("scenarios_generated/taiwan_formosa_watch.json")
	if err != nil {
		panic(err)
	}
	scDef, err := campaign.ParseScenarioJSON(data, "taiwan")
	if err != nil {
		panic(err)
	}
	m := campaign.FindMission(&scDef, "tw_attribution")
	if m == nil {
		panic("tw_attribution missing")
	}

	fmt.Println("=== tw_attribution AFK forecast (player idle, 90 min) ===")
	fmt.Println()

	// Spawn sanity (one instantiate)
	ctx := campaign.BuildContext{Vars: map[string]string{}}
	rt := campaign.Instantiate(&scDef, m, ctx)
	if rt == nil {
		panic("instantiate failed")
	}
	fmt.Println("--- spawns (single instantiate) ---")
	for _, e := range rt.AllEntities() {
		if e == nil {
			continue
		}
		fmt.Printf("  %-16s %8.0f,%8.0f  depth=%.0fft crew=%.0f\n", e.ID, e.X, e.Y, e.DepthFt, e.CrewSkill)
	}
	fmt.Println()

	seeds := []int64{1, 2, 3, 4, 5}
	for _, seed := range seeds {
		runForecast(&scDef, m, seed)
	}
}

func runForecast(scDef *campaign.ScenarioDef, m *campaign.MissionDef, seed int64) {
	ctx := campaign.BuildContext{Vars: map[string]string{}}
	rng := rand.New(rand.NewSource(seed))
	// deterministic crew from seed
	rt := campaign.Instantiate(scDef, m, ctx)
	if rt == nil {
		panic("instantiate")
	}
	// re-roll crew per seed for variance
	for _, e := range rt.AllEntities() {
		if e == nil {
			continue
		}
		rollCrewForSeed(rng, scDef, m, e)
	}

	eng := sim.NewEngine(rt)
	campaign.ApplyUnitPayloads(&eng.FireControl, m, ctx.Vars)
	player := eng.Scenario.Player
	player.OrderedSpeed = 0
	player.SpeedKts = 0

	deathAt := map[string]float64{}
	dt := 1.0 / sim.TickRate
	t0 := time.Now()

	for eng.Clock.GameTime < maxSec {
		eng.Update(dt)
		for _, e := range eng.Scenario.AllEntities() {
			if e == nil || e.Alive() || deathAt[e.ID] > 0 {
				continue
			}
			deathAt[e.ID] = eng.Clock.GameTime
		}
		if player != nil && !player.Alive() && deathAt["player"] == 0 {
			deathAt["player"] = eng.Clock.GameTime
		}
	}

	elapsed := time.Since(t0)
	fmt.Printf("=== seed %d  weather=%s  sim_cpu=%.1fs ===\n", seed, rt.Weather, elapsed.Seconds())

	order := []string{
		"player", "ally_spruance", "ally_rocn", "ally_688",
		"rf_victor", "plan_grisha", "plan_krivak", "rf_kilo_quiet", "civ_trawler",
	}
	for _, id := range order {
		e := findEntity(eng.Scenario, id)
		if e == nil {
			continue
		}
		st := "ALIVE"
		if t, ok := deathAt[id]; ok && t > 0 {
			st = fmt.Sprintf("DEAD @%dm%02ds", int(t)/60, int(t)%60)
		} else if !e.Alive() {
			st = "DEAD"
		}
		fmt.Printf("  %-16s %-14s hull=%3.0f%% crew=%3.0f ai=%-14s defcon=%d\n",
			id, st, hullPct(e), e.CrewSkill, e.AIState, e.Defcon)
	}

	// objectives
	eng.Scenario.CheckObjectives()
	fmt.Println("  objectives:")
	for _, o := range eng.Scenario.Objectives {
		mark := "[ ]"
		if o.Complete {
			mark = "[x]"
		}
		id := o.ID
		if o.Identified {
			id += " ID"
		}
		fmt.Printf("    %s %s\n", mark, id)
	}

	aliveEnemy := 0
	aliveAlly := 0
	for _, e := range eng.Scenario.Entities {
		if e == nil || !e.Alive() {
			continue
		}
		if e.Side == world.SideEnemy && (e.Kind == world.KindSubmarine || e.Kind == world.KindSurfaceShip) && e.ID != "civ_trawler" {
			aliveEnemy++
		}
		if world.IsAllyAI(e, player) {
			aliveAlly++
		}
	}
	fmt.Printf("  summary: allies_alive=%d enemies_alive=%d torpedoes=%d player=%s\n\n",
		aliveAlly, aliveEnemy, len(eng.FireControl.ActiveTorpedoes), playerStatus(player))
}

func rollCrewForSeed(rng *rand.Rand, scDef *campaign.ScenarioDef, m *campaign.MissionDef, e *world.Entity) {
	spec := findUnitSpec(m, e.ID)
	if spec == nil && m.Player.ID == e.ID {
		spec = &m.Player
	}
	if spec == nil || (spec.CrewSkill <= 0 && spec.CrewJitter <= 0) {
		return
	}
	e.CrewSkill = world.RandomCrewSkill(spec.CrewSkill, spec.CrewJitter, rng.Float64())
}

func findUnitSpec(m *campaign.MissionDef, id string) *campaign.UnitSpec {
	for i := range m.Units {
		if m.Units[i].ID == id {
			return &m.Units[i]
		}
	}
	return nil
}

func findEntity(sc *world.Scenario, id string) *world.Entity {
	if sc.Player != nil && sc.Player.ID == id {
		return sc.Player
	}
	for _, e := range sc.Entities {
		if e != nil && e.ID == id {
			return e
		}
	}
	return nil
}

func hullPct(e *world.Entity) float64 {
	if e == nil {
		return 0
	}
	e.EnsureDamage()
	return e.Damage.Eff[world.SysHull]
}

func playerStatus(p *world.Entity) string {
	if p == nil {
		return "n/a"
	}
	if p.Alive() {
		return fmt.Sprintf("alive %.0f%%", hullPct(p))
	}
	return "dead"
}
