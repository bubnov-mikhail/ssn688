---
name: scenario-authoring
description: >-
  Author portable SSN-688 campaign scenarios and missions (JSON schema, theaters,
  routes, units, objectives, events, COMM, branching vars). Requires original
  unit/route/objective design (not Catalina clones). Gemini/strong-model pass for
  natural Russian loc_string. Use when creating a new scenario, adding missions,
  writing briefs/debriefs, or generating files under scenarios_generated/.
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
| `scenarios_generated/theater_previews/` | PNG-превью карт (маршруты, спавны) — **не в git**, только для QA в сессии |
| `scenarios_generated/theater_bathy/` | Промежуточные `.bin` до inline в JSON — **не в git** |
| [`scenarios/`](../../scenarios/) | Только bundled-демо (например `demo_catalina.json`) после явной просьбы |
| [`scenarios/schema.json`](../../scenarios/schema.json) | Источник правды по полям JSON |

Игрок импортирует файл через **IMPORT SCENARIO** в главном меню. Агент **не** кладёт сгенерированные сценарии в `scenarios/` без запроса.

`format_version` major = `ScenarioFormatMajor` (**3**). При правках контента инкрементируй `version` по semver.

## Мультиязычность

Все **player-facing** тексты сценария — объекты `loc_string`:

```json
{ "en": "English text", "ru": "Русский текст" }
```

Обязателен ключ `en`; для каждого языка из `i18n.SupportedLangs` (`en`, `ru`) нужен осмысленный перевод.  
Поля: `title`, `backstory`, `postscript_*`, mission `title`/`description`/`comm_briefing`/`debrief_*`, `objectives[].description`, unit `name`, COMM/`events` `text`.

Язык в игре задаётся в **SETTINGS** (`en` / `ru`); UI и тексты сценария берутся через `GetText(lang)`.

Озвучка Kokoro сейчас **без русского** — WAV остаются EN; субтитры локализуются.

### Перевод на `ru` (обязательный проход)

Черновик `ru` от основного агента **недостаточен** для сдачи: кальки, канцелярит и «переводческие» обороты ломают тон COMM/брифингов.

1. Сначала пиши / правь **`en`** (источник смысла, ROE, placeholders `{{unit.*}}`, markdown-структура).
2. Для **всех** длинных player-facing `ru` (как минимум: `backstory`, mission `description` / `comm_briefing` / `debrief_*`, все COMM/`events` `text.ru`, `postscript_*`) сделай **отдельный проход адаптации** через **Gemini** или другую доступную сильную модель (Claude / GPT и т.п. — что есть в сессии или у пользователя).
3. Задача модели: не дословный перевод, а **естественный военный/оперативный русский** (радиограммы, штабной стиль), с сохранением:
   - markdown (`#` / `##`, списки, `**bold**`);
   - placeholders `{{unit.<id>.…}}` **байт-в-байт**;
   - ОТ/КОМУ, приоритетов (FLASH / СРОЧНО и т.п. — можно локализовать осмысленно, напр. FLASH → МОЛНИЯ);
   - смысла ROE / задач / угроз.
4. Практичный пайплайн: выгрузить таблицу «исходный ru → улучшить» (или en → ru), прогнать моделью, затем **пакетно заменить** в JSON; после замены — bump `version` (patch).
5. Короткие ярлыки (`title`, unit `name`, однострочные objective) можно править агентом, если звучат естественно; при сомнении — тот же внешний проход.

**Не сдавай** сценарий с сырым машинным/`en`-калькой в `ru` без этого прохода.

## Новый театр (карта)

Если нужна **новая география** (не Catalina): сначала навык  
[`bathymetry-and-routes`](../bathymetry-and-routes/SKILL.md) — ETOPO → `bathy.bin` → `theaters[].bathy.data_b64`, маршруты без суши, глубины ПЛ.

Переиспользуй существующий theater id, если карта та же.

