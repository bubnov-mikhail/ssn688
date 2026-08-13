---
name: bathymetry-and-routes
description: >-
  Build and wire bathymetry charts (BATH/bathy.bin), terrain on PLOT, vector coastlines,
  AI patrol routes, and submarine placement below the thermocline. Use when adding a
  new map, regenerating assets/bathy.bin, authoring scenario routes, or placing units
  near shore.
---

# Батиметрия, рельеф и маршруты

Навык для новых локаций и сценариев: от сырых глубин до маршрутов AI, которые не выходят на берег.

## Когда применять

- Новая карта / другой viewport (не Santa Catalina)
- Пересборка `assets/bathy.bin`
- Новые `Route` в сценарии (транзит, патруль, гражданские коридоры)
- Стартовые позиции кораблей и ПЛ у берега
- Проверка, что ПЛ могут работать **ниже термоклина**, а не на мелководье (если сценарий не требует обратного)

## Откуда брать данные

| Источник | Назначение |
|----------|------------|
| **NOAA ETOPO 2022** (15 arc-sec) | Глобальная батиметрия/суша; в проекте — через ERDDAP |
| Скрипт [`tools/gen_hormuz_bathy.py`](../../tools/gen_hormuz_bathy.py) | Эталон пайплайна: скачать subset → ресample 250 yd → `assets/bathy.bin` |

Текущая сцена: **20×20 nm** вокруг **33.30°N, 118.45°W** (Santa Catalina). Имя скрипта историческое (`hormuz`).

Для другой локации в скрипте меняют:

- `CENTER_LAT`, `CENTER_LON` (долгота **0–360**, как в ETOPO: запад = 360 − |lon|)
- `LAT_MIN/MAX`, `LON_MIN/MAX` — bbox subset
- при необходимости `CELL_YD` (шаг сетки, сейчас **250 yd**)

Альтернативы ETOPO: GEBCO, локальные survey grids — главное привести к тому же **игровому формату** (см. ниже).

## Куда класть в игре

```
assets/bathy.bin          ← бинарник BATH v1 (коммитится в репо)
assets/embed.go           ← //go:embed bathy.bin
main.go                   ← LoadBathymetry → world.SetDefaultBathymetry
internal/world/scenario.go ← Scenario.Bathy = &DefaultBathy
internal/sim/engine.go    ← BottomDepthFt акустики из bathy под ownship
internal/ui/tactical.go   ← рельеф + векторный берег на PLOT
```

Пересборка после правки скрипта:

```bash
python tools/gen_hormuz_bathy.py
go build -o ssn688 .
```

Проверка загрузки:

```bash
go test ./internal/world/ -run Bathymetry -count=1
```

## Формат `bathy.bin` (BATH v1)

Парсер: [`internal/world/bathymetry.go`](../../internal/world/bathymetry.go) — `LoadBathymetry`.

| Поле | Тип | Смысл |
|------|-----|--------|
| magic | `BATH` | |
| version | `uint32` = 1 | |
| width, height | `uint32` | ячейки сетки |
| originX, originY | `float64` | SW угол карты, **ярды** (восток / север) |
| cellSize | `float64` | размер ячейки, **ярды** |
| depths | `float32[w×h]` | **положительное = глубина воды (ft)**; **≤ 0 = суша** |

Off-chart в рантайме: `DepthAtFt` → `-50` (как суша).  
Суша/отмель для AI: `cellDepthBlocked` — глубина ячейки **< 40 ft** (`bathymetry_shore.go`).

## Рельеф на PLOT и береговая линия (вектор)

- **Заливка глубин**: `drawTacticalBathymetry` — цвет по `DepthAtFt` (лог-шкала до ~6000 ft).
- **Берег (вектор)**: `buildCoastSegments` — **marching squares** по 2×2 ячейкам (вода/суша по знаку глубины), отрезки `coastSegment` рисуются в `drawTacticalCoastline`.

Для AI «острый» берег — **не** bilinear `DepthAtFt`, а **`IsSurfaceBlocked` / `IsShoreBlocked`** по **центрам ячеек** (`bathymetry_shore.go`). При авторинге маршрутов ориентируйся на те же проверки, что и `NavigableFor` / `DistanceToShoreYd`.

## Термоклин и глубины для ПЛ

Акустика: [`internal/acoustics/environment.go`](../../internal/acoustics/environment.go) — `DefaultEnvironment()` (один профиль на всю карту):

| Слой | Глубина (ft) |
|------|----------------|
| mixed | 0 – 240 |
| thermocline | 240 – 800 |
| deep | 800+ |

**Правило размещения ПЛ** (если сценарий не требует мелководья / перископной глубины):

1. Точка на карте: `NavigableFor(x, y, KindSubmarine, patrolDepth)`.
2. Дно: `bottom := bathy.DepthAtFt(x, y)` — запас под килем: **`bottom >= patrolDepth + 40`** (см. `NavigableFor`).
3. **Ниже термоклина**: `patrolDepth >= 800` (слой `deep`), если дно позволяет.
4. Практический минимум дна для патруля «под термоклином»: **`bottom >= 900 ft`** (800 + кил + запас; см. `clampSubToBottom` — отступ **60 ft** от дна).
5. После расстановки: **`clampSubToBottom(e, bathy)`** — не ставить OrderedDepth глубже, чем позволяет дно.

