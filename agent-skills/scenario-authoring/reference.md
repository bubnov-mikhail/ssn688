# Scenario authoring — reference

Companion to [SKILL.md](SKILL.md). Schema: [`scenarios/schema.json`](../../scenarios/schema.json).

## Top-level fields

| Field | Purpose |
|-------|---------|
| `format_version` | Schema semver; **major must be 3** |
| `version` | Content semver (patch = text/tweaks, minor = new missions/fields) |
| `min_game_version` | Minimum game `VERSION` |
| `id` | Stable id (`^[a-z][a-z0-9_]{0,63}$`) — saves / import |
| `title` | UI title — **loc_string** `{"en","ru"}` |
| `backstory` | Scenario select markdown (mood only) — **loc_string** |
| `cover` | `{ mime, data_b64 }` image — **обязателен для каждой миссии**, если нужна brief map в игре |
| `overview_map` | *(planned)* regional gray map + geo boxes — inline после аппрува `__brief_map.png` |
| `postscript_success` / `postscript_failure` | End-of-campaign markdown — **loc_string** |
| `theaters` | Shared charts |
| `missions` | Ordered campaign beats |
| `events` | Optional scenario-level (prefer per-mission) |

### loc_string

```json
{ "en": "English", "ru": "Русский" }
```

Required: `en`. Duplicate all player-facing prose for every language in `i18n.SupportedLangs`. Plain strings are **invalid** in format 3.

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
| `start_time` | Wall clock `HH:MM` (24h); UI/COMM = start + elapsed |
| `comm_schedule` | Timed COMM `{ id, at_sec, text }` |
| `events` | When/then rules (runtime dispatch in sim) |
| `end_after_event` | Mission event `id` that must fire before COMM **report** / mission end |
| `outputs` | Campaign vars on mission end |
| `debrief_lead`, `debrief_lines` | After-action text by objective |

### Routes

- `waypoints`: `{ x, y }` in **yards** east/north of chart origin
- `mode`: `open` (stop), `pingpong` (reverse), `loop` (cycle)
- `player_clearance: true` — used when placing ownship near a corner

Helpers in `internal/world/diagonal_routes.go`, `coastal_routes.go`.

### Map preview validation

Before handoff, run:

```bash
go run ./tools/render_theater_routes.go -scenario scenarios_generated/<id>.json
```

Outputs under `scenarios_generated/theater_previews/` (see [SKILL.md](SKILL.md) §«Валидация карт»). Require `route_land_hits=0` in stdout/manifest. After user approves all missions and confirms finalization, delete previews per SKILL §«Уборка».

## Anti-patterns (content)

Do **not** clone Catalina mission content into a new theater:

- Objectives: sink diesel + identify/sink Grisha + identify tanker (HOLD)
- Units: `enemy_foxtrot`, `enemy_grisha`, `civ_tanker` with the same roles
- Routes: same diagonal lanes copied with new coordinates

Use `demo_catalina.json` for **JSON shape** only. Vary signatures, unit ids, route geometry, and objective verbs per [SKILL.md](SKILL.md) §«Оригинальность». Do not edit user scenarios (e.g. Taiwan) unless explicitly asked.

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
| `exercise_target` | Practice hulk — EX fish only, no warshot ASW |
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
| `harpoons` | Sub-Harpoon (allied 688) |
| `cruise_missiles` | Klub/Oniks/Kalibr ASCM |
| `asw_rockets` | Rastrub / Otvet / ASROC |
| `ship_tubes` | Lightweight tube fish |
| `exercise_torpedoes` | Mk48 EX signal fish (practice hulks) |
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

**Runtime** (dispatched each sim tick from `missions[].events`):

| Type | Fields | Fires when |
|------|--------|------------|
| `time` | `at_sec` | Mission clock ≥ `at_sec` (static `comm_schedule` with same `at_sec` is merged at load) |
| `objective_complete` | `objective_id` | Objective `Complete` |
| `objective_identified` | `objective_id` | Objective `Identified` (uses `IdentifiedAtSec` for ordering gates) |
| `unit_destroyed` | `unit_id` | Unit no longer `StatusActive` |
| `enemy_prosecutes_allies` | — | Hostile combatant prosecuting ownship or ally within ~35 kyd |

