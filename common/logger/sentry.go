package logger

import (
	"log"

	"github.com/getsentry/sentry-go"
)

func init() {
	// httpTransport := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }

	// sentryTransport := sentry.NewHTTPTransport()
	// sentryTransport.Transport = &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }

	err := sentry.Init(sentry.ClientOptions{
		Dsn: "http://4307e08db9bad95bd9f55122cefe2fc3@sentry.nursor.org/6",
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for tracing.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
		// Transport:        sentryTransport,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

}