## Валидация карт (рендер)

После того как в JSON есть `theaters[].bathy` и mission `routes` / `player` / `units`, **обязательно** сгенерируй превью и проверь геометрию **до** сдачи сценария пользователю.

### Команда

```bash
go run ./tools/render_theater_routes.go -scenario scenarios_generated/<scenario_id>.json
```

Альтернатива сразу после ETOPO-сборки зон (если bathy ещё только в `.bin`):

```bash
python tools/gen_bathy_zone.py --preview --scenario scenarios_generated/<scenario_id>.json
```

(`--preview` в конце вызывает тот же Go-рендерер.)

### Выход

| Путь | Содержимое |
|------|------------|
| `scenarios_generated/theater_previews/00_<scenario_id>__overview.png` | Карта региона: все театры миссий (прямоугольники + inset bathy) |
| `scenarios_generated/theater_previews/NN_<theater>__<mission_id>.png` | Карта миссии: bathy, берег, маршруты, подписи юнитов, зона PLAYER |
| `scenarios_generated/theater_previews/NN_<mission_id>__brief_map.png` | Brief map для экрана миссии: gray overview, past B&W, current color |
| `scenarios_generated/theater_previews/manifest.json` | `land_pct`, `route_land_hits` по каждому PNG |

Промежуточная батиметрия до inline: `scenarios_generated/theater_bathy/<theater_id>.bin`.  
Regional overview: `theater_bathy/<scenario>_overview.bin` (для Taiwan — `taiwan_overview.bin`).

`NN` — порядковый номер миссии (`01`, `02`, …).

### Brief map (экран описания миссии)

После overview покажи **`NN_<mission_id>__brief_map.png`** — макет regional-карты справа от текста брифинга:

- фон — серая regional bathy (как overview);
- прошлые миссии — ч/б прямоугольники без цветной заливки;
- текущая — цветная рамка + inset из **`missions[].cover`**;
- будущие миссии не рисуются;
- **без mission `cover` в игре панель не показывается** (в превью — `NO COVER` + color bathy для layout).

Каждая миссия кампании должна иметь свой **`cover`**. Inline brief map в JSON — только после аппрува PNG пользователем.

### Сообщение пользователю о превью

После первого успешного рендера **обязательно** объясни пользователю (адаптируй формулировки под сценарий):

> **Превью карт** — это не часть игры и не попадает в импорт. PNG в `scenarios_generated/theater_previews/` нужны, чтобы **до импорта** проверить геометрию: берег, глубины, маршруты AI, точки спавна юнитов и зону старта PLAYER.  
> На каждой карте миссии цветные линии — маршруты (подпись = `unit.id`), голубой круг — зона ownship. **Красные квадраты** на сегменте маршрута означают пересечение суши — такую миссию нельзя сдавать, пока маршруты не исправлены.  
> Overview (`00_<scenario_id>__overview.png`) показывает, какие театры занимают миссии кампании.  
> **Brief map** (`NN_<mission_id>__brief_map.png`) — макет regional-карты на экране брифинга (past B&W, current color, future hidden); требует mission `cover` для игры.

Не переходи к «сценарий готов» без визуальной проверки пользователем.

### Инспекция и аппрув миссий

Работай **по одной миссии**:

1. Покажи PNG `NN_<theater>__<mission_id>.png` (и кратко: театр, число юнитов/маршрутов, `route_land_hits` из manifest/stdout).
2. Попроси пользователя **проинспектировать** карту: маршруты в воде, спавны логичны, PLAYER не на суше, нет неожиданных пересечений берега.
3. Дождись явного **аппрува** этой миссии («ок», «принято», правки внесены и снова ок). Если нужны правки — правь JSON/маршруты, **перерендери**, повтори шаг 1–3 для той же миссии.
4. Переходи к следующей миссии только после аппрува текущей.

