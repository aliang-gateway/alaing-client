package logger

// LogSilent variable for backward compatibility
var LogSilent = "false"

// InitHttp initializes the HTTP logger (backward compatibility)
func InitHttp() error {
	// Logger is now initialized via factory, no-op here
	return nil
}

// SetHttpLogLevel sets the HTTP logger level (backward compatibility)
func SetHttpLogLevel(level LogLevel) {
	// Use new unified config
	cfg := GetLogConfig()
	switch level {
	case DEBUG_COMPAT:
		cfg.Level = DEBUG
	case INFO_COMPAT:
		cfg.Level = INFO
	case WARN_COMPAT:
		cfg.Level = WARN
	case ERROR_COMPAT:
		cfg.Level = ERROR
	}
	SetLogConfig(cfg)
}

// HttpDebug logs debug message to HTTP logger
func HttpDebug(v ...interface{}) {
	GetMainLogger().Debug(v...)
}

// HttpInfo logs info message to HTTP logger
func HttpInfo(v ...interface{}) {
	GetMainLogger().Info(v...)
}

// HttpWarn logs warn message to HTTP logger
func HttpWarn(v ...interface{}) {
	GetMainLogger().Warn(v...)
}

// HttpError logs error message to HTTP logger
func HttpError(v ...interface{}) {
	GetMainLogger().Error(v...)
}

// HttpTrace logs trace message to the unified logger.
func HttpTrace(v ...interface{}) {
	GetMainLogger().Trace(v...)
}
