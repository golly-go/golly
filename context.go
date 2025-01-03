package golly

import (
	"context"
	"sync"
	"sync/atomic"
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

// Context represents an application-specific context with custom cancellation and data handling.
type Context struct {
	loader *DataLoader
	data   *sync.Map
	config *viper.Viper
	env    EnvName

	route *Route

	done     chan struct{}
	err      atomic.Value // Atomic error storage
	deadline atomic.Value // Atomic time.Time storag
}

// Deadline returns the time when this context will be canceled, if any.
func (c Context) Deadline() (time.Time, bool) {
	v := c.deadline.Load()
	if v == nil {
		return time.Time{}, false
	}
	return v.(time.Time), true
}

// Done returns a channel that is closed when this context is canceled.
func (c Context) Done() <-chan struct{} {
	return c.done
}

// Err returns the error associated with this context, if any.
func (c Context) Err() error {
	v := c.err.Load()
	if v == nil {
		return nil
	}
	return v.(error)
}

// Value returns the value associated with this context for the given key.
func (c Context) Value(key interface{}) interface{} {
	if val, ok := c.data.Load(key); ok {
		return val
	}
	return nil
}

// Cancel cancels the context, closing the done channel and setting the error.
func (c Context) Cancel(err error) {
	select {
	case <-c.done:
		// Already closed
	default:
		close(c.done)
		c.err.Store(err)
	}
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

func (c Context) Env() EnvName {
	return c.env
}

// NewContext returns a new application context provided some basic information
func NewContext(ctx context.Context) Context {
	return Context{
		loader: NewDataLoader(),
		// We probably want to deprecate this
		// as both it and the dataloader are not necessary

		data: &sync.Map{},
	}
}

func (c *Context) Config() *viper.Viper {
	return c.config
}

func (c *Context) Loader() *DataLoader {
	return c.loader
}

func (c *Context) UpdateLogFields(fields log.Fields) {
	c.data.Store(LoggerKey, c.Logger().WithFields(fields))
}

func (c *Context) SetLogger(l *log.Entry) {
	c.data.Store(LoggerKey, l)
}

func (c *Context) Dup() Context {
	return *c // shallow copy
}

func FromContext(ctx context.Context) Context {
	if c, ok := ctx.Value(StoreKey).(Context); ok {
		return c
	}
	return NewContext(ctx)
}

func (c Context) Logger() *log.Entry {
	if c.data != nil {
		if lgr, found := c.data.Load(LoggerKey); found {
			if l, ok := lgr.(*log.Entry); ok {
				return l
			}
		}
	}

	// Always make sure we return a log
	// this may be required for some applications
	return NewLogger()
}

func (a Application) NewContext(parent context.Context) Context {
	ctx := NewContext(parent)

	ctx.env = a.Env
	ctx.route = a.routes
	ctx.config = a.Config

	ctx.SetLogger(a.Logger)

	return ctx
}

var _ context.Context = Context{}
