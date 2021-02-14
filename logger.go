package golly

import (
	log "github.com/sirupsen/logrus"
	"github.com/slimloans/golly/env"
)

func init() {
	if !env.IsDevelopment() {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{})
	}
}

// NewLogger returns a new logger intance
func NewLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"service": appName,
		"version": Version(),
		"env":     env.CurrentENV(),
	})
}
