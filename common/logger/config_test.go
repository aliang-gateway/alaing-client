package logger

import (
	"testing"
)

func TestDefaultLogConfigUsesInfoInProd(t *testing.T) {
	t.Setenv("ALIANG_BUILD_MODE", "prod")

	cfg := DefaultLogConfig()
	if cfg.Level != INFO {
		t.Fatalf("DefaultLogConfig().Level = %v, want %v", cfg.Level, INFO)
	}

	httpCfg := HTTPLogConfig()
	if httpCfg.Level != INFO {
		t.Fatalf("HTTPLogConfig().Level = %v, want %v", httpCfg.Level, INFO)
	}
}

func TestDefaultLogConfigKeepsDevDefaults(t *testing.T) {
	t.Setenv("ALIANG_BUILD_MODE", "dev")

	cfg := DefaultLogConfig()
	if cfg.Level != DEBUG {
		t.Fatalf("DefaultLogConfig().Level = %v, want %v", cfg.Level, DEBUG)
	}

	httpCfg := HTTPLogConfig()
	if httpCfg.Level != TRACE {
		t.Fatalf("HTTPLogConfig().Level = %v, want %v", httpCfg.Level, TRACE)
	}
}
