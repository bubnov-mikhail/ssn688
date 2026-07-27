#!/usr/bin/env bash
# Generates officer voice lines with Kokoro neural TTS (mlx-audio on Apple Silicon).
# Fallback: macOS `say` if the TTS venv is missing.
#
# Setup (once):
#   brew install python@3.12 espeak-ng ffmpeg
#   /opt/homebrew/opt/python@3.12/bin/python3.12 -m venv .venv-tts
#   source .venv-tts/bin/activate
#   pip install 'mlx-audio' 'misaki[en]' soundfile scipy
#
# Then:
#   ./scripts/generate_voices.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VENV="$ROOT/.venv-tts"
PY="$VENV/bin/python"

if [[ -x "$PY" ]]; then
  echo "Using Kokoro neural TTS ($PY)"
  exec "$PY" "$ROOT/scripts/generate_voices_kokoro.py"
fi

echo "WARNING: .venv-tts not found — falling back to macOS say (synthetic)."
echo "See comments in this script for Kokoro setup."

OUT="$ROOT/internal/audio/voices"
RATE=168

gen() {
  local voice="$1"
  local file="$2"
  local text="$3"
  local dir
  dir="$(dirname "$file")"
  mkdir -p "$dir"
  local tmp
  tmp="$(mktemp -t ssn688-voice).aiff"
  say -v "$voice" -r "$RATE" -o "$tmp" "$text"
  ffmpeg -y -loglevel error -i "$tmp" -ar 44100 -ac 1 -sample_fmt s16 "$file"
  rm -f "$tmp"
  echo "  $file"
}

echo "Generating captain (Daniel)..."
gen Daniel "$OUT/capt/mission_brief.wav" "Rig ship for silent running. Locate and engage assigned targets."
gen Daniel "$OUT/capt/hold_simulation.wav" "Hold simulation."
gen Daniel "$OUT/capt/save_complete.wav" "Save complete."

echo "Generating sonar (Samantha)..."
gen Samantha "$OUT/sonar/passive_on.wav" "Passive sonar online."
gen Samantha "$OUT/sonar/passive_off.wav" "Passive sonar offline."
gen Samantha "$OUT/sonar/active_standby.wav" "Active sonar standby."
gen Samantha "$OUT/sonar/active_ping.wav" "Transmitting active pulse."

echo "Generating weapons (Fred)..."
gen Fred "$OUT/weps/impact_confirmed.wav" "Weapon impact confirmed."
gen Fred "$OUT/weps/outer_door_closed.wav" "Outer door closed."
gen Fred "$OUT/weps/gyro_set.wav" "Gyro angle set."
gen Fred "$OUT/weps/run_depth_set.wav" "Run depth set."
gen Fred "$OUT/weps/speed_high.wav" "Torpedo speed high."
gen Fred "$OUT/weps/speed_low.wav" "Torpedo speed low."
gen Fred "$OUT/weps/seeker_on.wav" "Seeker enabled."
gen Fred "$OUT/weps/seeker_off.wav" "Seeker disabled."
gen Fred "$OUT/weps/wire_cut.wav" "Wire cut."
for n in 1 2 3 4; do
  gen Fred "$OUT/weps/outer_door_open_${n}.wav" "Outer door open, tube ${n}."
  gen Fred "$OUT/weps/torpedo_away_${n}.wav" "Torpedo away, tube ${n}."
done

echo "Generating maneuvering (Ralph)..."
gen Ralph "$OUT/dive/come_left.wav" "Come left, aye."
gen Ralph "$OUT/dive/come_right.wav" "Come right, aye."
gen Ralph "$OUT/dive/make_depth.wav" "Make depth, aye."

echo "Generating navigation (Karen)..."
gen Karen "$OUT/nav/speed_half.wav" "Time acceleration one half."
gen Karen "$OUT/nav/speed_normal.wav" "Time acceleration normal."
gen Karen "$OUT/nav/speed_double.wav" "Time acceleration double."
gen Karen "$OUT/nav/speed_quad.wav" "Time acceleration quadruple."
gen Karen "$OUT/nav/speed_eight.wav" "Time acceleration eight times."

echo "Done. Generated $(find "$OUT" -name '*.wav' | wc -l | tr -d ' ') voice files."
