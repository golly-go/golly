package golly

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// NewLogger returns a new logger intance
func NewLogger() *log.Entry {
	var formatter log.Formatter = &log.JSONFormatter{}
	level := LogLevel()

	if Env().IsDevelopmentOrTest() {
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
		"env":     Env(),
	})
}

func LogLevel() log.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	}

	return log.InfoLevel
}
