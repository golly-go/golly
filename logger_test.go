package golly

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		logEnvValue   string
		expectedLevel log.Level
	}{
		{"default", "", log.InfoLevel},
		{"debug", "debug", log.DebugLevel},
		{"info", "info", log.InfoLevel},
		{"warn", "warn", log.WarnLevel},
		{"error", "error", log.ErrorLevel},
		{"fatal", "fatal", log.FatalLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv("LOG_LEVEL", tt.logEnvValue)

			// Action
			got := LogLevel()

			// Assert
			if got != tt.expectedLevel {
				t.Errorf("LogLevel() = %v; want %v", got, tt.expectedLevel)
			}
		})
	}
}
