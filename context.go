package golly

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"
)

// ContextKey is a type-safe key for context values.
// Use this to create strongly-typed context keys and avoid collisions.
type ContextKey string

// ContextFuncError is a function that operates on a Context and may return an error.
type ContextFuncError func(*Context) error

// ContextFunc is a function that operates on a Context.
type ContextFunc func(*Context)

// maxContextTreeWalk is the maximum depth when walking up the context parent chain
// for Logger() and Application(). Prevents infinite loops from circular references.
const maxContextTreeWalk = 12

// Context is a custom context.Context implementation with Golly-specific features.
// It implements the standard context.Context interface while adding:
//   - Application reference for accessing the Golly app instance
//   - Logger with automatic inheritance from parent contexts
//   - DataLoader (cache) support
//   - Detachable contexts for async operations
//
// Context is safe for concurrent use and follows stdlib context patterns.
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

// Logger returns the logger entry for this context.
// It walks up the parent chain to find a cached logger, or creates a new default logger.
// The returned logger is cached in the current context for future calls.
//
// Maximum tree walk depth is limited to maxContextTreeWalk to prevent infinite loops.
func (c *Context) Logger() *Entry {
	// Fast path: Check for an existing logger in the current context
	if logger, ok := c.logger.Load().(*Entry); ok && logger != nil {
		return logger
	}

	// Walk up parent chain iteratively (stdlib pattern - prevents stack overflow)
	var current context.Context = c
	for range maxContextTreeWalk {
		if current == nil {
			break
		}
		switch p := current.(type) {
		case *Context:
			if logger, ok := p.logger.Load().(*Entry); ok && logger != nil {
				c.logger.Store(logger)
				return logger
			}
			current = p.parent
		case *WebContext:
			if logger, ok := p.Context.logger.Load().(*Entry); ok && logger != nil {
				c.logger.Store(logger)
				return logger
			}
			current = p.Context.parent
		default:
			break
		}
	}

	// No logger found, create default
	logger := defaultLogger.newEntry()
	c.logger.Store(logger)
	return logger
}

// Cache returns the DataLoader (cache) for this context.
// It checks the immediate parent only, then creates a new DataLoader if not found.
// The returned DataLoader is cached in the current context for future calls.
//
// Note: Unlike Logger() and Application(), Cache() only checks one level up.
// This is because caches are typically request-scoped and don't chain deeply.
func (c *Context) Cache() *DataLoader {
	if loader := c.loader.Load(); loader != nil {
		return loader.(*DataLoader)
	}

	// Only check immediate parent, then create (caches rarely chain deep)
	var loader *DataLoader
	switch p := c.parent.(type) {
	case *Context:
		if l := p.loader.Load(); l != nil {
			loader = l.(*DataLoader)
		}
	case *WebContext:
		if l := p.Context.loader.Load(); l != nil {
			loader = l.(*DataLoader)
		}
	}

	if loader == nil {
		loader = NewDataLoader()
	}

	c.loader.Store(loader)
	return loader
}

// Application returns a link to the Golly application instance.
// It walks up the parent chain to find an application reference, or falls back to the global app.
// The returned application is cached in the current context for future calls.
//
// Maximum tree walk depth is limited to maxContextTreeWalk to prevent infinite loops.
// Note: This can return nil in test mode or when no application is configured.
func (c *Context) Application() *Application {
	if c.application != nil {
		return c.application
	}

	// Walk up parent chain iteratively to find application
	var current context.Context = c.parent
	for range maxContextTreeWalk {
		if current == nil {
			break
		}
		switch p := current.(type) {
		case *Context:
			if p.application != nil {
				c.application = p.application
				return c.application
			}
			current = p.parent
		case *WebContext:
			if p.Context.application != nil {
				c.application = p.Context.application
				return c.application
			}
			current = p.Context.parent
		default:
			break
		}
	}

	// Fallback to global app
	if app != nil {
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
	// Without bruning the stack, or doing fancy
	// cycle detection right now that will take a some heavy
	// thought to make it optimzed just catch 1000 == we are busted
	const maxValueDepth = 1000

	for range maxValueDepth {
		if inf == nil {
			break
		}

		switch ctx := inf.(type) {
		case *Context:
			if ctx.key == key {
				return ctx.val
			}
			inf = ctx.parent
		case *WebContext:
			if ctx.key == key {
				return ctx.val
			}
			inf = ctx.parent
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
