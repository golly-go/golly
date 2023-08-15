package golly

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type ContextKeyT string

const (
	// LoggerKey key to the data map for the logger
	LoggerKey ContextKeyT = "logger"
	StoreKey  ContextKeyT = "store"
)

type Context struct {
	data *sync.Map

	cancel context.CancelFunc

	context context.Context
	config  *viper.Viper

	runmode string

	root *Route
}

func (c *Context) RunMode() string {
	return c.runmode
}

// TODO Implement
func (*Context) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*Context) Done() <-chan struct{}       { return nil }
func (*Context) Err() error                  { return nil }

func (c *Context) Value(key interface{}) interface{} {
	return c.context.Value(key)
}

// Set set a value on the context
func (c *Context) Set(key interface{}, value interface{}) Context {
	c.data.Store(key, value)
	return *c
}

// Get get a value from the context
func (c *Context) Get(key interface{}) (interface{}, bool) {
	return c.data.Load(key)
}

// NewContext returns a new application context provided some basic information
func NewContext(ctx context.Context) Context {
	c, cancel := context.WithCancel(ctx)

	return Context{
		context: c,
		cancel:  cancel,
		data:    &sync.Map{},
	}
}

func (c *Context) Config() *viper.Viper {
	return c.config
}

func (a Application) NewContext(parent context.Context) Context {
	ctx := NewContext(parent)
	ctx.root = a.routes
	ctx.config = a.Config

	ctx.SetLogger(a.Logger)

	return ctx
}

func (c *Context) WithContext(ctx context.Context) context.Context {
	c.context = ctx
	return c.context
}

func (c *Context) UpdateLogFields(fields log.Fields) {
	c.data.Store(LoggerKey, c.Logger().WithFields(fields))
}

func (c *Context) SetLogger(l *log.Entry) {
	c.data.Store(LoggerKey, l)
}

func (c *Context) Dup() Context {
	return *c
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
	if c.data != nil {
		if lgr, found := c.data.Load(LoggerKey); found {
			if l, ok := lgr.(*log.Entry); ok {
				return l
			}
		}
	}

	if c.context != nil {
		if lgr, ok := c.context.Value(LoggerKey).(*log.Entry); ok {
			return lgr
		}
	}

	// Always make sure we return a log
	// this may be required for some applications
	return NewLogger()
}

// Context returns the context
func (c Context) Context() context.Context {
	return c.context
}
