package runner

import (
	"os"
	"strings"
	"sync"
	"time"
)

var TunSignal = make(chan os.Signal, 1)
var RunStatusChan = make(chan map[string]string, 1)

type StartupProgress struct {
	Active             bool   `json:"active"`
	Status             string `json:"status"`
	Phase              string `json:"phase"`
	Progress           int    `json:"progress_percent"`
	Message            string `json:"message"`
	Error              string `json:"error,omitempty"`
	PermissionRequired bool   `json:"permission_required"`
	UpdatedAt          int64  `json:"updated_at"`
}

var (
	startupProgressMu sync.RWMutex
	startupProgress   = StartupProgress{
		Active:    false,
		Status:    "idle",
		Phase:     "idle",
		Progress:  0,
		Message:   "TUN startup has not started.",
		UpdatedAt: time.Now().Unix(),
	}
)

func ResetStartupProgress() {
	setStartupProgress(StartupProgress{
		Active:    false,
		Status:    "idle",
		Phase:     "idle",
		Progress:  0,
		Message:   "TUN startup has not started.",
		UpdatedAt: time.Now().Unix(),
	})
}

func UpdateStartupProgress(status string, phase string, progress int, message string, errMsg string, permissionRequired bool) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	setStartupProgress(StartupProgress{
		Active:             status == "starting",
		Status:             status,
		Phase:              phase,
		Progress:           progress,
		Message:            message,
		Error:              errMsg,
		PermissionRequired: permissionRequired,
		UpdatedAt:          time.Now().Unix(),
	})
}

func FailStartupProgress(phase string, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	UpdateStartupProgress("failed", phase, 100, "TUN startup failed.", message, isPermissionLikeError(message))
}

func CompleteStartupProgress(message string) {
	UpdateStartupProgress("success", "running", 100, message, "", false)
}

func GetStartupProgress() StartupProgress {
	startupProgressMu.RLock()
	defer startupProgressMu.RUnlock()
	return startupProgress
}

func setStartupProgress(progress StartupProgress) {
	startupProgressMu.Lock()
	defer startupProgressMu.Unlock()
	startupProgress = progress
}

func isPermissionLikeError(message string) bool {
	lowered := strings.ToLower(strings.TrimSpace(message))
	return strings.Contains(lowered, "access is denied") ||
		strings.Contains(lowered, "operation not permitted") ||
		strings.Contains(lowered, "administrator") ||
		strings.Contains(lowered, "elevation") ||
		strings.Contains(lowered, "runas")
}