Для кампании из N миссий нужны N отдельных аппрувов (overview + **каждая** brief map — отдельно; маршрутная карта миссии — отдельно).

### Завершение генерации

Когда **все** миссии аппрувнуты и JSON финален (`version` bumped, `route_land_hits=0` везде), **спроси пользователя**:

> Все миссии приняты. Завершить генерацию сценария и убрать временные файлы (`theater_previews/`, `theater_bathy/`, cover до inline, одноразовые gen/patch-скрипты)? Файл `scenarios_generated/<id>.json` останется.

**Уборку (§«Уборка») выполняй только после явного согласия.** Если пользователь хочет оставить превью или bathy `.bin` для доработок — не удаляй, пока не попросит.

### Что смотреть на PNG

| Элемент | Ожидание |
|---------|----------|
| **Песочный контур** | Береговая линия (marching squares) |
| **Цветные линии + подписи** | Маршруты; подпись = `unit.id` (или `id1+id2`, если два юнита на одном route) |
| **Красные квадраты** | Сегмент маршрута пересекает сушу — **исправить** (waypoints / midpoints / другая геометрия) |
| **Голубой круг `PLAYER …`** | Зона спавна ownship (`player.corner` + `corner_inset_yd`) |
| **stdout / manifest** | `route_land_hits=0` на всех миссиях; `land_pct` театра в целевом диапазоне (см. bathymetry-and-routes) |

Если hits > 0: правь маршруты в JSON или пересобери через хелперы [`bathymetry-and-routes`](../bathymetry-and-routes/SKILL.md), снова `go run ./tools/render_theater_routes.go`, пока красных меток нет.

Для мульти-театровых кампаний (свой `theater_id` на миссию) — **отдельное превью на каждую миссию**; не ограничивайся одной картой.

Покажи пользователю путь к `scenarios_generated/theater_previews/` и PNG по миссиям (см. §«Инспекция и аппрув миссий»).

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
4. **AI**: маршруты, стороны, `combatant`, `ally_ignore`, DEFCON/crew_skill должны быть логичны (патруль, ASW, гражданские коридоры). Маршруты — **разные оси/формы** (не пачка параллельных диагоналей); см. [`bathymetry-and-routes`](../bathymetry-and-routes/SKILL.md) §«Разнообразие геометрии».
5. **COMM как реакция** на действия игрока (`objective_complete` / `objective_identified` / `unit_destroyed`) — сильный плюс; живой радиообмен важнее длинного статичного брифинга.

### Оригинальность: не копировать эталон

`scenarios/demo_catalina.json` и `tools/gen_demo_scenario_json.go` — **только** эталон **формата** (поля JSON, ветвление `outputs`, events, COMM).  
**Не** переносить оттуда состав миссии «как есть».

**Запрещённый шаблон** (и близкие варианты — тот же геймплей под другим названием):

| Клише Catalina | Почему плохо |
|----------------|--------------|
| Потопить дизель (Foxtrot / Kilo) + идентифицировать и потопить Grisha + идентифицировать танкер (HOLD) | Игрок уже проходил это в демо; нет смысла в «новом» театре |
| Те же `signature_id` / роли: `foxtrot`, `grisha`, `tanker` в той же тройке задач | Нет разнообразия классов и тактики |
| Те же `unit.id` (`enemy_foxtrot`, `enemy_grisha`, `civ_tanker`) и имена маршрутов (`route_foxtrot`, …) | Путаница при отладке, ощущение клона |
| Те же оси маршрутов и расстановка «как на Catalina», только другая карта | Карта меняется, головоломка — нет |

**Уже существующие сценарии пользователя не переписывать** без явной просьбы (в т.ч. `scenarios_generated/taiwan_formosa_watch.json` и импортированные в `~/Library/Application Support/ssn688/scenarios/`). Новые правила — для **следующих** сценариев.

#### Чем заменять (минимум 3 оси разнообразия на кампанию)

