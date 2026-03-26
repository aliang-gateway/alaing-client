package models

// RunStartRequest is the request body for starting service
type RunStartRequest struct {
	InnerToken string `json:"inner_token"`
}

// RunStopRequest is the request body for stopping service
type RunStopRequest struct {
	// No fields needed for stop
}

type RunUserInfoRequest struct {
	UserUUID   string `json:"user_uuid,omitempty"`
	InnerToken string `json:"inner_token,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
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
