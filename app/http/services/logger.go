package services

import (
	"errors"
	"strings"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/common/logger"
)

// Custom errors for logger operations
var (
	ErrInvalidLogLevel       = errors.New("invalid log level")
	ErrInvalidDurationFormat = errors.New("invalid duration format")
	ErrNoValidConfigFields   = errors.New("no valid configuration fields provided")
	ErrInvalidRequestBody    = errors.New("invalid request body")
)

// LogService handles log retrieval and management operations
type LogService struct{}

// NewLogService creates a new log service instance
func NewLogService() *LogService {
	return &LogService{}
}

// GetLogs retrieves logs from the buffer with filtering
func (ls *LogService) GetLogs(params models.LogsQueryParams) []models.LogEntryResponse {
	// Convert level string to LogLevelType
	var level logger.LogLevelType
	if params.Level != "" {
		parsedLevel, err := StringToLogLevelType(params.Level)
		if err == nil {
			level = parsedLevel
		} else {
			level = 0 // All levels on invalid input
		}
	}

	// Get logs from buffer
	entries := logger.GetBufferEntries(params.Limit, level, params.Source)

	// Convert to response format
	var responses []models.LogEntryResponse
	for _, entry := range entries {
		responses = append(responses, models.LogEntryResponse{
			Level:     LogLevelTypeToString(entry.Level),
			Timestamp: entry.Timestamp.Format("2006-01-02 15:04:05.000"),
			Message:   entry.Message,
			Source:    entry.Source,
			TraceID:   entry.TraceID,
		})
	}

	return responses
}

// ClearLogs clears the log buffer
func (ls *LogService) ClearLogs() error {
	logger.ClearBuffer()
	return nil
}

// UpdateLogLevel updates only the log level
func (ls *LogService) UpdateLogLevel(levelStr string) (logger.LogLevelType, error) {
	levelUpStr := strings.ToUpper(levelStr)
	level, err := StringToLogLevelType(levelUpStr)
	if err != nil {
		return 0, err
	}

	logger.UpdateLogLevel(level)
	return level, nil
}

// SubscribeLogStream subscribes to real-time log stream
// Returns a channel that receives log entries
func (ls *LogService) SubscribeLogStream() (<-chan *logger.LogEntry, func()) {
	logChan := make(chan *logger.LogEntry, 100)

	observer := func(entry *logger.LogEntry) {
		select {
		case logChan <- entry:
		default:
			// Channel full, drop the entry
		}
	}

	logger.GetGlobalBuffer().Subscribe(observer)

	// Return channel and cleanup function
	cleanup := func() {
		close(logChan)
	}

	return logChan, cleanup
}

// LogLevelTypeToString converts logger.LogLevelType to string
func LogLevelTypeToString(level logger.LogLevelType) string {
	switch level {
	case logger.TRACE:
		return "TRACE"
	case logger.DEBUG:
		return "DEBUG"
	case logger.INFO:
		return "INFO"
	case logger.WARN:
		return "WARN"
	case logger.ERROR:
		return "ERROR"
	case logger.FATAL:
		return "FATAL"
	case logger.PANIC:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// StringToLogLevelType converts string to logger.LogLevelType
func StringToLogLevelType(levelStr string) (logger.LogLevelType, error) {
	switch levelStr {
	case "TRACE":
		return logger.TRACE, nil
	case "DEBUG":
		return logger.DEBUG, nil
	case "INFO":
		return logger.INFO, nil
	case "WARN":
		return logger.WARN, nil
	case "ERROR":
		return logger.ERROR, nil
	case "FATAL":
		return logger.FATAL, nil
	case "PANIC":
		return logger.PANIC, nil
	default:
		return 0, ErrInvalidLogLevel
	}
}

// maskSensitiveData masks sensitive information like DSN
func maskSensitiveData(data string) string {
	if data == "" {
		return ""
	}
	if len(data) <= 8 {
		return "***"
	}
	return data[:8] + "***"
}
