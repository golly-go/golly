package golly

import (
	"context"
	"slices"
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

// Without burning the stack, or doing fancy
// cycle detection right now that will take a some heavy
// thought to make it optimzed just catch 1000 == we are busted
const maxValueDepth = 1000

// Context is a custom context.Context implementation with Golly-specific features.
// It implements the standard context.Context interface while adding:
//   - Application reference for accessing the Golly app instance
//   - Logger with automatic inheritance from parent contexts
//   - DataLoader (cache) support
//   - Detachable contexts for async operations
//
// Context is safe for concurrent use and follows stdlib context patterns.
type Context struct {
	_ struct{} // prevent embedding

	application *Application

	loader atomic.Pointer[DataLoader]
	// logger removed (fields stored directly)

	// context.Context implementation (stdlib pattern)
	parent   context.Context
	key      interface{} // Single key-value pair (creates new context per WithValue)
	val      interface{}
	deadline atomic.Value
	done     atomic.Value
	err      atomic.Value

	fields []Field // Logger fields accumulated in this context

	children   unsafe.Pointer
	isDetached bool // If true, cuts off cancellation propagation
}

// Logger returns the logger entry for this context.
// It walks up the parent chain to find a cached logger, or creates a new default logger.
// The returned logger is cached in the current context for future calls.
//
// Maximum tree walk depth is limited to maxContextTreeWalk to prevent infinite loops.
// Logger returns the logger entry for this context.
// It reconstructs the logger by collecting fields from the context chain.
func (c *Context) Logger() *Entry {
	// Collect fields from parent chain
	fields := c.collectFields(nil)

	// Create new entry
	e := defaultLogger.newEntry()
	if len(fields) > 0 {
		e.fields = append(e.fields, fields...)
	}
	return e
}

func (c *Context) collectFields(acc []Field) []Field {
	var inf any = c

	// Pass 1: count how many fields we’ll append (bounded walk)
	needed := 0
	for range maxContextTreeWalk {
		if inf == nil {
			break
		}

		ctx, ok := inf.(*Context)
		if !ok {
			break // hit a stdlib context (or other), stop walking
		}

		needed += len(ctx.fields)
		inf = ctx.parent
	}

	if needed == 0 {
		return acc
	}

	// Ensure capacity once (Go 1.21+)
	if acc == nil {
		acc = make([]Field, 0, needed+4) // small headroom is optional
	} else {
		acc = slices.Grow(acc, needed)
	}

	// Pass 2: append fields in the same order as the walk (leaf -> root)
	inf = c
	for range maxContextTreeWalk {
		if inf == nil {
			break
		}

		ctx, ok := inf.(*Context)
		if !ok {
			break
		}

		if len(ctx.fields) > 0 {
			acc = append(acc, ctx.fields...)
		}

		inf = ctx.parent
	}

	return acc
}

// Cache returns the DataLoader (cache) for this context.
// It checks the immediate parent only, then creates a new DataLoader if not found.
// The returned DataLoader is cached in the current context for future calls.
//
// Note: Unlike Logger() and Application(), Cache() only checks one level up.
// This is because caches are typically request-scoped and don't chain deeply.
func (c *Context) Cache() *DataLoader {
	if loader := c.loader.Load(); loader != nil {
		return loader
	}

	// Only check immediate parent, then create (caches rarely chain deep)
	var loader *DataLoader
	switch p := c.parent.(type) {
	case *Context:
		if l := p.loader.Load(); l != nil {
			loader = l
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
walk:
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
		default:
			break walk
		}
	}

	// Fallback to global app
	if app != nil {
		c.application = app
	}

	return c.application
}

// SetApplication explicitly sets the application instance for this context.
// This is typically used during application initialization or testing.
func (c *Context) SetApplication(app *Application) {
	c.application = app
}

// canceler is an internal interface for contexts that can be canceled.
type canceler interface {
	cancel(err error)
}

// Deadline returns the deadline for this context, if one is set.
// It implements the context.Context interface.
// Returns the deadline time and true if set, or zero time and false otherwise.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if d := c.deadline.Load(); d != nil {
		return d.(time.Time), true
	}
	if c.parent != nil {
		return c.parent.Deadline()
	}
	return time.Time{}, false
}

