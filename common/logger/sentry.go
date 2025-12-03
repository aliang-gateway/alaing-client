package logger

import (
	"log"
	"sync"

	"github.com/getsentry/sentry-go"
)

var (
	sentryInitOnce sync.Once
)

// InitSentry initializes Sentry with the configured DSN
func InitSentry() {
	sentryInitOnce.Do(func() {
		config := GetLogConfig()

		// Skip Sentry initialization if DSN is not configured or disabled
		if !config.EnableSentry || config.SentryDSN == "" {
			return
		}

		err := sentry.Init(sentry.ClientOptions{
			Dsn:              config.SentryDSN,
			TracesSampleRate: 0.1,
		})
		if err != nil {
			log.Printf("sentry.Init: %s", err)
			// Don't fail fatally - logging should continue without Sentry
		}
	})
}

func init() {
	// Initialize Sentry from config with safety
	InitSentry()
}