Исключения (явно в сценарии): перископная охота (~60 ft), лежание на дне у берега, мелководный choke — тогда **не** требовать 800+ ft.

Надводные корабли: `DepthFt = 0`, достаточно **`NavigableFor(..., KindSurfaceShip, 0)`** и **`DistanceToShoreYd >= clearance`**.

## Константы зазора от берега

| Константа | Значение | Где |
|-----------|----------|-----|
| `transitMinClearanceYd` | **1000 yd** | `diagonal_routes.go` — транзитные линии |
| `coastMinClearanceYd` | **1000 yd** | `coastal_routes.go` — прибрежные дуги |
| `shoreClearanceYd` (AI) | **800 yd** | `ai/shore_avoid.go` — look-ahead уход от берега |
| `ShoreRayMaxYd` | **6000 yd** | `DistanceToShoreYd` |

Новые маршруты проектируй с **`transitMinClearanceYd` (1000 yd)** минимум; для прибрежного патруля — `BuildCoastalLoop` + `projectToShoreBand`.

## Как строить маршруты

API: [`internal/world/route.go`](../../internal/world/route.go), хелперы в [`diagonal_routes.go`](../../internal/world/diagonal_routes.go), [`coastal_routes.go`](../../internal/world/coastal_routes.go).

### Типы

| Флаг | Поведение | Риск |
|------|-----------|------|
| **`PingPong: true`** | разворот на концах полилинии | **Предпочтительно** — нет «хорды» через сушу |
| `Looped: true` | последняя точка = первая, движение по кругу | **Опасно**: прямой отрезок last→first может пересечь сушу |

Для демо-транзитов и союзных патрулей используй **PingPong**, не Looped.

### Алгоритм для новой линии

1. Задать опорные точки в **ярдах** (X восток, Y север) внутри `BoundsYards()`.
2. Каждую точку прогнать через **`snapNavigableClear(bathy, x, y, transitMinClearanceYd)`** — сдвиг на воду с зазором от берега.
3. **`dedupeWaypoints`** — убрать дубликаты ближе **120 yd**.
4. Для **каждого сегмента** между соседними WP: выборка каждые ~**250 yd** — все точки `NavigableFor` + `DistanceToShoreYd >= clearance`.
5. Для **PingPong**: проверить, что **обратный проход** по тем же WP не ближе к берегу (особенно у мыса).
6. Зарегистрировать в `Scenario.Routes`, назначить юнитам через **`AssignRoute`** / `PlaceOnRouteFraction`.

### Готовые паттерны в коде

- **`BuildNWSETransit`** — NW→SE транзит с lateral offset (гражданские/вражеские коридоры).
- **`BuildAllyEdgePatrol`** — SE → SW по низу карты → NW по западному краю.
- **`BuildCoastalLoop`** — дуга вдоль берега с `NearestShoreBearingDeg` + `projectToShoreBand`.

### Расстановка юнитов на маршруте

- **`PlaceOnRouteFraction(e, route, t, bathy)`** — t∈[0,1]; при необходимости повторный snap.
- **`PlaceNearChartCorner`** — SW/SE/NW/NE с проверкой `NavigableFor` и дистанции до transit lanes.
- **`placeAwayFrom` / `placeAtBearing`** — fallback с проверкой суши.
- Подлодки: **`clampSubToBottom`** после размещения.

### Runtime (не заменяет хороший авторинг)

- Надводные AI: **`applyShoreAvoidance`** — если курс ведёт внутрь **800 yd** от берега, `SHORE_AVOID` + `InterruptRoute`.
- Это страховка; маршрут всё равно должен быть валиден **без** постоянного shore avoid.

## Чеклист перед сдачей карты/маршрутов

- [ ] `python tools/gen_hormuz_bathy.py` (или аналог) → `assets/bathy.bin` обновлён
- [ ] `go test ./internal/world/...` — bathy, routes, scenario placement
- [ ] На PLOT видны **глубины** и **береговая линия**; суша не «дырявая»
- [ ] Все WP маршрутов: `NavigableFor` + `DistanceToShoreYd >= 1000` (или заданный clearance)
- [ ] PingPong-маршруты: нет сегментов через `IsShoreBlocked`
- [ ] ПЛ на патруле (если не мелководный сценарий): **`OrderedDepth >= 800`**, дно **`>= ~900 ft`**
- [ ] `go build -o ssn688 .`

## Связанные тесты

```bash
go test ./internal/world/ -run 'Bathymetry|Route|Diagonal|TrainingScenario|Shore' -count=1
go test ./internal/ai/ -run Shore -count=1
```

## Не делать

- Не класть сырые GeoTIFF/CSV в `assets/` — только **`bathy.bin`** (или расширять `LoadBathymetry` осознанно).
- Не использовать **`Looped: true`** через сушу — только замкнутые **водные** петли или PingPong.
- Не полагаться только на bilinear глубину для «можно ли пройти» — используй **`NavigableFor`** / **`IsShoreBlocked`**.
- Не ставить ПЛ на **`DepthAtFt < 900`** для «глубокого» патруля без явного сценарного повода.
