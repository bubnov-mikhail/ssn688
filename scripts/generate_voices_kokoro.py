#!/usr/bin/env python3
"""Generate officer voice lines with Kokoro (mlx-audio) neural TTS."""

from __future__ import annotations

import sys
from pathlib import Path

import numpy as np
from mlx_audio.tts.generate import generate_audio
from mlx_audio.tts.utils import load_model
from scipy import signal
import soundfile as sf

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "internal" / "audio" / "voices"
MODEL = "mlx-community/Kokoro-82M-4bit"
TARGET_SR = 44100

# Distinct Kokoro voices per compartment (US/UK accents).
VOICES = {
    "capt": "bm_george",   # British male — CO
    "sonar": "af_bella",   # American female — sonar
    "weps": "am_michael",  # American male — weapons
    "dive": "am_fenrir",   # American male — diving officer
    "nav": "bf_emma",      # British female — navigation/time
}

# Slightly deliberate delivery for CIC radio chatter.
SPEED = {
    "capt": 0.95,
    "sonar": 1.0,
    "weps": 0.98,
    "dive": 0.97,
    "nav": 1.02,
}

LINES: list[tuple[str, str, str]] = [
    ("capt", "mission_brief.wav", "Rig ship for silent running. Locate and engage assigned targets."),
    ("capt", "hold_simulation.wav", "Hold simulation."),
    ("capt", "save_complete.wav", "Save complete."),
    ("sonar", "passive_on.wav", "Passive sonar online."),
    ("sonar", "passive_off.wav", "Passive sonar offline."),
    ("sonar", "active_standby.wav", "Active sonar standby."),
    ("sonar", "active_online.wav", "Active sonar online."),
    ("sonar", "active_ping.wav", "Transmitting active pulse."),
    ("sonar", "deploy_towed.wav", "Deploying towed array."),
    ("sonar", "towed_held.wav", "Towed array held."),
    ("sonar", "retract_towed.wav", "Retracting towed array."),
    ("sonar", "bt_launch.wav", "Launching bathythermograph."),
    ("sonar", "layer_survey_complete.wav", "Layer survey complete."),
    ("sonar", "contact_classified.wav", "Contact classified."),
    ("sonar", "enable_active_first.wav", "Enable active sonar before transmitting."),
    ("weps", "impact_confirmed.wav", "Weapon impact confirmed."),
    ("weps", "outer_door_closed.wav", "Outer door closed."),
    ("weps", "gyro_set.wav", "Gyro angle set."),
    ("weps", "run_depth_set.wav", "Run depth set."),
    ("weps", "speed_high.wav", "Torpedo speed high."),
    ("weps", "speed_low.wav", "Torpedo speed low."),
    ("weps", "seeker_on.wav", "Seeker enabled."),
    ("weps", "seeker_off.wav", "Seeker disabled."),
    ("weps", "wire_cut.wav", "Wire cut."),
    ("weps", "outer_door_open_1.wav", "Outer door open, tube one."),
    ("weps", "outer_door_open_2.wav", "Outer door open, tube two."),
    ("weps", "outer_door_open_3.wav", "Outer door open, tube three."),
    ("weps", "outer_door_open_4.wav", "Outer door open, tube four."),
    ("weps", "torpedo_away_1.wav", "Torpedo away, tube one."),
    ("weps", "torpedo_away_2.wav", "Torpedo away, tube two."),
    ("weps", "torpedo_away_3.wav", "Torpedo away, tube three."),
    ("weps", "torpedo_away_4.wav", "Torpedo away, tube four."),
    ("dive", "come_left.wav", "Come left, aye."),
    ("dive", "come_right.wav", "Come right, aye."),
    ("dive", "make_depth.wav", "Make depth, aye."),
    ("dive", "hold_depth.wav", "Holding depth, aye."),
    ("dive", "unable_deeper.wav", "Unable to dive deeper. Bottom limits ordered depth."),
    ("nav", "speed_half.wav", "Time acceleration one half."),
    ("nav", "speed_normal.wav", "Time acceleration normal."),
    ("nav", "speed_double.wav", "Time acceleration double."),
    ("nav", "speed_quad.wav", "Time acceleration quadruple."),
    ("nav", "speed_eight.wav", "Time acceleration eight times."),
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


def main() -> int:
    # Optional args: generate only matching filenames or dept/filename paths.
    # Example: python generate_voices_kokoro.py unable_deeper.wav deploy_towed.wav
    only = {a.lower() for a in sys.argv[1:]} if len(sys.argv) > 1 else None

    print(f"Loading Kokoro model: {MODEL}")
    model = load_model(MODEL)
    tmp_dir = ROOT / ".tts_tmp"
    tmp_dir.mkdir(exist_ok=True)

    selected = [
        (dept, filename, text)
        for dept, filename, text in LINES
        if only is None
        or filename.lower() in only
        or f"{dept}/{filename}".lower() in only
        or Path(filename).stem.lower() in only
    ]
    if only and not selected:
        print(f"ERROR: no lines matched {sorted(only)}", file=sys.stderr)
        return 1

    for i, (dept, filename, text) in enumerate(selected, 1):
        voice = VOICES[dept]
        speed = SPEED[dept]
        out_path = OUT / dept / filename
        out_path.parent.mkdir(parents=True, exist_ok=True)
        prefix = f"{dept}_{Path(filename).stem}"
        lang = "b" if voice.startswith(("bm_", "bf_")) else "a"
        print(f"[{i}/{len(selected)}] {dept}/{filename}  voice={voice}")
        generate_audio(
            text=text,
            model=model,
            voice=voice,
            speed=speed,
            lang_code=lang,
            file_prefix=str(tmp_dir / prefix),
            audio_format="wav",
            join_audio=True,
            play=False,
            verbose=False,
        )
        produced = tmp_dir / f"{prefix}.wav"
        if not produced.exists():
            # mlx-audio sometimes appends _000
            candidates = sorted(tmp_dir.glob(f"{prefix}*.wav"))
            if not candidates:
                print(f"ERROR: no output for {prefix}", file=sys.stderr)
                return 1
            produced = candidates[0]
        to_pcm16_mono(produced)
        produced.replace(out_path)
        print(f"  -> {out_path.relative_to(ROOT)}")

    for p in tmp_dir.glob("*"):
        p.unlink()
    tmp_dir.rmdir()
    print(f"Done. Generated {len(selected)} clips with Kokoro.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
