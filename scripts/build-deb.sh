#!/bin/bash
set -e

# Build DEB package for Linux
# Architecture: Shell + Core (systemd)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$SCRIPT_DIR/build-deb"
PAYLOAD_DIR="$BUILD_DIR/pkg"
CONTROL_DIR="$PAYLOAD_DIR/DEBIAN"
SYSTEMD_DIR="$PAYLOAD_DIR/lib/systemd/system"
APP_DIR="$PAYLOAD_DIR/usr/local/bin"
SHARE_DIR="$PAYLOAD_DIR/usr/share/aliang"
VERSION="${VERSION:-1.0.0}"
ARCH="${ARCH:-amd64}"

echo "=== Building Aliang DEB Package ==="
echo "Project dir: $PROJECT_DIR"
echo "Version: $VERSION"
echo "Arch: $ARCH"

# Clean previous build
rm -rf "$BUILD_DIR"
mkdir -p "$CONTROL_DIR"
mkdir -p "$SYSTEMD_DIR"
mkdir -p "$APP_DIR"
mkdir -p "$SHARE_DIR"
mkdir -p "$PAYLOAD_DIR/usr/share/applications"
mkdir -p "$PAYLOAD_DIR/var/lib/aliang"
mkdir -p "$PAYLOAD_DIR/var/log/aliang"

# Step 1: Copy pre-built binary
echo "=== Copying binary ==="
cp "$SCRIPT_DIR/aliang" "$APP_DIR/aliang"
chmod 755 "$APP_DIR/aliang"

# Step 2: Create control file
echo "=== Creating DEB control file ==="
cat > "$CONTROL_DIR/control" << CONTROL_EOF
Package: aliang
Version: $VERSION
Section: net
Priority: optional
Maintainer: Aliang <nursor@aliang.one>
Description: Aliang Gateway Proxy Client
 Aliang is a proxy gateway client that provides secure and fast
 network access with automatic proxy switching and rule-based routing.
Homepage: https://aliang.one
Architecture: $ARCH
Depends: libc6 (>= 2.34)
Control: aliang
CONTROL_EOF

# Step 3: Create systemd unit
echo "=== Creating systemd unit ==="
cat > "$SYSTEMD_DIR/aliang.service" << 'SERVICE_EOF'
[Unit]
Description=Aliang Core Service
Documentation=https://aliang.one
After=network-online.target
Wants=network-online.target
ConditionPathExists=/usr/local/bin/aliang

[Service]
Type=simple
ExecStart=/usr/local/bin/aliang core
WorkingDirectory=/var/lib/aliang
Restart=always
RestartSec=5
Environment=ALIANG_DATA_DIR=/var/lib/aliang
Environment=ALIANG_LOG_DIR=/var/log/aliang
Environment=ALIANG_SOCKET_PATH=/run/aliang-core.sock

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/run /var/lib/aliang /var/log/aliang

[Install]
WantedBy=multi-user.target
SERVICE_EOF

# Step 4: Create postinst script
echo "=== Creating postinst script ==="
cat > "$CONTROL_DIR/postinst" << 'POSTINST_EOF'
#!/bin/bash
set -e

case "$1" in
    configure)
        echo "Configuring Aliang service..."

        # Create directories
        mkdir -p /var/lib/aliang
        mkdir -p /var/log/aliang

        if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
            systemctl daemon-reload || true
            systemctl enable aliang >/dev/null 2>&1 || true
            systemctl restart aliang >/dev/null 2>&1 || systemctl start aliang >/dev/null 2>&1 || true
        fi

        # Update icon cache for desktop environment
        if command -v gtk-update-icon-cache >/dev/null 2>&1; then
            gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || true
        fi

        # Update desktop database for menu
        if command -v update-desktop-database >/dev/null 2>&1; then
            update-desktop-database /usr/share/applications 2>/dev/null || true
        fi

        echo "Aliang service configured and started"
        ;;
esac

exit 0
POSTINST_EOF
chmod 755 "$CONTROL_DIR/postinst"

# Step 5: Create prerm script
echo "=== Creating prerm script ==="
cat > "$CONTROL_DIR/prerm" << 'PRERM_EOF'
#!/bin/bash
set -e