Перед финалом JSON сверься: **хотя бы три** пункта ниже должны отличаться от демо-Catalina **и** от предыдущей миссии в этой кампании.

**Состав сил (units)**

- Меняй **классы** и роли: корвет/фрегат/ЭМ вместо Grisha; Yasen/Akula/Kilo/Song вместо «очередного Foxtrot»; гражданские — сухогруз, контейнеровоз, рыболов, а не только VLCC.
- Меняй **число и сочетание** угроз: одна ПЛ + два корабля; только surface ASW; засадный дизель + тыловой крейсер; союзный патруль + вражеский контакт в neutral lane.
- Уникальные `id` юнитов: `enemy_patrol_01`, `civ_lane_traffic_a` — не `enemy_grisha` / `civ_tanker`.

**Маршруты**

- Новая геометрия под **берег и bathy** театра: дуга вдоль шельфа, поперечный коридор, петля у мыса, pingpong в проливе — не копия NW–SE диагоналей демо.
- Разный `mode`: `loop` для патруля, `open` для транзита, `pingpong` для рыболовов.
- Разные `spawn`: `route` + `route_frac` vs `corner`; разный `player_clearance` / угол ownship.

**Задачи (objectives + COMM)**

- Не сводить миссию к тройке «sink sub + sink/ID surface + ID merchant». Варианты:
  - **только разведка** (ID + TMA, без уничтожения);
  - **ROE-смена** (сначала HOLD, потом `reveal_objective` + разрешение на огонь);
  - **сопровождение / прикрытие** союзника (`ally_*` + «не допустить потопления»);
  - **временное окно** (достичь точки / перехватить до `at_sec`);
  - **EMCON / тишина** (не спалить себя пингом до этапа);
  - **выбор ветки** (`outputs` → разные `require_var` и **разный** состав вражеских сил в миссии 2).
- Скрытые цели — не только «оказывается, танкер надо топить»: маскировка под гражданского, ложный контакт, поздний ввод корабля через `events`.

**Быстрый self-check перед сдачей**

```
[ ] Нет миссии с тройкой: потопить дизель + Grisha/Krivak «как в демо» + идентифицировать танкер
[ ] signature_id и unit.id не совпадают с демо-набором без веской причины сюжета
[ ] Маршруты построены под новую карту, а не перенесены координатами с Catalina
[ ] objectives/COMM описывают другую операцию (другие глаголы, ROE, twist)
[ ] render_theater_routes: нет красных shore hits на превью
[ ] Существующие сценарии пользователя (Taiwan и др.) не изменены
```

## Workflow

```
Task Progress:
- [ ] Concept: tone, arc, 2–5 missions, branching vars — **не клон Catalina по задачам/юнитам**
- [ ] Theater: reuse or bathymetry-and-routes
- [ ] Cover: image model → base64
- [ ] Draft JSON per schema (units, routes, objectives, events) — en first; **оригинальный состав сил и задач**
- [ ] Diversity pass: units / routes / objectives ≠ демо-тройка (Foxtrot+Grisha+tanker)
- [ ] **Render validation**: `go run ./tools/render_theater_routes.go` → 0 shore hits, spawn zone OK
- [ ] **Mission review loop**: для каждой миссии — PNG + инспекция → явный аппрув пользователя
- [ ] RU adaptation pass: Gemini (or other strong model) → apply to JSON
- [ ] Validate mentally vs schema; bump version
- [ ] Write to scenarios_generated/<id>.json
- [ ] Tell user to IMPORT SCENARIO in-game
- [ ] **Ask to finalize**: все миссии аппрувнуты → спросить, можно ли завершить генерацию
- [ ] **Cleanup** (только после согласия на завершение): см. §«Уборка»
```

Эталон **структуры** и ветвления: `tools/gen_demo_scenario_json.go` + `scenarios/demo_catalina.json` (не копировать контент миссий).

