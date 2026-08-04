# SSN-688(I) Hunter/Killer

Симулятор подводной лодки в духе **688(I) Hunter/Killer** на **Go** + [Ebiten](https://ebitengine.org/).

Сценарий: Santa Catalina (батиметрия ETOPO), пассив/актив, буксируемая антенна, Mk48 ADCAP с проводом, вражеский AI.

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
| `F4` | Картотека «подчерков» |
| `F5` | Fire Control (WEPS) |
| `F6` | Маневрирование |
| `M` | Тактическая карта |
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

### Fire Control (Mk48)
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
| WEPS UI | **COUNTERMEASURES**: счётчики DECOY / JITTER и кнопки запуска |

### Маневрирование
- `↑` / `↓` — скорость · `Q` / `E` — курс · `PgUp` / `PgDn` — глубина

Decoy (ADC) и jitter (jammer) слышны на пассивном водопаде и соблазняют/путают seeker врага; на PLOT, WEPS-карте и активном PPI не отображаются.
1. Надводный боевой корабль противника
2. Подводную лодку противника

Противник ищет, пингует и уклоняется от торпед.

## Сохранения

Текстовые `.sav` в каталоге приложения, например:
`~/Library/Application Support/ssn688/saves/` (macOS)

## Архитектура

```
internal/
  acoustics/  — SNR, термоклин, пассив/актив, blast washout
  ai/         — патруль, атака, уклонение от торпед
  audio/      — голоса офицеров + FX (embed WAV)
  config/     — настройки
  layout/     — координаты панелей UI
  render/     — отрисовка консолей
  save/       — текстовые сохранения
  sim/        — игровой тик
  weapons/    — Mk48, провод, seeker, decoy/jitter CM
  world/      — сущности, сценарий, батиметрия, подписи
  ui/         — экраны и ввод
assets/       — menu_bg.jpg, bathy.bin (embed)
scripts/      — macOS app, генерация голосов
tools/        — пересборка батиметрии
```

## Голоса офицеров

~40 WAV встроены в бинарник (`internal/audio/voices/`). Реплики — Kokoro (`mlx-audio` на Apple Silicon); `sonar/enemy_ping.wav` — отдельный FX-пинг (не TTS).

| Отсек | Голос Kokoro | Примеры |
|-------|--------------|---------|
| CAPT | `bm_george` | «Rig ship for silent running…» |
| SONAR | `af_bella` | «Passive sonar online» |
| WEPS | `am_michael` | «Torpedo away, tube 3» |
| DIVE | `am_fenrir` | «Make depth, aye» |
| NAV | `bf_emma` | «Time acceleration double» |

Субтитры подставляют точные значения (курс, глубина), когда в аудио — общая фраза.

Перегенерация (нужен venv с `mlx-audio`, `scipy`, `soundfile`):

```bash
python3 -m venv .venv-tts
source .venv-tts/bin/activate
pip install mlx-audio scipy soundfile
python scripts/generate_voices_kokoro.py
# или выборочно:
python scripts/generate_voices_kokoro.py unable_deeper.wav deploy_towed.wav
```

## Батиметрия

`assets/bathy.bin` — сетка глубин Catalina (NOAA ETOPO 2022). Пересборка:

```bash
python tools/gen_hormuz_bathy.py
```

(Имя скрипта историческое; viewport — Santa Catalina.)

## Дальнейшее развитие

- [ ] Симуляция перископа
- [x] Голосовые реплики офицеров (встроены в бинарник)
- [x] Акустика: термоклин, listen bands, blast washout
- [x] WEPS: провод, seeker, tube clear, уклонение AI
- [ ] Больше сценариев и классов кораблей
