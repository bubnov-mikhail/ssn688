#!/usr/bin/env bash
# Double-click launcher: runs the .app (Dock icon) and closes Terminal on quit.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
APP="$DIR/dist/SSN688.app"
ICON_SRC="$DIR/assets/app_icon/icon.png"
ICNS="$APP/Contents/Resources/AppIcon.icns"
BIN="$APP/Contents/MacOS/SSN688"

need_build=0
if [[ ! -x "$BIN" ]]; then
	need_build=1
elif [[ ! -f "$ICNS" ]]; then
	need_build=1
elif [[ "$ICON_SRC" -nt "$ICNS" ]]; then
	need_build=1
fi

if [[ "$need_build" -eq 1 ]]; then
	echo "Building SSN688.app ..."
	"$DIR/scripts/build_macos_app.sh"
fi

# Prefer the .app so macOS shows AppIcon in Dock (bare ./ssn688 will not).
open -W "$APP"
osascript -e 'tell application "Terminal" to close front window' 2>/dev/null || true
