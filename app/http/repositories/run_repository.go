package repositories

import (
	"nursor.org/nursorgate/app/http/services"
)

// RunRepositoryImpl provides access to run mode functionality
type RunRepositoryImpl struct {
	runService *services.RunService
}

// NewRunRepository creates a new run repository instance
func NewRunRepository() *RunRepositoryImpl {
	return &RunRepositoryImpl{
		runService: services.NewRunService(),
	}
}

// GetCurrentMode gets the current operating mode
func (rr *RunRepositoryImpl) GetCurrentMode() string {
	return rr.runService.GetCurrentMode()
}

// SetCurrentMode sets the operating mode
func (rr *RunRepositoryImpl) SetCurrentMode(mode string) {
	rr.runService.SetCurrentMode(mode)
}

// IsTunRunning checks if TUN service is running
func (rr *RunRepositoryImpl) IsTunRunning() bool {
	return rr.runService.IsTunRunning()
}

// SetTunRunning sets the TUN running state
func (rr *RunRepositoryImpl) SetTunRunning(running bool) {
	rr.runService.SetTunRunning(running)
}

// StartService starts the service for the current mode
func (rr *RunRepositoryImpl) StartService(innerToken string) map[string]interface{} {
	return rr.runService.StartService(innerToken)
}

// StopService stops the current running service
func (rr *RunRepositoryImpl) StopService() map[string]interface{} {
	return rr.runService.StopService()
}

// SetUserInfo sets user information
func (rr *RunRepositoryImpl) SetUserInfo(userUUID, innerToken, username, password string) map[string]interface{} {
	return rr.runService.SetUserInfo(userUUID, innerToken, username, password)
}

// GetStatus returns the current service status
func (rr *RunRepositoryImpl) GetStatus() map[string]interface{} {
	return rr.runService.GetStatus()
}

// SwitchMode switches the operating mode
func (rr *RunRepositoryImpl) SwitchMode(targetMode string) map[string]interface{} {
	return rr.runService.SwitchMode(targetMode)
}
