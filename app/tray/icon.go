package tray

import _ "embed"

//go:embed icon-active.png
var iconDataActive []byte

//go:embed icon-inactive.png
var iconDataInActive []byte

// GetIcon returns the application icon bytes
func GetIcon() []byte {
	return iconDataActive
}

// GetIconDisabled returns a grayed out icon for disabled state
func GetIconDisabled() []byte {
	return iconDataInActive
}
