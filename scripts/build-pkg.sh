#!/bin/bash
set -e

# Build PKG installer for macOS
# Architecture: Tray(.app) + Core(LaunchAgent)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$SCRIPT_DIR/build-pkg"
PAYLOAD_DIR="$BUILD_DIR/payload"
SCRIPTS_DIR="$BUILD_DIR/scripts"
APP_DIR="$PAYLOAD_DIR/Applications/Aliang.app"
VERSION="${VERSION:-1.0.0}"

echo "=== Building Aliang PKG Installer ==="
echo "Project dir: $PROJECT_DIR"
echo "Version: $VERSION"

# Clean previous build
rm -rf "$BUILD_DIR"
mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"
mkdir -p "$SCRIPTS_DIR"

# Step 1: Build the binary
echo "=== Building aliang binary ==="
cd "$PROJECT_DIR"

# On macOS, CGO is required for systray (Cocoa/Objective-C)
if [ "$(uname)" = "Darwin" ]; then
	go build -ldflags="-s -w" -o "$SCRIPT_DIR/aliang" ./cmd/aliang/main.go
else
	CGO_ENABLED=0 go build -ldflags="-s -w" -o "$SCRIPT_DIR/aliang" ./cmd/aliang/main.go
fi

# Step 2: Copy binary to app bundle
echo "=== Copying binary to app bundle ==="
cp "$SCRIPT_DIR/aliang" "$APP_DIR/Contents/MacOS/aliang"
chmod +x "$APP_DIR/Contents/MacOS/aliang"

# Step 3: Copy Info.plist and icon
echo "=== Copying Info.plist and icon ==="
cp "$SCRIPT_DIR/Info.plist" "$APP_DIR/Contents/Info.plist"
if [ -f "$SCRIPT_DIR/Aliang.icns" ]; then
    cp "$SCRIPT_DIR/Aliang.icns" "$APP_DIR/Contents/Resources/Aliang.icns"
    echo "=== App icon copied ==="
else
    echo "=== Warning: Aliang.icns not found, skipping icon ==="
fi

# Step 4: Create preinstall script
echo "=== Creating preinstall script ==="
cat > "$SCRIPTS_DIR/preinstall" << 'PREINSTALL_SCRIPT'
#!/bin/bash

echo "Preinstall: Stopping old services..."

# Get current user info
CURRENT_USER=$(whoami)
USER_ID=$(id -u "$CURRENT_USER")
echo "Preinstall: Running as user: $CURRENT_USER (UID: $USER_ID)"

# Stop and remove old tray agent if exists
echo "Preinstall: Stopping old tray agent..."
launchctl bootout "gui/${USER_ID}/one.aliang.tray" 2>&1 || true
rm -f "$HOME/Library/LaunchAgents/one.aliang.tray.plist" 2>&1 || true

# Stop and remove old core service if exists
echo "Preinstall: Stopping old core service..."
launchctl bootout "gui/${USER_ID}/one.aliang.core" 2>&1 || true
rm -f "$HOME/Library/LaunchAgents/one.aliang.core.plist" 2>&1 || true

echo "Preinstall: Old services cleaned up"
PREINSTALL_SCRIPT
chmod +x "$SCRIPTS_DIR/preinstall"

# Step 5: Create postinstall script
echo "=== Creating postinstall script ==="
cat > "$SCRIPTS_DIR/postinstall" << 'POSTINSTALL_SCRIPT'
#!/bin/bash

echo "Postinstall: Setting up core service..."

# Get current user info
CURRENT_USER=$(whoami)
USER_ID=$(id -u "$CURRENT_USER")
echo "Postinstall: Running as user: $CURRENT_USER (UID: $USER_ID)"

# Create log directory
LOG_DIR="$HOME/Library/Logs/Aliang"
mkdir -p "$LOG_DIR"
chmod 755 "$LOG_DIR"
echo "Postinstall: Log directory ready at $LOG_DIR"

# Create the plist directly in user LaunchAgents directory
USER_PLIST="$HOME/Library/LaunchAgents/one.aliang.core.plist"

cat > "$USER_PLIST" << 'PLIST_EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>one.aliang.core</string>
	<key>ProgramArguments</key>
	<array>
		<string>/Applications/Aliang.app/Contents/MacOS/aliang</string>
		<string>start</string>
	</array>
	<key>RunAtLoad</key>
	<false/>
	<key>KeepAlive</key>
	<false/>
	<key>WorkingDirectory</key>
	<string>/Applications/Aliang.app/Contents/MacOS</string>
	<key>StandardOutPath</key>
	<string>__LOGDIR__/core.log</string>
	<key>StandardErrorPath</key>
	<string>__LOGDIR__/core.error.log</string>
</dict>
</plist>
PLIST_EOF

# Fix log paths in the plist
sed -i '' "s|__LOGDIR__|$HOME/Library/Logs/Aliang|g" "$USER_PLIST" 2>/dev/null || true
echo "Postinstall: Plist created at $USER_PLIST"

# Bootstrap core service (register but don't start - RunAtLoad=false)
echo "Postinstall: Bootstrapping core service..."
if launchctl bootstrap "gui/${USER_ID}" "$USER_PLIST" 2>&1; then
    echo "Postinstall: Core service registered successfully"
else
    echo "Postinstall: WARNING - bootstrap returned non-zero (service may already be registered)"
fi

echo "Postinstall: Core service setup complete"
POSTINSTALL_SCRIPT
chmod +x "$SCRIPTS_DIR/postinstall"

# Step 6: No need to put plist in app bundle - postinstall creates it directly
# Just create the app bundle structure without the plist
echo "=== App bundle ready ==="

# Step 7: Build component package with pkgbuild
echo "=== Building component package ==="
pkgbuild --identifier org.nursor.aliang \
    --version "$VERSION" \
    --root "$PAYLOAD_DIR" \
    --scripts "$SCRIPTS_DIR" \
    --install-location "/" \
    "$BUILD_DIR/Aliang.pkg"

# Step 8: Create distribution package with productbuild
echo "=== Building distribution package ==="
productbuild --identifier org.nursor.aliang \
    --version "$VERSION" \
    --package "$BUILD_DIR/Aliang.pkg" \
    "$SCRIPT_DIR/Aliang-${VERSION}.pkg"

echo ""
echo "=== Build Complete ==="
echo "PKG Installer: $SCRIPT_DIR/Aliang-${VERSION}.pkg"
echo ""
echo "Installation:"
echo "  - Aliang.app will be installed to /Applications/Aliang.app"
echo "  - Core service plist will be registered (not started)"
echo "  - Opening Aliang.app will start the core service"
