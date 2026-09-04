#!/usr/bin/env bash
# Build SSN688.app — launches without a Terminal window on macOS, with Dock icon.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP_NAME="SSN688"
APP_DIR="$ROOT/dist/${APP_NAME}.app"
BIN="$APP_DIR/Contents/MacOS/${APP_NAME}"
ICON_SRC="$ROOT/assets/app_icon/icon.png"
RES_DIR="$APP_DIR/Contents/Resources"
ICONSET="$ROOT/dist/${APP_NAME}.iconset"
VER="$(tr -d '[:space:]' < "$ROOT/VERSION")"

mkdir -p "$APP_DIR/Contents/MacOS" "$RES_DIR"

echo "Sync VERSION ..."
cp "$ROOT/VERSION" "$ROOT/internal/version/VERSION"

echo "Building $BIN ..."
(cd "$ROOT" && go build -o "$BIN" .)
chmod +x "$BIN"

# Required for Finder to treat the bundle as an application.
printf 'APPL????' > "$APP_DIR/Contents/PkgInfo"

echo "Building AppIcon.icns ..."
rm -rf "$ICONSET"
mkdir -p "$ICONSET"

# Master → RGBA PNG (Finder is happier with alpha).
MASTER="$ROOT/dist/icon_master.png"
sips -s format png "$ICON_SRC" --out "$MASTER" >/dev/null
# Ensure square 1024
sips -z 1024 1024 "$MASTER" --out "$MASTER" >/dev/null

make_icon() {
  local name="$1"
  local px="$2"
  sips -z "$px" "$px" "$MASTER" --out "$ICONSET/${name}.png" >/dev/null
}

# Exact iconutil names (1x and 2x).
make_icon "icon_16x16" 16
make_icon "icon_16x16@2x" 32
make_icon "icon_32x32" 32
make_icon "icon_32x32@2x" 64
make_icon "icon_128x128" 128
make_icon "icon_128x128@2x" 256
make_icon "icon_256x256" 256
make_icon "icon_256x256@2x" 512
make_icon "icon_512x512" 512
make_icon "icon_512x512@2x" 1024

iconutil -c icns "$ICONSET" -o "$RES_DIR/AppIcon.icns"
rm -rf "$ICONSET" "$MASTER"

cat > "$APP_DIR/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>SSN688</string>
	<key>CFBundleIdentifier</key>
	<string>com.ssn688.sim</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>SSN688</string>
	<key>CFBundleDisplayName</key>
	<string>SSN-688 Modern Submarine Combat Simulator</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleSignature</key>
	<string>????</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>CFBundleShortVersionString</key>
	<string>${VER}</string>
	<key>CFBundleVersion</key>
	<string>${VER}</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
</dict>
</plist>
PLIST

# Bump bundle mtime + re-register so Finder/Dock pick up the new icon.
touch "$APP_DIR"
xattr -cr "$APP_DIR" 2>/dev/null || true
LSREG="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
if [[ -x "$LSREG" ]]; then
  "$LSREG" -f "$APP_DIR" >/dev/null 2>&1 || true
fi

echo "Done: $APP_DIR"
echo "Launch: open \"$APP_DIR\""
echo "Do not run ./ssn688 for Dock icon — only the .app bundle works on macOS."
echo "If Dock still shows a blank icon: killall Dock"
