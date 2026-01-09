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
	if c.done != nil {
		return c.done
	}
	if c.parent != nil {
		return c.parent.Done()
	}
	return nil
}

func (c *Context) Err() error {
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
// Note: Only copies values from golly.WithValue (not stdlib context.WithValue).
// For full compatibility, use golly.WithValue in your code.
//
// Example:
//
//	detached := ctx.Detach()
//	go processAsync(detached, event)  // Safe even after request finishes
func (c *Context) Detach() *Context {
	// Create new context with independent lifecycle
	newCtx := &Context{
		parent:      context.Background(), // No cancellation
		application: c.Application(),
	}

	// Copy values by walking up the Golly context chain
	var values []struct{ k, v interface{} }
	cur := c
	for cur != nil {
		if cur.key != nil {
			values = append(values, struct{ k, v interface{} }{cur.key, cur.val})
		}
		if parent, ok := cur.parent.(*Context); ok {
			cur = parent
		} else {
			break
		}
	}

	// Build new chain with copied values
	result := newCtx
	for i := len(values) - 1; i >= 0; i-- {
		result = WithValue(result, values[i].k, values[i].v)
	}

	return result
}

func NewContext(parent context.Context) *Context {
	if parent == nil {
		parent = context.TODO()
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
		if _, ok := parent.(*Context); !ok {
			context.AfterFunc(parent, func() {
				ctx.cancel(parent.Err())
			})
		}
	}

	return ctx
}

func ToGollyContext(ctx context.Context) *Context {
	if wc, ok := ctx.(*WebContext); ok {
		return wc.Context
	}

	if gc, ok := ctx.(*Context); ok {
		return gc
	}

	return NewContext(ctx)
}

// WithValue returns a new context with the given key-value pair.
// Each call creates a new context (stdlib pattern).
func WithValue(parent context.Context, key, val interface{}) *Context {
	if parent == nil {
		parent = context.TODO()
	}

	ctx := &Context{
		parent: parent,
		key:    key,
		val:    val,
	}

	// Inherit application
	if c, ok := parent.(*Context); ok && c != nil {
		ctx.application = c.application
	}

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
	ctx := ToGollyContext(parent)

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
	gctx := ToGollyContext(parent)

	gctx.logger.Store(gctx.Logger().WithFields(fields))

	if c, ok := parent.(*Context); ok && c != nil {
		gctx.loader = c.loader
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
