# System Tray Module

This module provides a cross-platform system tray for Nonelane.

## Features

- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Right-click menu with common actions
- ✅ Start/Stop/Restart server
- ✅ Open web dashboard in browser
- ✅ Version information
- ✅ Graceful quit

## Usage

### Start with System Tray

```bash
# Start with tray icon
./nonelane tray

# Start with config file and tray
./nonelane tray --config ./config.json

# Start with tray and token
./nonelane tray --token your-token
```

### Menu Options

- **Open Dashboard**: Opens the web interface in your default browser
- **Start Server**: Starts the HTTP server
- **Stop Server**: Stops the HTTP server
- **Restart Server**: Restarts the HTTP server
- **Version**: Shows application version
- **Quit**: Exits the application

## Icon Setup

The tray icon is loaded from `icon.png` in this directory. To customize:

1. Create a PNG image (recommended: 64x64 or 128x128 pixels)
2. Name it `icon.png`
3. Place it in `app/tray/` directory
4. Rebuild the application

### Platform-Specific Icon Guidelines

- **Windows**: ICO format preferred, PNG works
- **macOS**: PNG with transparency
- **Linux**: PNG with transparency

## Architecture

```
app/tray/
├── tray.go       # Main tray application logic
├── icon.go       # Icon embedding and management
├── icon.png      # Icon file (embedded at build time)
└── README.md     # This file
```

## How It Works

1. When `nonelane tray` is executed, the `runTray` function in `cmd/tray.go` is called
2. It loads configuration (if provided)
3. It initializes the system tray using `systray.Run()`
4. The tray icon appears in the system tray area
5. Users can right-click to access the menu
6. Server starts automatically when tray is ready

## Dependencies

- `github.com/getlantern/systray`: Cross-platform system tray library

## Troubleshooting

### Linux

On Linux, you may need to install `libappindicator3-dev`:

```bash
# Ubuntu/Debian
sudo apt-get install libappindicator3-dev

# Fedora
sudo dnf install libappindicator-gtk3-devel

# Arch Linux
sudo pacman -S libappindicator
```

### macOS

No additional dependencies required.

### Windows

No additional dependencies required.

## Future Enhancements

- [ ] Add status indicator in the icon (running/stopped)
- [ ] Add configuration dialog
- [ ] Add log viewer
- [ ] Support for multiple themes
- [ ] Notifications for important events