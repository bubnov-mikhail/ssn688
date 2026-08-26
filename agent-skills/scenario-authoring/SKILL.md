---
name: scenario-authoring
description: >-
  Author portable SSN-688 campaign scenarios and missions (JSON schema, theaters,
  routes, units, objectives, events, COMM, branching vars). Use when creating a new
  scenario, adding missions, writing briefs/debriefs, or generating files under
  scenarios_generated/.
---

# Создание сценариев и миссий

Игра слабее современных аналогов по графике — **выигрывать нужно сюжетом и механиками**.  
Сценарий должен ощущаться живым: эскалация, ветвление от вторичных задач, реалистичное AI и активный COMM.

## Когда применять

- Новый сценарий / новые миссии в существующем
- Правка `backstory`, mission `description`, COMM, objectives, events
- Генерация JSON в `scenarios_generated/` для ручного импорта в игру

## Куда класть результат

| Путь | Назначение |
|------|------------|
| [`scenarios_generated/`](../../scenarios_generated/) | Готовые `.json` для импорта (**не в git** — см. `.gitignore`) |
| [`scenarios/`](../../scenarios/) | Только bundled-демо (например `demo_catalina.json`) после явной просьбы |
| [`scenarios/schema.json`](../../scenarios/schema.json) | Источник правды по полям JSON |

Игрок импортирует файл через **IMPORT SCENARIO** в главном меню. Агент **не** кладёт сгенерированные сценарии в `scenarios/` без запроса.

`format_version` major = `ScenarioFormatMajor` (**2**). При правках контента инкрементируй `version` по semver.

## Новый театр (карта)

Если нужна **новая география** (не Catalina): сначала навык  
[`bathymetry-and-routes`](../bathymetry-and-routes/SKILL.md) — ETOPO → `bathy.bin` → `theaters[].bathy.data_b64`, маршруты без суши, глубины ПЛ.

Переиспользуй существующий theater id, если карта та же.

## Обложка

Картинку сценария (`cover`) генерируй **внешней image-моделью** (не текстовым агентом).  
Учитывай запрос пользователя и тон сценария (театр, эпоха, напряжение). Inline как JPEG/PNG `data_b64` в JSON. Mission-level `cover` обычно не нужен.

## Нарратив и спойлеры

**`backstory` (экран сценария)**  
Задаёт **общее настроение**, геополитику, flashpoints.  
**Не** раскрывать состав и задачи каждой миссии, конкретные классы целей первой миссии, скрытые twist’ы (sink танкера и т.п.).

**`description` миссии**  
Только то, что известно **на старте** миссии. Допустимы намёки «от разведки».  
Не светить будущие COMM-приказы и скрытые objectives.

**COMM / debrief**  
Детали tasking, ROE-смены, позиции — сюда и в `events`, не в backstory.

Тексты — markdown subset (`#` / `##`, списки, `**bold**`). Placeholders: `{{unit.<id>.pos}}`, `.course`, `.speed`, `.name` (см. `ExpandUnitPlaceholders`).

## Дизайн кампании

1. **Эскалация**: первые миссии легче центральных и финала (напряжение, число угроз, crew_skill / DEFCON).
2. **Вторичные задачи**: провал усложняет следующие миссии (`outputs` → `require_var` / `unless_var`, выше DEFCON, veteran crew), но **не** делает их невыполнимыми.
3. **Скрытые objectives** (`hidden: true` + `reveal_objective`) — для twist’ов после ID/события.
4. **AI**: маршруты, стороны, `combatant`, `ally_ignore`, DEFCON/crew_skill должны быть логичны (патруль, ASW, гражданские коридоры).
5. **COMM как реакция** на действия игрока (`objective_complete` / `objective_identified` / `unit_destroyed`) — сильный плюс; живой радиообмен важнее длинного статичного брифинга.

## Workflow

```
Task Progress:
- [ ] Concept: tone, arc, 2–5 missions, branching vars
- [ ] Theater: reuse or bathymetry-and-routes
- [ ] Cover: image model → base64
- [ ] Draft JSON per schema (units, routes, objectives, events)
- [ ] Validate mentally vs schema; bump version
- [ ] Write to scenarios_generated/<id>.json
- [ ] Tell user to IMPORT SCENARIO in-game
- [ ] Cleanup: remove this job's temp files from tools/
```

Эталон структуры и ветвления: `tools/gen_demo_scenario_json.go` + `scenarios/demo_catalina.json`.

## Уборка после причёсывания

Когда сценарий **закончен** (контент уложен в `scenarios_generated/<id>.json`, правок больше не планируется в этой сессии) — **прибрать за собой** в `tools/`.

Cover и bathy уже inline в JSON (`data_b64`); оставлять дубликаты на диске не нужно.

| Удалить | Примеры |
|---------|---------|
| Cover этой генерации | `tools/<scenario>_cover.png`, `tools/*_cover.jpg` |
| Bathy театра этой работы | `tools/bathy_<theater>.bin` (не трогать `bathy_catalina.bin`, если он часть демо-пайплайна) |
| Одноразовые генераторы | `tools/gen_<scenario>_*.go`, `tools/gen_<theater>_bathy.py`, созданные только под этот сценарий |
| Временный кэш ETOPO/subset | `tools/.etopo_cache/` или скачанные csv/nc **только** для этого театра |

**Не удалять** общие эталоны: `tools/gen_hormuz_bathy.py`, `tools/gen_demo_scenario_json.go`, vendor и прочие shared tools.

Если пользователь явно просит **оставить пайплайн пересборки** — генераторы и `bathy_*.bin` не трогать; cover всё равно можно удалить после inline.

## Структура JSON (кратко)

Полная схема: [`scenarios/schema.json`](../../scenarios/schema.json). Каталоги полей, юнитов, оружия и events — в [reference.md](reference.md).

| Блок | Роль |
|------|------|
| `format_version` / `version` / `min_game_version` | Совместимость и semver контента |
| `id`, `title`, `backstory`, `cover` | Карточка сценария |
| `postscript_success` / `postscript_failure` | Эпилог кампании |
| `theaters[]` | Карты с inline bathy |
| `missions[]` | Миссии: routes, player, units, objectives, COMM, events, outputs, debrief |
| `events[]` (scenario) | Редко; предпочтительны mission events |

Ключевые поля миссии: `theater_id`, `routes`, `player`, `units`, `objectives`, опционально `comm_briefing`, `comm_schedule`, `events`, `outputs`, `debrief_*`.

Юнит: `signature_id`, `side`, `spawn` (`corner`\|`route`), маршруты, `require_var`/`unless_var`, опционально **`payload`** (магазины AI).

Objective: `need_identify` / `need_destroy` / `primary` / `hidden` + var-фильтры.

## Чеклист качества

- [ ] Backstory без спойлеров миссий
- [ ] Mission description = только стартовый intel
- [ ] Эскалация сложности по арке
- [ ] Вторички влияют на следующую миссию, но оставляют win-path
- [ ] Маршруты в воде; ПЛ ниже термоклина (если не coastal-special)
- [ ] Хотя бы несколько reactive COMM / events
- [ ] Файл в `scenarios_generated/`, не в git
- [ ] `format_version` 2.x.x, осмысленный `version`
- [ ] После финала: убраны temp-файлы этой работы из `tools/` (cover, bathy.bin, одноразовые gen_*)
