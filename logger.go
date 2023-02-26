package golly

import (
	"os"

	"github.com/golly-go/golly/env"
	log "github.com/sirupsen/logrus"
)

// NewLogger returns a new logger intance
func NewLogger() *log.Entry {
	var formatter log.Formatter = &log.JSONFormatter{}
	level := log.InfoLevel

	if env.IsDevelopment() {
		level = log.DebugLevel
		formatter = &log.TextFormatter{}
	}

	l := &log.Logger{
		Out:          os.Stderr,
		Formatter:    formatter,
		Hooks:        make(log.LevelHooks),
		Level:        level,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}

	return l.WithFields(log.Fields{
		"service": appName,
		"version": Version(),
		"env":     env.CurrentENV(),
	})
}
