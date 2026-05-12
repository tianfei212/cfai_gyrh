package app

import (
	"testing"

	"gyrh-go-v2/backend/internal/logger"
)

func TestParseLogLevelDefaultsToInfo(t *testing.T) {
	if got := parseLogLevel("unknown"); got != logger.InfoLevel {
		t.Fatalf("expected info level, got %v", got)
	}
}
