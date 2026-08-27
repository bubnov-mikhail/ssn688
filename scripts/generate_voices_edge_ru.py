#!/usr/bin/env python3
"""Generate Russian officer voice lines with Microsoft Edge neural TTS (edge-tts).

Outputs to internal/audio/voices/ru/{dept}/{filename} (PCM16 mono 44.1 kHz),
matching English clip stems under voices/{dept}/.

Handcrafted ElevenLabs clips listed in HANDCRAFTED are skipped unless --force.

Requires: edge-tts, soundfile, scipy, ffmpeg (for mp3→wav).
Example:
  source .venv-tts/bin/activate
  pip install edge-tts soundfile scipy
  python scripts/generate_voices_edge_ru.py
  python scripts/generate_voices_edge_ru.py unable_deeper.wav
  python scripts/generate_voices_edge_ru.py --force ownship_hit.wav
"""

from __future__ import annotations

import asyncio
import subprocess
import sys
import time
from pathlib import Path

import edge_tts
import numpy as np
import soundfile as sf
from scipy import signal

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "internal" / "audio" / "voices" / "ru"
TARGET_SR = 44100

# Edge neural voices per compartment (RU locale).
VOICES = {
    "capt": "ru-RU-DmitryNeural",    # male — CO
    "sonar": "ru-RU-SvetlanaNeural", # female — sonar
    "weps": "ru-RU-DmitryNeural",    # male — weapons
    "dive": "ru-RU-DmitryNeural",    # male — diving officer
    "nav": "ru-RU-SvetlanaNeural",   # female — navigation/time
}

# Slight urgency bump for danger callouts; routine stays neutral.
RATE = {
    "default": "+0%",
    "alarm": "+18%",
}

# ElevenLabs (or other) handcrafted WAVs — do not overwrite with edge-tts.
# Keys: "dept/filename.wav". Pass --force to regenerate these from Edge.
HANDCRAFTED: set[str] = {
    "capt/ownship_hit.wav",
    "capt/critical_damage.wav",
    "capt/ownship_lost.wav",
    "capt/comm_message.wav",
    "capt/comm_traffic_waiting.wav",
    "weps/impact_confirmed.wav",
    "weps/torpedo_in_water.wav",
    "weps/torpedo_heading_ownship.wav",
    "weps/outer_door_closed.wav",
    "weps/run_depth_set.wav",
    "weps/speed_high.wav",
    "weps/speed_low.wav",
    "weps/seeker_on.wav",
    "weps/seeker_off.wav",
    "weps/wire_cut.wav",
    "weps/outer_door_open_1.wav",
    "weps/outer_door_open_2.wav",
    "weps/outer_door_open_3.wav",
    "weps/outer_door_open_4.wav",
}

# (dept, filename, Russian line, rate_key)
LINES: list[tuple[str, str, str, str]] = [
    ("capt", "hold_simulation.wav", "Пауза симуляции.", "default"),
    ("capt", "save_complete.wav", "Сохранение завершено.", "default"),
    ("capt", "ownship_hit.wav", "Мы подбиты! Повреждения систем!", "alarm"),
    ("capt", "critical_damage.wav", "Внимание! Критические повреждения! Отказ системы!", "alarm"),
    ("capt", "ownship_lost.wav", "Корабль потерян! Мы тонем!", "alarm"),
    ("capt", "comm_message.wav", "Срочное сообщение. Входящая передача.", "default"),
    ("capt", "comm_traffic_waiting.wav", "Ожидается срочная связь. Поднимите мачту связи.", "default"),
    ("sonar", "passive_on.wav", "Пассивный сонар включён.", "default"),
    ("sonar", "passive_off.wav", "Пассивный сонар выключен.", "default"),
    ("sonar", "active_standby.wav", "Активный сонар в режиме ожидания.", "default"),
    ("sonar", "active_online.wav", "Активный сонар включён.", "default"),
    ("sonar", "deploy_towed.wav", "Выпускаю буксируемую антенну.", "default"),
    ("sonar", "towed_held.wav", "Буксируемая антенна удерживается.", "default"),
    ("sonar", "retract_towed.wav", "Убираю буксируемую антенну.", "default"),
    ("sonar", "bt_launch.wav", "Запускаю батитермограф.", "default"),
    ("sonar", "layer_survey_complete.wav", "Съёмка слоя завершена.", "default"),
    ("sonar", "contact_classified.wav", "Контакт классифицирован.", "default"),
    ("weps", "impact_confirmed.wav", "Попадание оружия подтверждено.", "default"),
    ("weps", "torpedo_in_water.wav", "Торпеда в воде! Торпеда в воде!", "alarm"),
    ("weps", "torpedo_heading_ownship.wav", "Внимание! На нас идет торпеда!", "alarm"),
    ("weps", "outer_door_closed.wav", "Наружная крышка закрыта.", "default"),
    ("weps", "run_depth_set.wav", "Глубина хода задана.", "default"),
    ("weps", "speed_high.wav", "Скорость торпеды высокая.", "default"),
    ("weps", "speed_low.wav", "Скорость торпеды низкая.", "default"),
    ("weps", "seeker_on.wav", "Головка самонаведения включена.", "default"),
    ("weps", "seeker_off.wav", "Головка самонаведения выключена.", "default"),
    ("weps", "wire_cut.wav", "Провод наведения обрезан.", "default"),
    ("weps", "outer_door_open_1.wav", "Наружная крышка открыта, труба один.", "default"),
    ("weps", "outer_door_open_2.wav", "Наружная крышка открыта, труба два.", "default"),
    ("weps", "outer_door_open_3.wav", "Наружная крышка открыта, труба три.", "default"),
    ("weps", "outer_door_open_4.wav", "Наружная крышка открыта, труба четыре.", "default"),
    ("weps", "torpedo_away_1.wav", "Торпеда ушла, труба один.", "default"),
    ("weps", "torpedo_away_2.wav", "Торпеда ушла, труба два.", "default"),
    ("weps", "torpedo_away_3.wav", "Торпеда ушла, труба три.", "default"),
    ("weps", "torpedo_away_4.wav", "Торпеда ушла, труба четыре.", "default"),
    ("dive", "come_left.wav", "Лево руля, есть.", "default"),
    ("dive", "come_right.wav", "Право руля, есть.", "default"),
    ("dive", "make_depth.wav", "Занять глубину, есть.", "default"),
    ("dive", "hold_depth.wav", "Держать глубину, есть.", "default"),
    ("dive", "unable_deeper.wav", "Глубже нельзя - ограничение глубины!", "default"),
    ("nav", "speed_half.wav", "Ускорение времени половинное.", "default"),
    ("nav", "speed_normal.wav", "Ускорение времени нормальное.", "default"),
    ("nav", "speed_double.wav", "Ускорение времени двойное.", "default"),
    ("nav", "speed_quad.wav", "Ускорение времени четырёхкратное.", "default"),
    ("nav", "speed_eight.wav", "Ускорение времени восьмикратное.", "default"),
]


