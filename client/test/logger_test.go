package test

import (
	"errors"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"nursor.org/nursorgate/common/logger"
)

func TestLogger(t *testing.T) {
	sentry.CaptureException(errors.New("test"))
	logger.Error("test")
	sentry.Flush(2 * time.Second)
}
