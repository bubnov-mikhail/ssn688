# AGENTS.md — гайд для ИИ-агентов и разработчиков

Симулятор ПЛ **SSN-688 Modern Submarine Combat Simulator**: Go + [Ebiten v2](https://ebitengine.org/), модуль `github.com/ssn688/sim`.  
Подробности для игроков — в [README.md](README.md). Здесь — то, что нужно агенту, чтобы не ломать архитектуру и быстро находить код.

## Обязательные привычки агента

1. **После каждого изменения кода** пересобирай бинарник:
   ```bash
   cp VERSION internal/version/VERSION
   go build -o ssn688 .
   ```
2. **Перед сдачей фичи** прогоняй тесты затронутых пакетов (минимум):
   ```bash
   go test ./internal/<pkg>/...
   ```
   При широких правках: `go test ./...`
3. **Не коммить и не пушь**, пока пользователь явно не попросил.
4. **Не трогай** `.venv-tts/`, `dist/`, сгенерированные `.app`, чужие секреты. Бинарник `ssn688` в `.gitignore` — его можно пересобирать локально.
5. Меняй только то, что нужно задаче. Не рефакторь «заодно», не добавляй лишние markdown-файлы без запроса.
6. Стиль кода: существующие паттерны пакета, короткие комментарии «почему», без шумных docstrings на очевидное.
7. **Доменные навыки** — каталог [`agent-skills/`](agent-skills/README.md). Перед новой картой, `bathy.bin`, маршрутами или расстановкой у берега **прочитай подходящий `SKILL.md`** (см. таблицу в README каталога).

## Навыки агентов (`agent-skills/`)

Репозиторий хранит **специализированные инструкции** для агентов (отдельно от AGENTS.md):

| Путь | Тема |
|------|------|
| [`agent-skills/bathymetry-and-routes/SKILL.md`](agent-skills/bathymetry-and-routes/SKILL.md) | ETOPO → `bathy.bin`, рельеф/берег на PLOT, маршруты без выхода на сушу, глубины ПЛ ниже термоклина |

При совпадении задачи с описанием навыка — **сначала навык**, потом правки кода. Новые навыки добавляй в `agent-skills/<имя>/SKILL.md` и строку в [`agent-skills/README.md`](agent-skills/README.md).

## Стек и окружение

| | |
|---|---|
| Go | **1.25+** (`go.mod`) |
| UI / loop | `github.com/hajimehoshi/ebiten/v2` |
| Точка входа | `main.go` → `internal/ui.App` |
| Тик сима | `internal/sim` — `TickRate = 10` Hz |
| Сохранения | текстовые `.sav` через `internal/save` |
| Настройки | `~/Library/Application Support/ssn688/` (macOS) |

Запуск для разработки: `go run .` или `./ssn688` после сборки.  
macOS `.app`: `./scripts/build_macos_app.sh` → `open dist/SSN688.app`.

## Карта пакетов

```
main.go                 — Ebiten Game, audio context
assets/                 — embed: menu_bg, bathy.bin
scenarios/              — bundled JSON сценарии (*.json), embed через scenarios/embed.go
internal/
  sim/                  — игровой тик, связка world/AI/weapons/sonar
  world/                — Entity, Kind, батиметрия, signature library, runtime Scenario
  campaign/             — JSON-сценарии, театры, миссии, loadout, debrief, import
  version/              — embed VERSION для semver игры и ScenarioFormatMajor
  acoustics/            — Detect, SNR, waterfall, listen bands, TMA, contacts
  weapons/              — Mk48/Type53, seeker, CM (decoy/jitter/Nixie)
  ai/                   — враг/гражданские, уклонение от торпед + CM
  ui/                   — экраны F1–F6, PLOT, ввод, алерты
  render/ + layout/     — отрисовка и координаты панелей
  audio/                — голоса офицеров (embed WAV) + FX
  save/                 — сериализация Engine/Sonar/торпед
  config/               — settings.json, пути saves
scripts/                — macOS app, Kokoro TTS → voices
tools/                  — пересборка батиметрии
agent-skills/           — доменные навыки для ИИ-агентов (SKILL.md)
```

### Куда смотреть по задаче

| Задача | Файлы |
|--------|--------|
| Пассивный водопад / пеленг | `acoustics/bearing_waterfall.go`, `ui/waterfall.go`, `ui/passive_sonar.go` |
| Broadband vs HF (торпеды / корабли) | `acoustics/listen_band.go` + `waterfallListenBandKindGain` в `bearing_waterfall.go` |
| Контакты, классификация, TMA-луч | `acoustics/sonar.go`, `classifier.go`, `ui/tactical.go`, `ui/weps_ui.go` |
| Активный пинг / эхо | `acoustics/sonar.go`, `player_ping.go`, `ui/active_sonar_ui.go` |
| Торпеды, seeker, направленный пинг | `weapons/torpedo.go` (`ModeSearch`, `SeekConeHalfAngleDeg`, `LastPingTime`) |
| Приманки / jitter | `weapons/countermeasure.go`; на PLOT/активном PPI **не** рисуются |
| Угроза столкновения / «Incoming torpedo» | `world/collision.go`, `ui/app.go`, `ai/torpedo_evade.go` |
| Разгон судов | `world/entity.go` — `MaxSpeedAccelKtsPerSec` |
| Подписи шума (LOFAR) | `world/signatures.go`, `acoustics/source.go` |
| Голосовые клипы | `audio/clips.go`, `audio/voice.go`, `audio/voices/**` |
| Кампания / состав миссии | `scenarios/*.json`, `internal/campaign/instantiate.go`, `scenario_load.go`, `build.go` |
| Новая карта / bathy / маршруты | `agent-skills/bathymetry-and-routes/SKILL.md`, `tools/gen_hormuz_bathy.py`, `internal/world/bathymetry*.go`, `diagonal_routes.go`, `coastal_routes.go`, `scenarios/*.json` |

## Важные доменные правила (не ломать без явного запроса)

### Акустика и водопад

- `ListenBroadband` — охота на корабли/ПЛ; шум торпед на водопаде сильно глушится.
- `ListenHF` — охота на торпеды; корабли/ПЛ едва заметны.
- **Пинги** (корабль/ПЛ/торпеда) должны оставаться видимыми в **обоих** режимах.
- Пинг торпеды в `ModeSearch` — **направленный**: на водопаде только если рыба смотрит на слушателя (конус `SeekConeHalfAngleDeg`, сейчас ±35°). Реализация: `activePingAudibleOnWaterfall` в `bearing_waterfall.go`.
- Вражеские торпеды на WEPS/PLOT появляются как контакты **после** акустического обнаружения, не в момент пуска.

### TMA

- Оценка курса/скорости: `updateContactTMA` / `ContactTMAAccurate` в `acoustics/sonar.go`.
- Тёмно-серый луч курса на PLOT/WEPS при `TMAAccuracy >= tmaMinAccuracy` (сейчас **0.65**) и скорости ≥ 0.5 kts.

### Оружие и CM

- `weapons` **не** импортирует `acoustics` (цикл). Слой/atten для seeker передаётся колбэком из `sim`.
- Торпеды в акустику попадают как `Entity` через `Torpedo.AcousticEntity` / `Engine.AcousticEmitters`.
- AI ставит CM (ADC/jitter/Nixie) при **угрозе столкновения** (CPA), не просто «торпеда рядом».

### UI

- Экраны живут в `internal/ui`; геометрия панелей — `layout`, цвета/шрифты — `render`.
- Сохранение сессии / reload: `ui/session.go`, `save/save.go` — при новых полях контакта/торпеды обновляй сериализацию **и** тесты `save`.

### Сценарии (JSON) и версии

- Сценарии — **JSON-файлы**, не Go-код. Демо: `scenarios/demo_catalina.json` (embed в бинарник). Пользовательские: `~/Library/Application Support/ssn688/scenarios/` (рядом с saves). Формат документирован в [`scenarios/schema.json`](scenarios/schema.json).
- **Все бинарные ассеты** (обложка, bathy) — **inline base64** в том же JSON; `asset_ref` / `use_game_default` не поддерживаются.
- Bathymetry живёт только в сценарии. При save/reload chart берётся из theater выбранного сценария (`campaign.ResolveMissionBathy`); глобального `DefaultBathy` / `assets/bathy.bin` нет.
- Поля semver: `format_version` (major = схема JSON, см. `version.ScenarioFormatMajor`, сейчас **2**), `version` (версия сценария), `min_game_version` (минимальная версия игры).
- Маршруты: явные `waypoints` + `mode` (`open` | `pingpong` | `loop`); опционально `player_clearance` для расстановки ownship.
- **При правке сценария** инкрементируй `version` по semver (patch — правки контента, minor — новые миссии/поля без ломки, major — несовместимые изменения).
- **Версия игры** — файл `VERSION` в корне + копия `internal/version/VERSION` (embed). При **breaking change** в игре для сценариев: увеличь major в `VERSION`, обнули minor/patch; при смене JSON-схемы — увеличь `ScenarioFormatMajor` в `internal/version/version.go`.
- Несовместимые сценарии (старый `format_version` или `min_game_version` > текущей игры) — красные в списке, без кнопок запуска; saves для них не показываются и не грузятся.
- Импорт: главное меню **IMPORT SCENARIO** → валидация JSON, semver, дубликат (замена только если новая `version` ≥ установленной), базовая проверка base64-ассетов.
- События (`events` в JSON): схема для event-driven поведения (COMM, юниты, радио); на load применяются COMM-расписания через `ApplyCommEvents`.

## Импорты и границы

```
ui → sim, acoustics, weapons, world, audio, render, save, …
sim → acoustics, ai, weapons, world
ai → acoustics, weapons, world
acoustics → weapons (константы CM/blast), world
weapons → world          # не тянуть acoustics / ui / sim
world → (почти никого)
```

Новую логику сима клади в `sim`/`world`/`acoustics`/`weapons`, а не в `ui`, если это не чисто отображение.

## Тесты

- Рядом с кодом: `*_test.go`.
- Хелперы сущностей часто в `acoustics` test helpers (`testEntity`).
- После правок акустики/водопада:  
  `go test ./internal/acoustics/... -run 'Waterfall|ListenBand|TMA|Ping'`
- После WEPS/CM/торпед:  
  `go test ./internal/weapons/... ./internal/ai/...`
- После save-полей:  
  `go test ./internal/save/... ./internal/ui/ -run Session`

## Ассеты и голоса

- WAV офицеров — `internal/audio/voices/{capt,sonar,weps,dive,nav}/`, регистрация в `clips.go`.
- Перегенерация TTS (Apple Silicon, `mlx-audio`): см. README → `scripts/generate_voices_kokoro.py`.
- Батиметрия: только внутри `scenarios/*.json` (`theaters[].bathy.data_b64`). Пересборка сетки: `tools/gen_hormuz_bathy.py` → бинарник, затем inline в JSON (`tools/gen_demo_scenario_json.go`). Подробный пайплайн — [`agent-skills/bathymetry-and-routes/SKILL.md`](agent-skills/bathymetry-and-routes/SKILL.md).

## Коммиты (когда попросили)

- Сообщения — зачем изменение, не пересказ диффа.
- Не коммитить `ssn688`, `.venv-tts`, локальные сейвы, секреты.
- Не `git push` / force / amend без явной просьбы (см. правила пользователя в Cursor).

## Быстрый чеклист перед ответом «готово»

- [ ] Код в правильном пакете, без новых циклов импорта
- [ ] Тесты затронутых пакетов зелёные
- [ ] `go build -o ssn688 .` успешен
- [ ] Поведение согласовано с правилами выше (listen band, пинги, TMA, CM)
- [ ] Save/UI обновлены, если менялись персистентные поля
