#!/bin/bash
# Build SSN688.app — launches without a Terminal window on macOS.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP_NAME="SSN688"
APP_DIR="$ROOT/dist/${APP_NAME}.app"
BIN="$APP_DIR/Contents/MacOS/${APP_NAME}"

mkdir -p "$APP_DIR/Contents/MacOS"

echo "Building $BIN ..."
(cd "$ROOT" && go build -o "$BIN" .)
chmod +x "$BIN"

cat > "$APP_DIR/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>SSN688</string>
	<key>CFBundleIdentifier</key>
	<string>com.ssn688.sim</string>
	<key>CFBundleName</key>
	<string>SSN688</string>
	<key>CFBundleDisplayName</key>
	<string>SSN-688(I) Hunter/Killer</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
</dict>
</plist>
PLIST

echo "Done: $APP_DIR"
echo "Launch: open \"$APP_DIR\""
