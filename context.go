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

	// context.Context implementation (stdlib pattern)
	parent   context.Context
	key      interface{} // Single key-value pair (creates new context per WithValue)
	val      interface{}
	deadline atomic.Value
	done     chan struct{}
	err      atomic.Value

	children   unsafe.Pointer
	isDetached bool // If true, cuts off cancellation propagation
}

func (c *Context) Logger() *logrus.Logger {
	// Fast path: Check for an existing logger in the current context
	if logger, ok := c.logger.Load().(*logrus.Logger); ok && logger != nil {
		return logger
	}

	var logger *logrus.Logger
	if logger == nil {
		logger = logrus.New()
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
	} else if app != nil {
		c.application = app
	}

	return c.application
}

func (c *Context) SetApplication(app *Application) {
	c.application = app
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

func (c *Context) Done() <-chan struct{} {
	if c.isDetached {
		return nil
	}
	if c.done != nil {
		return c.done
	}
	if c.parent != nil {
		return c.parent.Done()
	}
	return nil
}

func (c *Context) Err() error {
	if c.isDetached {
		return nil
	}
	if e := c.err.Load(); e != nil {
		return e.(error)
	}
	// if parent canceled first and this context hasn't, you should reflect that
	if c.parent != nil {
		if err := c.parent.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) Value(key interface{}) interface{} {
	var inf context.Context = c

	for inf != nil {
		switch ctx := inf.(type) {
		case *Context:
			// Check single key
			if ctx.key == key {
				return ctx.val
			}
			inf = ctx.parent // Walk up chain
		case WebContext:
			// do nothing
			if ctx.key == key {
				return ctx.val
			}
			inf = ctx.parent // Walk up chain
		case *WebContext:
			// do nothing
			if ctx.key == key {
				return ctx.val
			}
			inf = ctx.parent // Walk up chain
		default:
			return inf.Value(key)
		}
	}
	return nil
}

func (c *Context) cancel(err error) {
	if c.done == nil {
		return // Not a cancellable context
	}
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
			newChildren = append(newChildren, children[pos])
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

// Detach creates a new context that preserves all values but has independent lifecycle.
// Useful for async operations that need identity/tenant info but outlive the request.
//
// The detached context:
// - Preserves all values (from both golly.WithValue and stdlib context.WithValue)
// - Won't be cancelled when the original context is cancelled
// - Has no deadline/timeout
// - Keeps the same Application reference
//
// Example:
//
//	detached := ctx.Detach()
//	go processAsync(detached, event)  // Safe even after request finishes
func (c *Context) Detach() *Context {
	return &Context{
		parent:      c, // Preserve for Value() lookups
		application: c.Application(),
		isDetached:  true, // Cut off cancellation
	}
}

func NewContext(parent context.Context) *Context {
	if parent == nil {
		parent = context.TODO()
	}

	// Unroll WebContext to prevent cycles
	if wc, ok := parent.(*WebContext); ok {
		parent = wc.Context
	}

	ctx := &Context{
		parent: parent,
		done:   make(chan struct{}),
	}

	// Inherit application
	if c, ok := parent.(*Context); ok && c != nil {
		ctx.application = c.application
	} else if app != nil {
		ctx.application = app
	}

	// Propagate cancellation from non-Golly parent contexts
	if parent.Done() != nil {
		switch parent.(type) {
		case *Context:
			// do nothing
		case WebContext:
			// do nothing
		case *WebContext:
			// do nothing
		default:
			context.AfterFunc(parent, func() {
				ctx.cancel(parent.Err())
			})
		}
	}

	return ctx
}

func ToGollyContext(ctx context.Context) *Context {
	switch c := ctx.(type) {
	case *Context:
		return c
	case WebContext:
		return c.Context
	case *WebContext:
		return c.Context
	}

	return NewContext(ctx)
}

// WithValue returns a new context with the given key-value pair.
// Each call creates a new context (stdlib pattern).
func WithValue(parent context.Context, key, val interface{}) *Context {
	if parent == nil {
		parent = context.TODO()
	}

	// Unroll WebContext to prevent cycles
	if wc, ok := parent.(*WebContext); ok {
		parent = wc.Context
	}

	ctx := &Context{
		parent: parent,
		key:    key,
		val:    val,
	}

	// inherit application
	switch p := parent.(type) {
	case *Context:
		ctx.application = p.application
	case WebContext:
		ctx.application = p.application
	case *WebContext:
		ctx.application = p.application
	default:
		ctx.application = app
	}

	return ctx
}

// WithCancel returns a copy of the parent context with a new cancel function
func WithCancel(parent context.Context) (*Context, context.CancelFunc) {
	ctx := NewContext(parent)

	switch p := parent.(type) {
	case *Context:
		p.addChild(ctx)
	case WebContext:
		p.addChild(ctx)
	case *WebContext:
		p.addChild(ctx)
	}

	cancel := func() {
		ctx.cancel(context.Canceled)
	}

	return ctx, cancel
}

// WithDeadline returns a context with a deadline
func WithDeadline(parent context.Context, d time.Time) (*Context, context.CancelFunc) {
	ctx := ToGollyContext(parent)

	ctx.deadline.Store(d)

	switch p := parent.(type) {
	case *Context:
		p.addChild(ctx)
	case WebContext:
		p.addChild(ctx)
	case *WebContext:
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
	gctx := ToGollyContext(parent)

	gctx.logger.Store(gctx.Logger().WithFields(fields))

	switch p := parent.(type) {
	case *Context:
		gctx.loader = p.loader
	case *WebContext:
		gctx.loader = p.Context.loader
	case WebContext:
		gctx.loader = p.loader
	}

	return gctx
}

// WithLoggerField adds a single field to the logger (lightweight, avoids map allocation).
func WithLoggerField(parent context.Context, key string, value interface{}) *Context {
	gctx := ToGollyContext(parent)

	gctx.logger.Store(gctx.Logger().WithField(key, value))

	return gctx
}

var _ context.Context = (*Context)(nil)