## Headless replay player (AFK forecast)

Для просмотра **headless-симуляции** (игрок бездействует) есть отдельный плеер. Агент **не** запускает запись и плеер сам по себе — **предложи пользователю** сгенерировать replay и открыть плеер, когда миссия готова к балансировке.

### Сборка бинарника

Бинарник **`ssn688-player`** лежит в **корне репо** (в `.gitignore`, в git не коммитится). Перед первым запуском или после правок `tools/sim_player/` / `internal/simreplay/` / `internal/theaterpreview/`:

```bash
# из корня репозитория
go build -o ssn688-player ./tools/sim_player/
```

Агент: после **любых** правок плеера или его зависимостей — **пересобери** (не полагайся на старый бинарник):

```bash
go build -o ssn688-player ./tools/sim_player/
```

Перед первым запуском у пользователя, если `./ssn688-player` отсутствует — та же команда. Альтернатива без бинарника: `go run ./tools/sim_player` (медленнее старт).

### Команды

```bash
# Превью карты (если ещё нет PNG)
go run ./tools/render_theater_routes.go -scenario scenarios_generated/<id>.json

# Сборка плеера (после правок — всегда)
go build -o ssn688-player ./tools/sim_player/

# Плеер
./ssn688-player

# Явные пути
./ssn688-player -scenario scenarios_generated/<id>.json -replay scenarios_generated/sim_replays/<mission>.replay.json

# Только перезаписать replay (без окна)
./ssn688-player -record-only -mission tw_attribution
```

**Автовыбор файлов** (если флаги не заданы):

- Сценарий: единственный `scenarios_generated/*.json` — берётся автоматически; если несколько — **системный диалог** выбора файла (macOS `osascript` / Linux `zenity`).
- Replay: после выбора сценария — единственный `.replay.json` с тем же `scenario_id` в `sim_replays/`, иначе **диалог списка** миссий/replay.

Карта в плеере: **MMB/RMB — pan**, **колесо — zoom**, **F — fit** (как PLOT).

Replay: `scenarios_generated/sim_replays/<mission_id>.replay.json` (не в git). Запись по умолчанию — **90 минут** (`-max-min`, default `90`).

**Источники данных плеера:**

| Что | Файл / флаг |
|-----|-------------|
| Сценарий + миссия (instantiate, bathy, запись replay) | авто: единственный `scenarios_generated/*.json` или диалог; `-scenario`, `-mission` при записи |
| Воспроизведение (кадры симуляции) | авто: replay с тем же `scenario_id` в `sim_replays/` или диалог; `-replay` |
| Фон карты (PNG) | `scenarios_generated/theater_previews/*__<mission>.png` (из `render_theater_routes`) |

При уборке сессии можно удалить `.replay.json` и PNG-превью; **`tools/sim_player/` не трогать**.

### Что показывает плеер

- Фон — тот же PNG, что `render_theater_routes` (bathy, берег, маршруты).
- Юниты и **оружие в полёте** — как **DEBUG** на тактической карте в игре: торпеды (MK48 / SET40 / LW), Harpoon (HSM), RBU (с линией к точке всплеска), Rastrub (RSTR), вспышки пуска (`RBU>`, …).
- Таблица: сторона, id, имя, статус (PATROL, SUNK, …), DEFCON.
- Управление: Play/Pause, ±10 мин, x1–x32, шкала времени.

После просмотра пользователь может попросить правки crew_skill, маршрутов или DEFCON — затем перезапись `-record`.

## Уборка после завершения генерации

Когда сценарий **закончен** — контент в `scenarios_generated/<id>.json`, пользователь **аппрувнул все миссии** и **явно согласился** завершить генерацию (см. §«Завершение генерации») — **обязательно прибери за собой** временные артефакты. Не оставляй мусор «на потом».

Cover, bathy и превью для игры уже не нужны на диске: всё нужное inline в JSON (`data_b64`); превью — только для валидации в сессии.

