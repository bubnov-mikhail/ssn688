#!/usr/bin/env python3
"""Generate Russian officer voice lines with Silero TTS.

Outputs to internal/audio/voices/ru/{dept}/{filename} (PCM16 mono 44.1 kHz),
matching English clip stems under voices/{dept}/.

Requires: torch, torchaudio, soundfile, scipy (and Silero deps via torch.hub).
Example:
  source .venv-tts/bin/activate
  pip install torch torchaudio soundfile scipy omegaconf
  python scripts/generate_voices_silero_ru.py
  python scripts/generate_voices_silero_ru.py unable_deeper.wav
"""

from __future__ import annotations

import sys
from pathlib import Path

import numpy as np
import soundfile as sf
import torch
from scipy import signal

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "internal" / "audio" / "voices" / "ru"
TARGET_SR = 44100
SILERO_SR = 48000
MODEL_ID = "v5_ru"

# Distinct Silero speakers per compartment (Russian-only voices).
VOICES = {
    "capt": "aidar",    # male — CO
    "sonar": "xenia",   # female — sonar
    "weps": "eugene",   # male — weapons
    "dive": "aidar",    # male — diving officer
    "nav": "kseniya",   # female — navigation/time
}

# (dept, filename, Russian line) — stems match EN Kokoro clips (no enemy_ping FX).
LINES: list[tuple[str, str, str]] = [
    ("capt", "hold_simulation.wav", "Пауза симуляции."),
    ("capt", "save_complete.wav", "Сохранение завершено."),
    ("capt", "ownship_hit.wav", "Попадание по своему кораблю. Системы повреждены."),
    ("capt", "critical_damage.wav", "Критические повреждения. Отказ системы."),
    ("capt", "ownship_lost.wav", "Корабль потерян. Мы тонем."),
    ("capt", "comm_message.wav", "Срочное сообщение. Входящая передача."),
    ("capt", "comm_traffic_waiting.wav", "Ожидается срочная связь. Поднимите мачту связи."),
    ("sonar", "passive_on.wav", "Пассивный сонар включён."),
    ("sonar", "passive_off.wav", "Пассивный сонар выключен."),
    ("sonar", "active_standby.wav", "Активный сонар в режиме ожидания."),
    ("sonar", "active_online.wav", "Активный сонар включён."),
    ("sonar", "deploy_towed.wav", "Выпускаю буксируемую антенну."),
    ("sonar", "towed_held.wav", "Буксируемая антенна удерживается."),
    ("sonar", "retract_towed.wav", "Убираю буксируемую антенну."),
    ("sonar", "bt_launch.wav", "Запускаю батитермограф."),
    ("sonar", "layer_survey_complete.wav", "Съёмка слоя завершена."),
    ("sonar", "contact_classified.wav", "Контакт классифицирован."),
    ("weps", "impact_confirmed.wav", "Попадание оружия подтверждено."),
    ("weps", "torpedo_in_water.wav", "Торпеда в воде."),
    ("weps", "torpedo_heading_ownship.wav", "Торпеда на нас!"),
    ("weps", "outer_door_closed.wav", "Наружная крышка закрыта."),
    ("weps", "run_depth_set.wav", "Глубина хода задана."),
    ("weps", "speed_high.wav", "Скорость торпеды высокая."),
    ("weps", "speed_low.wav", "Скорость торпеды низкая."),
    ("weps", "seeker_on.wav", "Головка самонаведения включена."),
    ("weps", "seeker_off.wav", "Головка самонаведения выключена."),
    ("weps", "wire_cut.wav", "Провод обрезан."),
    ("weps", "outer_door_open_1.wav", "Наружная крышка открыта, труба один."),
    ("weps", "outer_door_open_2.wav", "Наружная крышка открыта, труба два."),
    ("weps", "outer_door_open_3.wav", "Наружная крышка открыта, труба три."),
    ("weps", "outer_door_open_4.wav", "Наружная крышка открыта, труба четыре."),
    ("weps", "torpedo_away_1.wav", "Торпеда ушла, труба один."),
    ("weps", "torpedo_away_2.wav", "Торпеда ушла, труба два."),
    ("weps", "torpedo_away_3.wav", "Торпеда ушла, труба три."),
    ("weps", "torpedo_away_4.wav", "Торпеда ушла, труба четыре."),
    ("dive", "come_left.wav", "Лево руля, есть."),
    ("dive", "come_right.wav", "Право руля, есть."),
    ("dive", "make_depth.wav", "Занять глубину, есть."),
    ("dive", "hold_depth.wav", "Держать глубину, есть."),
    ("dive", "unable_deeper.wav", "Глубже нельзя — упор дна."),
    ("nav", "speed_half.wav", "Ускорение времени половинное."),
    ("nav", "speed_normal.wav", "Ускорение времени нормальное."),
    ("nav", "speed_double.wav", "Ускорение времени двойное."),
    ("nav", "speed_quad.wav", "Ускорение времени четырёхкратное."),
    ("nav", "speed_eight.wav", "Ускорение времени восьмикратное."),
]


def to_pcm16_mono(samples: np.ndarray, sr: int, target_sr: int = TARGET_SR) -> tuple[np.ndarray, int]:
    mono = np.asarray(samples, dtype=np.float32)
    if mono.ndim > 1:
        mono = mono.mean(axis=-1)
    if sr != target_sr:
        n = int(round(len(mono) * target_sr / sr))
        mono = signal.resample(mono, n).astype(np.float32)
    peak = float(np.max(np.abs(mono))) if mono.size else 0.0
    if peak > 1e-6:
        mono = mono / peak * 0.92
    return mono, target_sr


def load_silero(device: torch.device):
    model, _ = torch.hub.load(
        repo_or_dir="snakers4/silero-models",
        model="silero_tts",
        language="ru",
        speaker=MODEL_ID,
        trust_repo=True,
    )
    model.to(device)
    return model


def main() -> int:
    only = {a.lower() for a in sys.argv[1:]} if len(sys.argv) > 1 else None

    device = torch.device("cpu")
    print(f"Loading Silero TTS {MODEL_ID} on {device}…")
    model = load_silero(device)

    selected = [
        (dept, filename, text)
        for dept, filename, text in LINES
        if only is None
        or filename.lower() in only
        or f"{dept}/{filename}".lower() in only
        or f"ru/{dept}/{filename}".lower() in only
        or Path(filename).stem.lower() in only
    ]
    if only and not selected:
        print(f"ERROR: no lines matched {sorted(only)}", file=sys.stderr)
        return 1

    for i, (dept, filename, text) in enumerate(selected, 1):
        speaker = VOICES[dept]
        out_path = OUT / dept / filename
        out_path.parent.mkdir(parents=True, exist_ok=True)
        print(f"[{i}/{len(selected)}] ru/{dept}/{filename}  speaker={speaker}")
        with torch.inference_mode():
            audio = model.apply_tts(
                text=text,
                speaker=speaker,
                sample_rate=SILERO_SR,
                put_accent=True,
                put_yo=True,
            )
        if isinstance(audio, torch.Tensor):
            samples = audio.detach().cpu().numpy()
        else:
            samples = np.asarray(audio, dtype=np.float32)
        mono, sr = to_pcm16_mono(samples, SILERO_SR)
        sf.write(out_path, mono, sr, subtype="PCM_16")
        print(f"  -> {out_path.relative_to(ROOT)}")

    print(f"Done. Generated {len(selected)} Russian clips with Silero.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
