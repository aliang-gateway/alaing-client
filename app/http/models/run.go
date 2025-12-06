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
type RunUserInfoRequest struct {
	UserUUID   string `json:"user_uuid"`
	InnerToken string `json:"inner_token"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// RunStatusResponse is the response body for getting run status
type RunStatusResponse struct {
	CurrentMode    string   `json:"current_mode"`
	TunRunning     bool     `json:"tun_running"`
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
