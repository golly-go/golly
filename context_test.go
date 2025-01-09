package golly

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContextChildGrowth(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	var wg sync.WaitGroup
	gctx := NewContext(parent)

	var initialChildrenCount int
	childrenPtr := atomic.LoadPointer(&gctx.children)
	if childrenPtr != nil {
		initialChildrenCount = len(*(*[]canceler)(childrenPtr))
	}

	// Simulate spawning multiple child contexts
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			WithValue(gctx, "key", "val")
			time.Sleep(10 * time.Millisecond) // Simulate work
		}()
	}

	wg.Wait()

	var finalChildrenCount int
	childrenPtr = atomic.LoadPointer(&gctx.children)
	if childrenPtr != nil {
		finalChildrenCount = len(*(*[]canceler)(childrenPtr))
	}

	assert.LessOrEqual(t, finalChildrenCount, initialChildrenCount, "Child contexts should be cleaned up")
}

func TestDeadline(t *testing.T) {
	tests := []struct {
		name           string
		setDeadline    bool
		parentDeadline time.Duration
		childDeadline  time.Duration
		expectTimeout  bool
	}{
		{
			name:           "Child deadline should apply",
			setDeadline:    true,
			parentDeadline: 200 * time.Millisecond,
			childDeadline:  100 * time.Millisecond,
			expectTimeout:  true,
		},
		{
			name:           "Parent deadline should propagate",
			setDeadline:    false,
			parentDeadline: 100 * time.Millisecond,
			childDeadline:  200 * time.Millisecond,
			expectTimeout:  true,
		},
		{
			name:           "No deadline should return false",
			setDeadline:    false,
			parentDeadline: 0,
			childDeadline:  0,
			expectTimeout:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var parent context.Context
			if tt.parentDeadline > 0 {
				parent, _ = WithDeadline(context.Background(), time.Now().Add(tt.parentDeadline))
			} else {
				parent = context.Background()
			}

			var ctx context.Context
			if tt.setDeadline {
				ctx, _ = WithDeadline(parent, time.Now().Add(tt.childDeadline))
			} else {
				ctx = parent
			}

			deadline, ok := ctx.Deadline()
			if tt.expectTimeout {
				assert.True(t, ok)
				assert.NotZero(t, deadline)
			} else {
				assert.False(t, ok)
			}
		})
	}
}

func TestWithValue(t *testing.T) {
	tests := []struct {
		name      string
		key       interface{}
		value     interface{}
		parentKey interface{}
		parentVal interface{}
		expectVal interface{}
	}{
		{
			name:      "Value should propagate",
			key:       "key",
			value:     "childValue",
			parentKey: "key",
			parentVal: "parentValue",
			expectVal: "childValue",
		},
		{
			name:      "Parent value should be inherited",
			key:       "key",
			value:     nil,
			parentKey: "key",
			parentVal: "parentValue",
			expectVal: "parentValue",
		},
		{
			name:      "Value should be nil if not set",
			key:       "missingKey",
			value:     nil,
			parentKey: "key",
			parentVal: "value",
			expectVal: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := WithValue(context.Background(), tt.parentKey, tt.parentVal)

			var ctx context.Context
			if tt.value != nil {
				ctx = WithValue(parent, tt.key, tt.value)
			} else {
				ctx = parent
			}

			assert.Equal(t, tt.expectVal, ctx.Value(tt.key))
		})
	}
}

func TestWithCancel(t *testing.T) {
	tests := []struct {
		name      string
		cancelNow bool
		expectErr error
	}{
		{
			name:      "Cancel should propagate",
			cancelNow: true,
			expectErr: context.Canceled,
		},
		{
			name:      "Context should remain active",
			cancelNow: false,
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := WithCancel(context.Background())

			if tt.cancelNow {
				cancel()
			}

			assert.Equal(t, tt.expectErr, ctx.Err())
		})
	}
}

func TestWithDeadline(t *testing.T) {
	tests := []struct {
		name      string
		deadline  time.Duration
		wait      time.Duration
		expectErr error
	}{
		{
			name:      "Context should timeout",
			deadline:  50 * time.Millisecond,
			wait:      100 * time.Millisecond,
			expectErr: context.DeadlineExceeded,
		},
		{
			name:      "Context should not timeout",
			deadline:  100 * time.Millisecond,
			wait:      50 * time.Millisecond,
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := WithDeadline(context.Background(), time.Now().Add(tt.deadline))
			defer cancel()

			time.Sleep(tt.wait)
			assert.Equal(t, tt.expectErr, ctx.Err())
		})
	}
}

func TestRemoveChild(t *testing.T) {
	parent, _ := WithCancel(context.Background())
	child, _ := WithCancel(parent)

	parent.addChild(child)
	assert.Len(t, *(*[]canceler)(atomic.LoadPointer(&parent.children)), 1)

	parent.removeChild(child)
	assert.Len(t, *(*[]canceler)(atomic.LoadPointer(&parent.children)), 0)
}

func TestPropagateCancel(t *testing.T) {
	parent, cancelParent := WithCancel(context.Background())
	child, _ := WithCancel(parent)

	parent.addChild(child)

	cancelParent()

	assert.Equal(t, context.Canceled, child.Err())
}

// ***************************************************************************
// *  Benches
// ***************************************************************************

// Benchmark for context value retrieval.
func BenchmarkContextValue(b *testing.B) {
	ctx := WithValue(context.Background(), "key", "value")

	b.Run("BenchmarkContextValue", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctx.Value("key")
		}
	})
}

// Benchmark for context cancellation (parent cancel).
func BenchmarkContextCancel(b *testing.B) {
	parent, cancel := WithCancel(context.Background())
	defer cancel()
	_, childCancel := WithCancel(parent)
	defer childCancel()

	b.Run("BenchmarkContextCancel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cancel()
		}
	})
}

// Benchmark for deadline propagation.
func BenchmarkContextWithDeadline(b *testing.B) {
	deadline := time.Now().Add(10 * time.Millisecond)
	parent := context.Background()

	b.Run("BenchmarkContextWithDeadline", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, cancel := WithDeadline(parent, deadline)
			cancel()
		}
	})
}

// Benchmark for context value propagation.
func BenchmarkContextWithValuePropagation(b *testing.B) {
	parent := context.Background()

	b.Run("BenchmarkContextWithValuePropagation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = WithValue(parent, "key", i)
		}
	})
}
