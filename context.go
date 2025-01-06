package golly

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"
)

type Context struct {

	// context.Context implementation
	parent   context.Context
	values   map[interface{}]interface{}
	deadline atomic.Value
	done     chan struct{}
	err      atomic.Value

	children unsafe.Pointer
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
		for _, c := range children {
			if c == child {
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
		for _, child := range children {
			child.cancel(err)
		}
	}
}

func (c *Context) addChild(child canceler) {
	for {
		oldPtr := atomic.LoadPointer(&c.children)

		var children []canceler
		if oldPtr != nil {
			children = *(*[]canceler)(oldPtr)
			for _, existingChild := range children {
				if existingChild == child {
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
		parent: parent,
		values: make(map[interface{}]interface{}),
		done:   make(chan struct{}),
	}
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

var _ context.Context = (*Context)(nil)