| Удалить (артефакты **этой** генерации) | Примеры |
|----------------------------------------|---------|
| Превью карт | `scenarios_generated/theater_previews/*.png`, `scenarios_generated/theater_previews/manifest.json` |
| Replay AFK (опционально) | `scenarios_generated/sim_replays/<mission_id>.replay.json` — если пользователь не просил оставить |
| Cover до inline | `tools/<scenario>_cover.png`, `tools/*_cover.jpg` |
| Промежуточный bathy | `scenarios_generated/theater_bathy/<theater_id>.bin`, `tools/bathy_<theater>.bin` |
| Сводка зон | `scenarios_generated/theater_bathy/zones_summary.txt` — если создавалась только для этой работы |
| Одноразовые генераторы / патчеры | `tools/gen_<scenario>_*.go`, `tools/patch_<scenario>_*.go`, `tools/gen_<theater>_bathy.py`, созданные **только** под этот сценарий |
| Временный кэш ETOPO | `tools/.etopo_cache/`, скачанные csv/nc **только** для этих театров |

**Не удалять** общие инструменты и пакеты репозитория:

- `tools/gen_hormuz_bathy.py`, `tools/gen_bathy_zone.py`, `tools/gen_demo_scenario_json.go`, `tools/render_theater_routes.go`
- **`tools/sim_player/`** — исходники headless replay-плеера; бинарник **`ssn688-player`** в корне репо (не в git)
- `internal/simreplay/`, `internal/theaterpreview/` — библиотеки записи/карты для плеера
- vendor, демо `bathy_catalina.bin`

Паттерны `gen_<scenario>_*.go` / `patch_<scenario>_*.go` **не** затрагивают `sim_player` — но явно не трогай каталог `tools/sim_player/` при уборке.

**Не удалять** `scenarios_generated/<id>.json` — это результат для пользователя.

Если пользователь явно просит **оставить пайплайн пересборки** (bathy `.bin`, patch-скрипты) — не трогай их; превью и cover после inline всё равно убери.

Перед уборкой убедись, что финальный JSON записан и `version` bumped; рендер-валидация уже пройдена.

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

Events: `when.type` (`time`, `objective_identified`, `enemy_prosecutes_allies`, …) + **`require_event` / `unless_event`** — полный каталог триггеров и `actions` в [reference.md §Event triggers / Event actions](reference.md#event-triggers-whentype). Build-time ветвление — `require_var` / `unless_var` / `var_eq`. Завершение миссии может ждать `end_after_event`.

## Чеклист качества

- [ ] Все player-facing тексты — `loc_string` на **en** и **ru** (и будущие SupportedLangs)
- [ ] Длинные `ru` (backstory, briefings, COMM, debrief, postscript) прошли адаптацию через **Gemini или другую сильную модель** — не сырая калька
- [ ] Placeholders `{{unit.*}}` и markdown в `ru` не сломаны после замены
- [ ] Backstory без спойлеров миссий
- [ ] Mission description = только стартовый intel
- [ ] Эскалация сложности по арке
- [ ] **Оригинальность**: нет клона демо (дизель + Grisha + танкер); свои unit.id, маршруты, objectives
- [ ] Вторички влияют на следующую миссию, но оставляют win-path
- [ ] Маршруты в воде; ПЛ ниже термоклина (если не coastal-special)
- [ ] **Рендер-валидация**: `tools/render_theater_routes.go` → 0 `route_land_hits`, зона PLAYER на карте ок
- [ ] **Аппрув миссий**: пользователь принял превью **каждой** миссии
- [ ] Хотя бы несколько reactive COMM / events
- [ ] Файл в `scenarios_generated/`, не в git
- [ ] `format_version` 3.x.x, осмысленный `version`
- [ ] После согласия на завершение: убраны temp (превью, bathy.bin, cover, одноразовые gen_*/patch_*)