case "$1" in
    remove|purge)
        echo "Stopping Aliang service..."
        if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
            systemctl stop aliang 2>/dev/null || true
            systemctl disable aliang 2>/dev/null || true
            systemctl daemon-reload 2>/dev/null || true
        fi
        ;;
esac

exit 0
PRERM_EOF
chmod 755 "$CONTROL_DIR/prerm"

# Step 6: Copy icon and desktop file
echo "=== Copying assets ==="
if [ -f "$SCRIPT_DIR/logo.png" ]; then
    # Install icon to standard locations for desktop environment recognition
    mkdir -p "$PAYLOAD_DIR/usr/share/pixmaps"
    mkdir -p "$PAYLOAD_DIR/usr/share/icons/hicolor/48x48/apps"
    mkdir -p "$PAYLOAD_DIR/usr/share/icons/hicolor/128x128/apps"
    mkdir -p "$PAYLOAD_DIR/usr/share/icons/hicolor/256x256/apps"

    cp "$SCRIPT_DIR/logo.png" "$PAYLOAD_DIR/usr/share/pixmaps/aliang.png"
    cp "$SCRIPT_DIR/logo.png" "$PAYLOAD_DIR/usr/share/icons/hicolor/48x48/apps/aliang.png"
    cp "$SCRIPT_DIR/logo.png" "$PAYLOAD_DIR/usr/share/icons/hicolor/128x128/apps/aliang.png"
    cp "$SCRIPT_DIR/logo.png" "$PAYLOAD_DIR/usr/share/icons/hicolor/256x256/apps/aliang.png"

    # Update icon cache (will be done by postinst)
fi

cat > "$SHARE_DIR/aliang.desktop" << 'DESKTOP_EOF'
[Desktop Entry]
Name=Aliang
Comment=Aliang Gateway Proxy Client
Exec=/usr/local/bin/aliang
Icon=aliang
Terminal=false
Type=Application
Categories=Network;Proxy;
StartupNotify=true
DESKTOP_EOF

# Copy desktop file to standard location for desktop environment
cp "$SHARE_DIR/aliang.desktop" "$PAYLOAD_DIR/usr/share/applications/aliang.desktop"

# Step 7: Build DEB package
echo "=== Building DEB package ==="
cd "$BUILD_DIR"

# Create debian-binary file
echo "2.0" > debian-binary

# Create control archive
ar rcs "aliang_${VERSION}_${ARCH}.deb" \
    debian-binary \
    "$CONTROL_DIR/control" \
    "$CONTROL_DIR/postinst" \
    "$CONTROL_DIR/prerm"

# For proper DEB, use dpkg-deb
dpkg-deb --build --root-owned-path "$PAYLOAD_DIR" "aliang_${VERSION}_${ARCH}.deb" 2>/dev/null || {
    # Fallback: create using tar
    mkdir -p "$BUILD_DIR/DEBIAN"
    cp "$CONTROL_DIR/control" "$BUILD_DIR/DEBIAN/"
    cp "$CONTROL_DIR/postinst" "$BUILD_DIR/DEBIAN/"
    cp "$CONTROL_DIR/prerm" "$BUILD_DIR/DEBIAN/"
    chmod 755 "$BUILD_DIR/DEBIAN/"*

    # Create data tarball
    cd "$PAYLOAD_DIR"
    tar -czf "$BUILD_DIR/data.tar.gz" usr/

    # Create control tarball
    cd "$BUILD_DIR"
    tar -czf control.tar.gz DEBIAN/control DEBIAN/postinst DEBIAN/prerm

    # Create deb package using ar
    rm -f "aliang_${VERSION}_${ARCH}.deb"
   ar rc "aliang_${VERSION}_${ARCH}.deb" debian-binary
    ar r "aliang_${VERSION}_${ARCH}.deb" data.tar.gz control.tar.gz
}

# Move to scripts directory
mv "aliang_${VERSION}_${ARCH}.deb" "$SCRIPT_DIR/"

echo ""
echo "=== Build Complete ==="
echo "DEB Package: $SCRIPT_DIR/aliang_${VERSION}_${ARCH}.deb"
echo ""
echo "Installation:"
echo "  - Binary installed to /usr/local/bin/aliang"
echo "  - systemd service at /lib/systemd/system/aliang.service"
echo "  - Data directory: /var/lib/aliang"
echo "  - Run: sudo dpkg -i aliang_${VERSION}_${ARCH}.deb"
