package server

import (
	"nursor.org/nursorgate/common/logger"
)

// LogService handles log retrieval and management operations
type LogService struct{}

// NewLogService creates a new log service instance
func NewLogService() *LogService {
	return &LogService{}
}

// GetLogs retrieves logs from the buffer with filtering
func (ls *LogService) GetLogs(params LogsQueryParams) []LogEntryResponse {
	// Convert level string to LogLevelType
	var level logger.LogLevelType
	switch params.Level {
	case "TRACE":
		level = logger.TRACE
	case "DEBUG":
		level = logger.DEBUG
	case "INFO":
		level = logger.INFO
	case "WARN":
		level = logger.WARN
	case "ERROR":
		level = logger.ERROR
	case "FATAL":
		level = logger.FATAL
	case "PANIC":
		level = logger.PANIC
	default:
		level = 0 // All levels
	}

	// Get logs from buffer
	entries := logger.GetBufferEntries(params.Limit, level, params.Source)

	// Convert to response format
	var responses []LogEntryResponse
	for _, entry := range entries {
		responses = append(responses, LogEntryResponse{
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
	var level logger.LogLevelType

	switch levelStr {
	case "TRACE":
		level = logger.TRACE
	case "DEBUG":
		level = logger.DEBUG
	case "INFO":
		level = logger.INFO
	case "WARN":
		level = logger.WARN
	case "ERROR":
		level = logger.ERROR
	case "FATAL":
		level = logger.FATAL
	case "PANIC":
		level = logger.PANIC
	default:
		return 0, ErrInvalidLogLevel
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
