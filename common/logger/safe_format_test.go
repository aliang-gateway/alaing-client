package logger

import (
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

type panickingError struct{}

func (panickingError) Error() string {
	panic("boom")
}

type panickingWriter struct{}

func (panickingWriter) Write([]byte) (int, error) {
	panic("write boom")
}

func TestSafeValueStringRecoversFromPanickingError(t *testing.T) {
	got := SafeValueString(panickingError{})

	if !strings.Contains(got, "PANIC=Error method: boom") {
		t.Fatalf("SafeValueString() = %q, want fmt panic marker", got)
	}
	if got == "" {
		t.Fatalf("SafeValueString() returned an empty string")
	}
}

func TestSafeRecoveredValueStringAvoidsErrorMethod(t *testing.T) {
	got := SafeRecoveredValueString(panickingError{})

	if strings.Contains(got, "boom") {
		t.Fatalf("SafeRecoveredValueString() = %q, should not call Error()", got)
	}
	if !strings.Contains(got, "panickingError") {
		t.Fatalf("SafeRecoveredValueString() = %q, want type information", got)
	}
}

func TestSafeSprintPreservesPrefixWhenAnArgumentPanics(t *testing.T) {
	got := SafeSprint("panic: ", panickingError{})

	if !strings.HasPrefix(got, "panic: ") {
		t.Fatalf("SafeSprint() = %q, want prefix to be preserved", got)
	}
	if !strings.Contains(got, "PANIC=Error method: boom") {
		t.Fatalf("SafeSprint() = %q, want fmt panic marker", got)
	}
}

func TestMainLoggerInfoDoesNotPanicWhenWriterPanics(t *testing.T) {
	ml := &mainLogger{
		config: &LogConfig{
			Level:         DEBUG,
			ErrorWindow:   time.Minute,
			MaxErrorCount: 1,
		},
		mu:      &sync.RWMutex{},
		loggers: []*log.Logger{log.New(panickingWriter{}, "", 0)},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Info() panicked: %v", r)
		}
	}()

	ml.Info("hello")
}

func TestMainLoggerErrorDoesNotPanicWhenFormattingBrokenError(t *testing.T) {
	ml := &mainLogger{
		config: &LogConfig{
			Level:         DEBUG,
			ErrorWindow:   time.Minute,
			MaxErrorCount: 1,
		},
		mu:      &sync.RWMutex{},
		loggers: []*log.Logger{log.New(io.Discard, "", 0)},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Error() panicked: %v", r)
		}
	}()

	ml.Error("panic: ", panickingError{})
}