**Build-time only** (filters which events/missions/units load from campaign vars — not re-checked in sim):

| Type | Fields | Effect |
|------|--------|--------|
| `var_eq` | `var`, `value` (default `"true"`) | Include when campaign var equals |
| `var_unset` | `var` | Include when var not truthy |

On any `when` block you may also use `require_var` / `unless_var` (build-time) or `require_event` / `unless_event` (runtime ordering vs other mission events).

| Field | Effect |
|-------|--------|
| `require_event` | Prerequisite event `id` must have fired **before or at** trigger time |
| `unless_event` | Skip if prerequisite event already fired **before or at** trigger time |

For `objective_identified`, ordering uses the mission clock when the objective first became identified (`IdentifiedAtSec`).  
Use two events with the same `actions[].id` (e.g. `shadow_cue`) but different `when` gates for alternate COMM text.

```json
{
  "id": "shadow_cue_early",
  "when": {
    "type": "objective_identified",
    "objective_id": "obj_rf_shadow",
    "unless_event": "provocation"
  },
  "actions": [{ "type": "comm_schedule", "id": "shadow_cue", "text": { "en": "…stay on station…", "ru": "…" } }]
},
{
  "id": "shadow_cue_late",
  "when": {
    "type": "objective_identified",
    "objective_id": "obj_rf_shadow",
    "require_event": "provocation"
  },
  "actions": [{ "type": "comm_schedule", "id": "shadow_cue", "text": { "en": "…may request mission end…", "ru": "…" } }]
}
```

Build-time campaign filters remain `require_var` / `unless_var` on `when` (applied when the mission is instantiated, not each sim tick).

## Event actions (`actions[].type`)

| Type | Fields | Effect |
|------|--------|--------|
| `comm_schedule` | `id`, `text`, optional `at_sec` | Push COMM (immediate `at_sec` = trigger time if omitted) |
| `reveal_objective` | `objective_id` | Unhide a `hidden` objective |
| `set_defcon` | `unit_id`, `defcon` | `RaiseDefcon` on unit (never lowers) |
| `set_ai_state` | `unit_id`, `ai_state` | Force AI state string |
| `fire_weapon` | `shooter_id`, `target_id`, `weapon` | Scripted launch (see weapon kinds below) |
| `destroy_unit` | `unit_id` or `target_id`, optional `attributed_to` | Kill unit + detonation; attribution for debrief/blame |
| `plot_marker` | `id`, `x`, `y`, optional `name` | PLOT marker; `id: "enemy_group"` snaps to prosecuting enemy centroid |
| `ally_sub_assist` | `x`, `y` | Redirect allied sub (`INTERCEPT`) toward rendezvous (used with `enemy_prosecutes_allies`) |

### `fire_weapon` kinds

| `weapon` | Effect |
|----------|--------|
| `exercise_torpedo` / `exercise_fish` / `exercise` | Mk48 EX signal fish (`LaunchExerciseShipTube`) |
| `ship_torpedo` / `combat_torpedo` / `torpedo` | Lightweight ASW tube shot |
| `sub_torpedo` / `hostile_torpedo` | Heavy hostile fish from sub |
| `rbu` | RBU barrage (deferred splash) |
| `rastrub` | Rastrub/ASROC rocket (deferred splash) |

Aliases: `shooter_id` defaults to `unit_id` when omitted.

### Mission end gate

`end_after_event`: COMM **report** / mission end blocked until that event `id` has fired (e.g. `provocation` in `tw_twin_exercises`). Independent of objectives.

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

COMM placeholders: `{{unit.<id>.pos}}` (PLOT lat/lon), `.course`, `.speed`, `.depth`, `.x`, `.y`, `.name`, `.id`,
and `{{mission_time}}` (wall clock HH:MM:SS from mission `start_time` + elapsed / message AtSec).
Expanded via `ExpandPlaceholders`.

Mission `start_time` (`HH:MM` 24h) sets the wall clock origin; sim still counts seconds from 0.

## Markdown subset

`#` / `##` headings, `-` / `*` / `1.` lists, `**bold**` (markers stripped), blank line paragraphs.  
Renderer: `internal/render/markdown.go`.
