package golly

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	silenced bool = false
)

// NewLogger returns a new logger intance
func NewLogger() *log.Logger {
	var formatter log.Formatter = &log.JSONFormatter{}

	level := LogLevel()
	if Env().IsDevelopmentOrTest() {
		formatter = &log.TextFormatter{}
	}

	return &log.Logger{
		Out:          os.Stderr,
		Formatter:    formatter,
		Hooks:        make(log.LevelHooks),
		Level:        level,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
}

func LogLevel() log.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "trace":
		return log.TraceLevel
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

func Logger() *log.Logger {
	if app != nil {
		return app.logger
	}
	// this happens in test
	return NewLogger()
}

func Tracef(format string, args ...interface{}) {
	if silenced {
		return
	}

	if app == nil || app.logger == nil {
		return
	}

	if app.logger.IsLevelEnabled(log.TraceLevel) {
		app.logger.Tracef(format, args...)
	}
}

func TraceWithContext(ctx *Context, format string, args ...interface{}) {
	if silenced {
		return
	}

	if app == nil || app.logger == nil {
		return
	}

	if app.logger.IsLevelEnabled(log.TraceLevel) {
		ctx.Logger().Tracef(format, args...)
	}
}
