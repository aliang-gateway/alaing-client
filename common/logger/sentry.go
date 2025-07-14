package logger

import (
	"log"

	"github.com/getsentry/sentry-go"
)

func init() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "http://4307e08db9bad95bd9f55122cefe2fc3@sentry.nursor.org/6",
		TracesSampleRate: 0.1,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// sentry.ConfigureScope(func(scope *sentry.Scope) {
	// 	scope.SetTag("role", "user")
	// 	scope.SetTag("user_id", "25")
	// })
}
