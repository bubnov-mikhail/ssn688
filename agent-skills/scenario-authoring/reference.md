# Scenario authoring — reference

Companion to [SKILL.md](SKILL.md). Schema: [`scenarios/schema.json`](../../scenarios/schema.json).

## Top-level fields

| Field | Purpose |
|-------|---------|
| `format_version` | Schema semver; **major must be 2** |
| `version` | Content semver (patch = text/tweaks, minor = new missions/fields) |
| `min_game_version` | Minimum game `VERSION` |
| `id` | Stable id (`^[a-z][a-z0-9_]{0,63}$`) — saves / import |
| `title` | UI title |
| `backstory` | Scenario select markdown (mood only) |
| `cover` | `{ mime, data_b64 }` image |
| `postscript_success` / `postscript_failure` | End-of-campaign markdown |
| `theaters` | Shared charts |
| `missions` | Ordered campaign beats |
| `events` | Optional scenario-level (prefer per-mission) |

## Theater

```json
{ "id": "catalina", "bathy": { "mime": "application/octet-stream", "data_b64": "..." } }
```

Bathy = BATH v1 grid. New theater → [bathymetry-and-routes](../bathymetry-and-routes/SKILL.md).

## Mission blocks

| Field | Purpose |
|-------|---------|
| `id`, `title`, `description` | Identity + start-of-mission intel |
| `theater_id` | Which chart |
| `routes` | Named lanes (`open` / `pingpong` / `loop`) |
| `player` | Ownship spawn |
| `units` | Traffic + combatants |
| `objectives` | Primary/secondary/hidden tasks |
| `comm_briefing` | Immediate COMM on start (mast up) |
| `comm_schedule` | Timed COMM `{ id, at_sec, text }` |
| `events` | When/then rules |
| `outputs` | Campaign vars on mission end |
| `debrief_lead`, `debrief_lines` | After-action text by objective |

### Routes

- `waypoints`: `{ x, y }` in **yards** east/north of chart origin
- `mode`: `open` (stop), `pingpong` (reverse), `loop` (cycle)
- `player_clearance: true` — used when placing ownship near a corner

Helpers in `internal/world/diagonal_routes.go`, `coastal_routes.go`.

### Unit

Required: `id`, `name`, `kind`, `side`, `signature_id`, `spawn`.

| Field | Notes |
|-------|--------|
| `kind` | `submarine` \| `surface_ship` |
| `side` | `player` \| `enemy` \| `neutral` (allies = `player` + `combatant`) |
| `spawn` | `corner` (+ `corner`, `min_route_yd`, `max_route_yd`) or `route` (+ `route_id`, `route_frac`) |
| `combatant` | Weapons / prosecute |
| `defcon`, `crew_skill`, `crew_jitter` | Alertness / competence |
| `ai_state` | Seed state (`PATROL`, `CRUISE`, `SEARCH`, …) |
| `ally_ignore` | Friendly AI must not engage |
| `require_var` / `unless_var` | Conditional spawn |
| `payload` | Optional magazine overrides (see below) |

Fallback corner fields apply if route placement fails.

### Unit payload (optional)

Overrides class defaults for **AI** magazines (enemy or ally). Omit field → keep default.

```json
"payload": {
  "torpedoes": 10,
  "asw_rockets": 4,
  "ship_tubes": 6,
  "rbu": 8,
  "sam": 12,
  "ciws": 4
}
```

| Key | Maps to |
|-----|---------|
| `torpedoes` | Heavy fish (`EnemyMagazine`) |
| `asw_rockets` | Rastrub / Otvet / ASROC |
| `ship_tubes` | Lightweight tube fish |
| `rbu` | RBU-6000 salvos |
| `sam` / `ciws` | Point defense vs Harpoon |

Player tube mix is chosen in UI loadout, not here.

### Objectives

| Field | Effect |
|-------|--------|
| `primary` | Mission fail if incomplete (with other primaries) |
| `need_identify` | Visual &lt;3000 yd or 80% harmonics × 2 min |
| `need_destroy` | Target sunk / fatal |
| `hidden` | Not shown until `reveal_objective` |
| `require_var` / `unless_var` | Include/skip from prior mission outputs |

### Outputs

```json
{ "key": "grisha_neutralized", "value": "true", "when_objective_id": "obj_grisha" }
```

Also `when_primary_complete`. Vars feed next mission filters and event `require_var` / `unless_var` / `var_eq`.

## Event triggers (`when.type`)

| Type | Fields | Fires when |
|------|--------|------------|
| `time` | `at_sec` | Mission clock (also merges `comm_schedule` at load) |
| `objective_complete` | `objective_id` | Task becomes complete |
| `objective_identified` | `objective_id` | ID criterion met |
| `unit_destroyed` | `unit_id` | Platform lost |
| `var_eq` | `var`, `value` | Campaign var equals (default `"true"`) — filter at build |
| `var_unset` | `var` | Var not truthy — filter at build |

All `when` may also use `require_var` / `unless_var`.

## Event actions (`actions[].type`)

| Type | Fields | Effect |
|------|--------|--------|
| `comm_schedule` | `id`, `text`, optional `at_sec` | Push COMM (immediate if runtime trigger) |
| `set_defcon` | `unit_id`, `defcon` | Raise/set AI DEFCON |
| `set_ai_state` | `unit_id`, `ai_state` | Force AI state |
| `set_var` | `var`, `value` | Runtime campaign var |
| `reveal_objective` | `objective_id` | Unhide objective |

## Signature catalog (`signature_id`)

**Player / ally subs:** `los_angeles`, `ssn688_decoy` (decoy fish acoustic)

**Enemy / other subs:** `foxtrot`, `kilo`, `victor_iii`, `yasen_m`

**Surface combatants:** `grisha`, `udaloy`, `krivak`, `gorshkov`, `kresta2`, `spruance` (ally ASW)

**Civilians:** `merchant`, `tanker`, `fishing`

**Weapons / CM (signatures, not scenario unit kinds):** `mk48`, `umgt1`, `set40`, `mk46`, `type53`, `adc`, `nixie`, `jitter`

## Player ordnance (loadout UI)

`Mk48`, `Mk48 EX`, `Harpoon` — tube/magazine mix at mission start.

## AI weapon suites (by class defaults)

| Class | Typical loadout |
|-------|-----------------|
| Diesel/SSK | Heavy fish magazine |
| Surface ASW (Udaloy/Krivak/…) | Rastrub/Otvet + ship tubes + SAM/CIWS |
| Grisha | RBU + ship tubes (no Rastrub) |
| Spruance | ASROC + tubes + PD |

Use `payload` to deplete or reinforce for narrative (veteran survivor, green crew).

## COMM placeholders

`{{unit.<id>.pos}}` (PLOT lat/lon), `.course`, `.speed`, `.depth`, `.x`, `.y`, `.name`, `.id`  
Expanded at COMM delivery (`ExpandUnitPlaceholders`).

## Markdown subset

`#` / `##` headings, `-` / `*` / `1.` lists, `**bold**` (markers stripped), blank line paragraphs.  
Renderer: `internal/render/markdown.go`.
