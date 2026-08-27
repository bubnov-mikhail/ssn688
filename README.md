# SSN-688 Modern Submarine Combat Simulator

Симулятор подводной лодки класса **Los Angeles (SSN-688)** на **Go** + [Ebiten](https://ebitengine.org/).

Сценарий: **Santa Catalina** (батиметрия ETOPO), охота на надводные и подводные цели с пассивом/активом, буксируемой антенной, Mk48 / Harpoon и вражеским ASW AI.

## Возможности

### Сонар и акустика
- **Пассивный водопад** — Hull / TB-16 towed (~800 yd), пеленг с параллаксом буксира
- **Listen bands** — Broadband (корабли/ПЛ) и HF (торпеды); пинги видны в обоих режимах
- **Активный сонар** — PPI, одиночный пинг, эхо по целям
- **SPECTRUM** — LOFAR/тонали, классификация по профилю (ясность растёт с SNR / towed)
- **Термоклин, слой, blast washout** после взрывов
- **TMA** — оценка курса/скорости контакта; луч на PLOT/WEPS при достаточной точности

### Оружие и контрмеры
- **Mk48 ADCAP** — ТА 1–4, гиро/глубина/скорость, провод, seeker (ModeSearch после отхода от своей ПЛ), обрыв / self-destruct
- **UGM-84 Harpoon** — подводный выход, крейсер, SRCH/beam/destruct presets; ПВО противника (SAM/CIWS)
- **Soft-kill CM** — ADC (decoy) и jitter; слышны на пассиве, на PLOT/PPI не рисуются
- Быстрая панель **OWN SHIP / CM** на ключевых экранах

### Мачты и оптика (MAST, `F8`)
- **ESM** — перехват облучения радарами противника/нейтралов
- **COMM** — подъём/спуск мачты, **REPORT** (только при поднятой COMM)
- **Перископ** — подъём, поворот, IR-вид с аспектными спрайтами судов; риск среза мачт / повреждения

### Корабль и повреждения
- **HELM (`F6`)** — курс, глубина, машинный телеграф (вперёд / стоп / задний ход)
- **Damage Control (`F7`)** — системы, трубы, мачты, ремонт
- Сохранения `.sav` (состояние сима, сонар, торпеды, мачты)

### Тактика и мир
- **PLOT (`M`)** — тактическая карта, контакты, маркеры, TMA-лучи
- **LIBRARY (`F4`)** — картотека платформ: Udaloy, Krivak, Kresta II, Grisha, Kilo, Foxtrot, Victor III, гражданские
- **Вражеский ASW** — DEFCON (Aware → Hostile → Weapons Free); Rastrub→UMGT-1, RBU, корабельные ТА (SET-40/UMGT-1), тяжёлые 53-см рыбы у ПЛ
- Уклонение AI от торпед, постановка CM по угрозе CPA
- **Маршруты** гражданских/патрулей (waypoints, PingPong), **COLREGS** для надводного расхождения
- Голоса офицеров (CAPT / SONAR / WEPS / DIVE / NAV) + FX

## Запуск

### macOS

1. **Рекомендуется** — собрать `.app` (без окна Terminal):

```bash
./scripts/build_macos_app.sh
open dist/SSN688.app
```

2. Двойной клик по **`Play SSN688.command`** в корне — соберёт `.app` при первом запуске и закроет Terminal после выхода.

### Разработка

```bash
go run .
# или
go build -o ssn688 .
```

Нужен **Go 1.25+**. Работает на macOS, Linux и Windows.

## Управление

### Меню
- `↑` / `↓` — выбор · `Enter` — подтвердить · `Esc` — назад

### В игре
| Клавиша | Действие |
|---------|----------|
| `F1` | Пассивный сонар |
| `F2` | Активный сонар |
| `F3` | Анализатор спектра |
| `F4` | LIBRARY — картотека угроз |
| `F5` | WEPS — Fire Control |
| `F6` | HELM — маневрирование |
| `F7` | DC — Damage Control |
| `F8` | MAST — ESM / COMM / перископ |
| `M` | PLOT — тактическая карта |
| `Space` | Пауза |
| `+` / `-` | Ускорение / замедление времени |
| `Ctrl+S` / кнопка `SAVE` | Быстрое сохранение |
| `EXIT` (шапка) | Выход в главное меню, освобождение сессии |
| `Esc` | Пауза / снять паузу |

### Пассивный сонар
| Клавиша | Действие |
|---------|----------|
| `P` | Пассив вкл/выкл |
| `B` | Hull / towed array |
| `N` | Broadband / HF listen band |
| `U` / `Y` / `H` | Deploy / retract / hold towed |

TB-16 (~800 yd cable): TOWED waterfall uses array position (bearing parallax vs HULL). Long baseline shrinks passive range/bearing uncertainty (abeam best). Streamed cable: warn ~20 kn, shear ~24 kn (shorter scope tolerates more); parted array shows **DAMAGED — NO DATA** on TOWED.

### Активный сонар
- `A` — standby / online · `F` — одиночный импульс (и в standby)

### Спектр
- `←` / `→` — пеленг луча

### Fire Control (Mk48 / Harpoon / CM)
| Клавиша | Действие |
|---------|----------|
| `1`–`4` | Выбор ТА |
| `O` / `C` | Открыть / закрыть наружный люк |
| `G` / `D` | Гиро (+5°) / глубина хода (+50 ft) |
| `S` | Скорость LOW/HIGH |
| `H` | Seeker on/off (prep) |
| `Enter` | Пуск |
| `←` / `→` | Провод: курс ±10° |
| `W` | Обрыв провода |
| `X` | Self-destruct (пока провод жив) |
| WEPS UI | Harpoon prep · **COUNTERMEASURES**: DECOY / JITTER |

Decoy (ADC) и jitter слышны на пассивном водопаде и соблазняют/путают seeker врага; на PLOT, WEPS-карте и активном PPI не отображаются.

### HELM (маневрирование)
- `↑` / `↓` — скорость · `Q` / `E` — курс · `PgUp` / `PgDn` — глубина
- UI: машинный телеграф (вперёд / стоп / задний ход), станции HELM

### MAST
- ESM raise/lower · COMM raise/lower · **REPORT** (нужна поднятая COMM)
- Перископ: raise/lower, train left/right, IR optic

## Сохранения

Текстовые `.sav` в каталоге приложения, например:
`~/Library/Application Support/ssn688/saves/` (macOS)

## Архитектура

```
internal/
  acoustics/  — SNR, термоклин, пассив/актив, ESM/COMM/peri, blast
  ai/         — патруль/маршруты, DEFCON, COLREGS, атака, уклонение
  audio/      — голоса офицеров + FX (embed WAV)
  config/     — настройки
  layout/     — координаты панелей UI
  render/     — отрисовка консолей
  save/       — текстовые сохранения
  sim/        — игровой тик
  weapons/    — Mk48, Harpoon, Rastrub/RBU, seeker, CM
  world/      — сущности, маршруты, сценарий, батиметрия, damage
  ui/         — экраны F1–F8, PLOT, ввод
assets/       — menu_bg, bathy.bin, library/, peri_ships/ (embed)
scripts/      — macOS app, генерация голосов
tools/        — пересборка батиметрии
```

## Голоса офицеров

~40 WAV на язык встроены в бинарник (`internal/audio/voices/`, RU — `voices/ru/…`).
EN — Kokoro (`mlx-audio` на Apple Silicon); RU — Microsoft Edge neural TTS (`edge-tts`).
`sonar/enemy_ping.wav` — FX-пинг (не TTS; при RU UI играет EN-файл).

| Отсек | Kokoro (EN) | Edge (RU) | Примеры |
|-------|-------------|-----------|---------|
| CAPT | `bm_george` | `ru-RU-DmitryNeural` | «Hold simulation» / «Пауза симуляции» |
| SONAR | `af_bella` | `ru-RU-SvetlanaNeural` | «Passive sonar online» |
| WEPS | `am_michael` | `ru-RU-DmitryNeural` | «Torpedo away, tube 3» |
| DIVE | `am_fenrir` | `ru-RU-DmitryNeural` | «Make depth, aye» |
| NAV | `bf_emma` | `ru-RU-SvetlanaNeural` | «Time acceleration double» |

Субтитры подставляют точные значения (курс, глубина), когда в аудио — общая фраза.

Перегенерация EN (нужен venv с `mlx-audio`, `scipy`, `soundfile`):

```bash
python3 -m venv .venv-tts
source .venv-tts/bin/activate
pip install mlx-audio scipy soundfile
python scripts/generate_voices_kokoro.py
# или выборочно:
python scripts/generate_voices_kokoro.py unable_deeper.wav deploy_towed.wav
```

Перегенерация RU (edge-tts; те же stems, выход в `voices/ru/`; нужен `ffmpeg`):

```bash
source .venv-tts/bin/activate
pip install edge-tts soundfile scipy
python scripts/generate_voices_edge_ru.py
python scripts/generate_voices_edge_ru.py unable_deeper.wav
# handcrafted ElevenLabs clips are skipped unless:
python scripts/generate_voices_edge_ru.py --force ownship_hit.wav
```

## Credits / атрибуция

- **PASSIVE ambient:**
  [*ship waves hydrophones 03 190617_0038*](https://freesound.org/people/klankbeeld/sounds/609037/)
  by [klankbeeld](https://freesound.org/people/klankbeeld/) via [freesound.org](https://freesound.org/) (CC BY 4.0).
- **Bow wash (surface):**
  [*Under water sounds while scuba diving*](https://freesound.org/people/timsc/sounds/274491/)
  by [timsc](https://freesound.org/people/timsc/) (CC0).
- **Torpedo run:**
  [*S34-38 Torpedo traveling underwater*](https://freesound.org/people/craigsmith/sounds/675798/)
  by [craigsmith](https://freesound.org/people/craigsmith/) (CC0).
- **Other PASSIVE / WEPS FX** (launch, explosion, fishing/merchant/tanker/sub/combatant props):
  user-provided local assets; see `internal/audio/fx/CREDITS.txt`.
- **LIBRARY photos:** see `assets/library/CREDITS.txt`.
- **Periscope ship sprites:** see `assets/peri_ships/README.md`.

## Батиметрия

Глубины живут **внутри сценария** (`theaters[].bathy.data_b64` в JSON). Пересборка Catalina-сетки:

```bash
python tools/gen_hormuz_bathy.py          # → tools/bathy_catalina.bin
go run ./tools/gen_demo_scenario_json.go  # inline cover + bathy into scenarios/demo_catalina.json
```

(Имя скрипта историческое; viewport — Santa Catalina.)

## Дальнейшее развитие

- [x] Перископ / MAST (ESM, COMM, IR optic)
- [x] Голосовые реплики офицеров (встроены в бинарник)
- [x] Акустика: термоклин, listen bands, blast washout, TMA
- [x] WEPS: Mk48 + провод/seeker, Harpoon, CM soft-kill
- [x] Damage control, HELM reverse, threat library
- [x] Вражеский ASW (Rastrub / RBU / ТА / ПЛ) + DEFCON
- [ ] Больше сценариев и миссий
- [ ] Расширение классов / арсенала