// Done returns a channel that is closed when this context is canceled or times out.
// It implements the context.Context interface.
// Returns nil for detached contexts, which ignore parent cancellation.
func (c *Context) Done() <-chan struct{} {
	if c.isDetached {
		return nil
	}

	// Fast path
	if d := c.done.Load(); d != nil {
		return d.(chan struct{})
	}

	if c.parent != nil {
		switch p := c.parent.(type) {
		case *Context:
			if d, ok := p.done.Load().(chan struct{}); ok {
				return d
			}
		}
	}

	// Slow path: try to init
	newDone := make(chan struct{})
	if c.done.CompareAndSwap(nil, newDone) {
		return newDone
	}

	// Lost race, load winner
	return c.done.Load().(chan struct{})
}

// Err returns the error that caused this context to be canceled.
// It implements the context.Context interface.
// Returns nil if the context is not canceled, or if it's detached.
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

// Value returns the value associated with this context for key.
// It implements the context.Context interface.
//
// It walks up the parent chain iteratively to find the key-value pair.
// Maximum depth is 1000 to prevent infinite loops from circular references.
// This should be sufficient for any legitimate context chain.
//
// Returns nil if the key is not found or if max depth is reached.
func (c *Context) Value(key interface{}) interface{} {
	var inf context.Context = c

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
		default:
			return inf.Value(key)
		}
	}
	return nil
}

func (c *Context) cancel(err error) {
	// Ensure done channel exists so we can close it
	d := c.done.Load()
	if d == nil {
		newDone := make(chan struct{})
		if c.done.CompareAndSwap(nil, newDone) {
			d = newDone
		} else {
			d = c.done.Load()
		}
	}

	ch := d.(chan struct{})
	select {
	case <-ch:
		return // already closed
	default:
	}

	if !c.err.CompareAndSwap(nil, err) {
		return
	}

	close(ch)

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
// Detach creates a new context that preserves values from the parent chain
// but is independent from the parent's cancellation.
//
// This is useful for async operations that should continue even after
// the parent context is canceled (e.g., background processing, cleanup tasks).
//
// The detached context will still inherit values via Value(), but Done()
// and Err() will always return nil.
func (c *Context) Detach() *Context {
	return &Context{
		parent:      c, // Preserve for Value() lookups
		application: c.Application(),
		isDetached:  true, // Cut off cancellation
	}
}

// NewContext creates a new Golly context with the given parent.
// If parent is nil, it defaults to context.TODO().
//
// If the parent is a WebContext, it unwraps to the embedded Context
// to prevent circular references.
//
// The new context inherits:
//   - Application reference from Golly parents
//   - Cancellation propagation from the parent's Done() channel
//
// Returns a new *Context ready for use.
func NewContext(parent context.Context) *Context {
	if parent == nil {
		parent = context.TODO()
	}

	ctx := &Context{
		parent: parent,
		// done is lazy
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
		default:
			context.AfterFunc(parent, func() {
				ctx.cancel(parent.Err())
			})
		}
	}

	return ctx
}

// ToGollyContext converts a standard context.Context to a Golly *Context.
// If ctx is already a *Context, it returns it directly.
// If ctx is a *WebContext, it returns the embedded Context.
// Otherwise, it wraps the context in a new *Context via NewContext.
//
// This is useful for integrating with libraries that use standard contexts.
func ToGollyContext(ctx context.Context) *Context {
	switch c := ctx.(type) {
	case *Context:
		return c
	}

	return NewContext(ctx)
}

// WithValue returns a new context with the given key-value pair.
// It implements the stdlib context pattern where each call creates a new context.
//
// If parent is nil, it defaults to context.TODO().
// If parent is a WebContext, it unwraps to prevent cycles.
//
// The new context inherits the application reference from Golly parents,
// or uses the global app if available.
func WithValue(parent context.Context, key, val interface{}) *Context {
	if parent == nil {
		parent = context.TODO()
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
		case <-ctx.Done():
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

	// Convert map to []Field
	gctx.fields = make([]Field, 0, len(fields))
	for k, v := range fields {
		// Use helper to detect type? Or manual check?
		// We probably want a helper in logger to "makeField"
		// For now simple generic mapping
		gctx.fields = append(gctx.fields, Field{Key: k, Interface: v, Type: LogTypeAny})
	}

	// Check parent cache loader
	switch p := parent.(type) {
	case *Context:
		gctx.loader.Store(p.loader.Load())
	}

	return gctx
}

func WithLoggerField(parent context.Context, key string, value interface{}) *Context {
	gctx := NewContext(parent)

	gctx.fields = []Field{{Key: key, Interface: value, Type: LogTypeAny}}

	switch p := parent.(type) {
	case *Context:
		gctx.loader.Store(p.loader.Load())
	}

	return gctx
}

var _ context.Context = (*Context)(nil)
