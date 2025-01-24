package golly

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// Use this dont use it, upto you - this gives you a complex type
// for a context
type ContextKey string

type ContextFuncError func(*Context) error
type ContextFunc func(*Context)

type Context struct {
	application *Application

	loader atomic.Value // Stores *Dataloader
	logger atomic.Value // Stores *logrus.Entry

	// context.Context implementation
	parent   context.Context
	values   map[interface{}]interface{}
	deadline atomic.Value
	done     chan struct{}
	err      atomic.Value

	children unsafe.Pointer
}

func (c *Context) Logger() *logrus.Entry {
	// Fast path: Check for an existing logger in the current context
	if logger := c.logger.Load(); logger != nil {
		return logger.(*logrus.Entry)
	}

	var logger *logrus.Entry
	if parent, ok := c.parent.(*Context); ok && parent != nil {
		logger = parent.Logger()
	}

	if logger == nil {
		logger = logrus.NewEntry(Logger())
	}

	c.logger.Store(logger)
	return logger
}

func (c *Context) Cache() *DataLoader {
	if loader := c.loader.Load(); loader != nil {
		return loader.(*DataLoader)
	}

	var loader *DataLoader
	if parent, ok := c.parent.(*Context); ok && parent != nil {
		loader = parent.Cache()
	}

	if loader == nil {
		loader = NewDataLoader()
	}

	c.loader.Store(loader)
	return loader
}

// Application returns a link to the application
// buyer be wear this can be nil in test mode
func (c *Context) Application() *Application {
	if c.application != nil {
		return c.application
	}

	if parent, ok := c.parent.(*Context); ok && parent != nil {
		if parent.application != nil {
			c.application = parent.application
		}
	} else {
		c.application = app
	}

	return c.application
}

// Implementation of context.Context so we can be passed around
// regardless of library, this allows golly to be more portable
// while still providing its libaries a consistent Context
type canceler interface {
	cancel(err error)
}

// Deadline returns the deadline, if set, otherwise zero time
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if d := c.deadline.Load(); d != nil {
		return d.(time.Time), true
	}
	if c.parent != nil {
		return c.parent.Deadline()
	}
	return time.Time{}, false
}

// Done returns a channel that is closed when the context is canceled
func (c *Context) Done() <-chan struct{} {
	if c.parent != nil {
		return c.parent.Done()
	}
	return c.done
}

// Err returns the error explaining why the context was canceled, if any
func (c *Context) Err() error {
	if e := c.err.Load(); e != nil {
		return e.(error)
	}
	if c.parent != nil {
		return c.parent.Err()
	}
	return nil
}

func (c *Context) Value(key interface{}) interface{} {
	var inf context.Context = c

	for inf != nil {
		switch ctx := inf.(type) {
		case *Context:
			if val, exists := ctx.values[key]; exists {
				return val
			}
			inf = ctx.parent // Fix is here
		default:
			return inf.Value(key)
		}
	}
	return nil
}

func (c *Context) cancel(err error) {
	if !c.err.CompareAndSwap(nil, err) {
		return
	}

	close(c.done)

	if p, ok := c.parent.(*Context); ok {
		p.removeChild(c)
	}
	c.propagateCancel(err)
}

func (c *Context) removeChild(child canceler) {
	for {
		oldPtr := atomic.LoadPointer(&c.children)
		if oldPtr == nil {
			return
		}

		children := *(*[]canceler)(oldPtr)

		var newChildren []canceler
		for pos := range children {
			if children[pos] == child {
				continue
			}
			newChildren = append(newChildren, c)
		}

		if atomic.CompareAndSwapPointer(&c.children, oldPtr, unsafe.Pointer(&newChildren)) {
			return
		}
	}
}

func (c *Context) propagateCancel(err error) {
	if childrenPtr := atomic.LoadPointer(&c.children); childrenPtr != nil {
		children := *(*[]canceler)(childrenPtr)
		for pos := range children {
			children[pos].cancel(err)
		}
	}
}

func (c *Context) addChild(child canceler) {
	for {
		oldPtr := atomic.LoadPointer(&c.children)

		var children []canceler
		if oldPtr != nil {
			children = *(*[]canceler)(oldPtr)
			for pos := range children {
				if children[pos] == child {
					return
				}
			}
		}

		newChildren := append(children, child)
		if atomic.CompareAndSwapPointer(&c.children, oldPtr, unsafe.Pointer(&newChildren)) {
			return
		}
	}
}

func NewContext(parent context.Context) *Context {
	return &Context{
		parent:      parent,
		application: app,
		values:      make(map[interface{}]interface{}),
		done:        make(chan struct{}),
	}
}

func NewTestContext() *Context {
	return NewContext(context.TODO())
}

func WithValue(parent context.Context, key, val interface{}) *Context {
	ctx := NewContext(parent)
	ctx.values[key] = val

	return ctx
}

// WithCancel returns a copy of the parent context with a new cancel function
func WithCancel(parent context.Context) (*Context, context.CancelFunc) {
	ctx := NewContext(parent)

	if p, ok := parent.(*Context); ok {
		p.addChild(ctx)
	}

	cancel := func() {
		ctx.cancel(context.Canceled)
	}

	return ctx, cancel
}

// WithDeadline returns a context with a deadline
func WithDeadline(parent context.Context, d time.Time) (*Context, context.CancelFunc) {
	ctx := NewContext(parent)

	ctx.deadline.Store(d)

	if p, ok := parent.(*Context); ok {
		p.addChild(ctx)
	}

	cancel := func() {
		ctx.cancel(context.DeadlineExceeded)
	}

	go func() {
		select {
		case <-parent.Done():
			ctx.cancel(parent.Err())
		case <-time.After(time.Until(d)):
			cancel()
		case <-ctx.done:
		}
	}()
	return ctx, cancel
}

func WithApplication(parent context.Context, app *Application) *Context {
	gctx := NewContext(parent)
	gctx.application = app
	return gctx
}

func WithLoggerFields(parent context.Context, fields map[string]interface{}) *Context {
	gctx := NewContext(parent)

	gctx.logger.Store(gctx.Logger().WithFields(fields))

	if c, ok := parent.(*Context); ok && c != nil {
		gctx.loader = c.loader
	}

	return gctx
}

var _ context.Context = (*Context)(nil)
