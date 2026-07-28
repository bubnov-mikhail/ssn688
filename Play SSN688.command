#!/bin/bash
# Double-click launcher: runs the game and closes this Terminal window when you quit.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
APP="$DIR/dist/SSN688.app"

if [[ ! -x "$APP/Contents/MacOS/SSN688" ]]; then
	echo "Building SSN688.app (first run)..."
	"$DIR/scripts/build_macos_app.sh"
fi

# -W blocks until the player quits; then we close this Terminal window.
open -W "$APP"
osascript -e 'tell application "Terminal" to close front window' 2>/dev/null || true
