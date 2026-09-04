#!/usr/bin/env bash
# Build Android debug APK (requires ANDROID_HOME, JDK 17, ebitenmobile).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -n "${JAVA_HOME:-}" && -x "${JAVA_HOME}/bin/java" ]]; then
  :
else
  JAVA17="${JAVA17:-/opt/homebrew/opt/openjdk@17}"
  if [[ ! -x "$JAVA17/bin/java" ]]; then
    echo "Need JDK 17: set JAVA_HOME or JAVA17=..." >&2
    exit 1
  fi
  export JAVA_HOME="$JAVA17"
fi
export PATH="$JAVA_HOME/bin:${HOME}/go/bin:/opt/homebrew/bin:${PATH:-}"

export ANDROID_HOME="${ANDROID_HOME:-/opt/homebrew/share/android-commandlinetools}"
export ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$ANDROID_HOME}"
if [[ -z "${ANDROID_NDK_HOME:-}" ]]; then
  if [[ -d "$ANDROID_HOME/ndk/26.3.11579264" ]]; then
    export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/26.3.11579264"
  else
    # Prefer newest installed NDK under ANDROID_HOME/ndk.
    newest="$(ls -1d "$ANDROID_HOME"/ndk/* 2>/dev/null | sort -V | tail -n 1 || true)"
    if [[ -n "$newest" ]]; then
      export ANDROID_NDK_HOME="$newest"
    fi
  fi
fi

if [[ -z "${ANDROID_NDK_HOME:-}" || ! -d "$ANDROID_NDK_HOME" ]]; then
  echo "NDK not found (set ANDROID_NDK_HOME)" >&2
  exit 1
fi

if ! command -v ebitenmobile >/dev/null; then
  go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest
fi

cp VERSION internal/version/VERSION

AAR="$ROOT/mobile/android/ssn688lib/ssn688.aar"
mkdir -p "$(dirname "$AAR")"
echo "==> ebitenmobile bind → $AAR"
ebitenmobile bind -target android -androidapi 23 -javapkg com.bubnov.ssn688 \
  -o "$AAR" ./mobile

printf 'sdk.dir=%s\n' "$ANDROID_HOME" > mobile/android/local.properties

echo "==> gradle assembleDebug"
(
  cd mobile/android
  chmod +x gradlew
  ./gradlew assembleDebug --no-daemon
)

OUT="$ROOT/dist/ssn688-android-debug.apk"
mkdir -p "$ROOT/dist"
cp mobile/android/app/build/outputs/apk/debug/app-debug.apk "$OUT"
ls -lh "$OUT"
echo "Install: adb install -r $OUT"
