package golly

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type ContextKeyT string
type storeKeyT string

const (
	// LoggerKey key to the data map for the logger
	LoggerKey ContextKeyT = "logger"
	StoreKey  ContextKeyT = "store"
)

type Context struct {
	store *Store

	context context.Context
	config  *viper.Viper

	root *Route
}

// Set set a value on the context
func (c *Context) Set(key string, value interface{}) {
	c.store.Set(key, value)
}

// Get get a value from the context
func (c *Context) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

// NewContext returns a new application context provided some basic information
func NewContext(ctx context.Context) Context {
	return Context{
		context: ctx,
		store:   NewStore(),
	}
}

func (c *Context) Config() *viper.Viper {
	return c.config
}

func (c *Context) WithContext(ctx context.Context) context.Context {
	c.context = ctx
	return c.context
}

func (c *Context) UpdateLogFields(fields log.Fields) {
	c.store.Set(LoggerKey, c.Logger().WithFields(fields))
}

func (c *Context) SetLogger(l *log.Entry) {
	c.store.Set(LoggerKey, l)
}

func FromContext(ctx context.Context) Context {
	if c, ok := ctx.Value(StoreKey).(Context); ok {
		return c
	}
	return NewContext(ctx)
}

func (c Context) ToContext() context.Context {
	return context.WithValue(c.context, StoreKey, c)
}

func (c Context) Logger() *log.Entry {
	if lgr, found := c.store.Get(LoggerKey); found {
		if l, ok := lgr.(*log.Entry); ok {
			return l
		}
	}

	if lgr, ok := c.context.Value(LoggerKey).(*log.Entry); ok {
		return lgr
	}

	// Always make sure we return a log
	// this may be required for some applications
	return log.NewEntry(log.New())
}

// Context returns the context
func (c Context) Context() context.Context {
	return c.context
}
