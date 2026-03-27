//go:build !windows

package tray

import _ "embed"

//go:embed icon-active.png
var iconDataActive []byte

//go:embed icon-inactive.png
var iconDataInActive []byte

// GetIcon returns the application icon bytes for the active state.
func GetIcon() []byte {
	return iconDataActive
}

// GetIconDisabled returns the application icon bytes for the inactive state.
func GetIconDisabled() []byte {
	return iconDataInActive
}