def to_pcm16_mono(path: Path, target_sr: int = TARGET_SR) -> None:
    audio, sr = sf.read(path, dtype="float32", always_2d=True)
    mono = audio.mean(axis=1)
    if sr != target_sr:
        n = int(round(len(mono) * target_sr / sr))
        mono = signal.resample(mono, n).astype(np.float32)
    peak = float(np.max(np.abs(mono))) if mono.size else 0.0
    if peak > 1e-6:
        mono = mono / peak * 0.92
    sf.write(path, mono, target_sr, subtype="PCM_16")


async def synth_one(text: str, voice: str, rate: str, dest: Path, attempts: int = 8) -> None:
    dest.parent.mkdir(parents=True, exist_ok=True)
    tmp_mp3 = dest.with_suffix(".mp3")
    last_err: Exception | None = None
    for i in range(attempts):
        try:
            tmp_mp3.unlink(missing_ok=True)
            communicate = edge_tts.Communicate(text, voice, rate=rate)
            await communicate.save(str(tmp_mp3))
            if not tmp_mp3.exists() or tmp_mp3.stat().st_size < 200:
                raise RuntimeError("empty/missing mp3 from edge-tts")
            subprocess.check_call(
                [
                    "ffmpeg", "-y", "-i", str(tmp_mp3),
                    "-ac", "1", "-ar", str(TARGET_SR), str(dest),
                ],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            tmp_mp3.unlink(missing_ok=True)
            to_pcm16_mono(dest)
            return
        except Exception as e:
            last_err = e
            wait = min(30.0, 2.0 * (i + 1))
            print(f"  retry {i + 1}/{attempts}: {e} (sleep {wait:.0f}s)")
            await asyncio.sleep(wait)
    raise RuntimeError(f"failed after {attempts} attempts: {last_err}")


async def run(selected: list[tuple[str, str, str, str]], *, force: bool) -> None:
    skipped = 0
    generated = 0
    for i, (dept, filename, text, rate_key) in enumerate(selected, 1):
        key = f"{dept}/{filename}"
        out_path = OUT / dept / filename
        if not force and key in HANDCRAFTED:
            print(f"[{i}/{len(selected)}] SKIP handcrafted ru/{key}  (use --force to overwrite)")
            skipped += 1
            continue
        voice = VOICES[dept]
        rate = RATE[rate_key]
        print(f"[{i}/{len(selected)}] ru/{key}  voice={voice} rate={rate}")
        await synth_one(text, voice, rate, out_path)
        print(f"  -> {out_path.relative_to(ROOT)}")
        generated += 1
        # Be polite to the free endpoint.
        await asyncio.sleep(0.8)
    print(f"Generated {generated}, skipped handcrafted {skipped}.")


def main() -> int:
    args = [a for a in sys.argv[1:] if a]
    force = False
    if "--force" in args:
        force = True
        args = [a for a in args if a != "--force"]
    only = {a.lower() for a in args} if args else None
    selected = [
        row
        for row in LINES
        if only is None
        or row[1].lower() in only
        or f"{row[0]}/{row[1]}".lower() in only
        or f"ru/{row[0]}/{row[1]}".lower() in only
        or Path(row[1]).stem.lower() in only
    ]
    if only and not selected:
        print(f"ERROR: no lines matched {sorted(only)}", file=sys.stderr)
        return 1

    t0 = time.time()
    asyncio.run(run(selected, force=force))
    print(f"Done with edge-tts in {time.time() - t0:.0f}s.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
