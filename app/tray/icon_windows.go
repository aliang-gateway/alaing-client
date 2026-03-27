//go:build windows

package tray

import _ "embed"

// Windows systray icons must be valid ICO bytes because systray uses
// LoadImageW with IMAGE_ICON under the hood.

//go:embed icon-active.ico
var iconDataActive []byte

//go:embed icon-inactive.ico
var iconDataInActive []byte

// GetIcon returns the application icon bytes for the active state.
func GetIcon() []byte {
	return iconDataActive
}

// GetIconDisabled returns the application icon bytes for the inactive state.
func GetIconDisabled() []byte {
	return iconDataInActive
}
