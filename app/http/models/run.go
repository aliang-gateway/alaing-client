package models

// RunStartRequest is the request body for starting service
type RunStartRequest struct {
	InnerToken string `json:"inner_token"`
}

// RunStopRequest is the request body for stopping service
type RunStopRequest struct {
	// No fields needed for stop
}

// RunUserInfoRequest is the request body for setting user info
// Note: User information is now automatically obtained through Token activation.
// These parameters are optional and are primarily used for backward compatibility.
// For new implementations, use /api/auth/activate to activate a token and get user info.
type RunUserInfoRequest struct {
	UserUUID   string `json:"user_uuid,omitempty"`     // Optional: normally auto-populated via token activation
	InnerToken string `json:"inner_token,omitempty"`   // Optional: normally auto-populated via token activation
	Username   string `json:"username,omitempty"`      // Optional: normally auto-populated via token activation
	Password   string `json:"password,omitempty"`      // Optional: normally auto-populated via token activation
}

// RunStatusResponse is the response body for getting run status
type RunStatusResponse struct {
	CurrentMode    string   `json:"current_mode"`
	IsRunning      bool     `json:"is_running"`
	AvailableModes []string `json:"available_modes"`
	Status         string   `json:"status,omitempty"`
	Description    string   `json:"description,omitempty"`
}

// RunSwiftRequest is the request body for swift mode switching
type RunSwiftRequest struct {
	TargetMode string `json:"target_mode"`
}

// RunMode represents the current operation mode
type RunMode string

const (
	ModeHTTP RunMode = "http"
	ModeTUN  RunMode = "tun"
)
