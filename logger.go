package golly

import (
	log "github.com/sirupsen/logrus"
	"github.com/slimloans/golly/env"
)

func init() {
	if env.IsDevelopment() {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{})
		return
	}
	log.SetFormatter(&log.JSONFormatter{})
}

// NewLogger returns a new logger intance
func NewLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"service": appName,
		"version": Version(),
		"env":     env.CurrentENV(),
	})
}
