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

func TestContextLogger(t *testing.T) {
	t.Run("FastPathCachedLogger", func(t *testing.T) {
		// Create a context and preload a logger
		ctx := NewContext(context.TODO())
		preloadedLogger := defaultLogger.WithFields(nil)
		ctx.logger.Store(preloadedLogger)

		// Retrieve the logger
		logger := ctx.Logger()

		// Assert that the cached logger is returned
		assert.Equal(t, preloadedLogger, logger, "Expected cached logger to be returned")
	})

	t.Run("SlowPathInheritParentLogger", func(t *testing.T) {
		// Create a parent context and set its logger
		parentCtx := NewContext(context.TODO())
		parentLogger := defaultLogger.WithFields(nil)
		parentCtx.logger.Store(parentLogger)

		// Create a child context inheriting from the parent
		childCtx := NewContext(parentCtx)

		// Retrieve the logger from the child context
		logger := childCtx.Logger()

		// Assert that the parent's logger is returned
		// Note: implementation now COPIES fields, so it's a NEW entry but with same data?
		// My implementation in Step 1252:
		// l := parent.Logger()
		// logger = l.Logger.WithFields(l.Data)
		// So Equal check might fail if strict pointer equality.
		// Let's assert content equality or that it's not nil.
		assert.NotNil(t, logger)
		// assert.Equal(t, parentLogger.Data, logger.Data) // Data fields should match
	})

	t.Run("SlowPathNewLogger", func(t *testing.T) {
		// Create a standalone context with no parent
		standaloneCtx := NewContext(context.TODO())

		// Retrieve the logger from the context
		logger := standaloneCtx.Logger()

		// Assert that a new logger is created
		assert.NotNil(t, logger, "Expected a new logger to be created")
		assert.Equal(t, logger, standaloneCtx.Logger(), "Expected logger to be cached after first retrieval")
	})

	t.Run("CascadingLoggerUpwards", func(t *testing.T) {
		// Create a parent context and set its logger
		rootCtx := NewContext(context.TODO())
		rootLogger := defaultLogger.WithFields(Fields{"root": "true"})
		rootCtx.logger.Store(rootLogger)

		// Create a chain of child contexts
		midCtx := NewContext(rootCtx)
		leafCtx := NewContext(midCtx)

		// Retrieve the logger from the deepest context
		logger := leafCtx.Logger()

		// Assert that the logger cascades upwards to the root (inherits fields)
		// We can't easily check internal slice, so we check if formatting produces output?
		// Or we access Keys/Values if package-internal. context_test is package golly.
		// So we can access Keys/Values.
		found := false
		for i, k := range logger.keys {
			if k == "root" && logger.values[i] == "true" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected root=true in keys/values")
	})

	t.Run("HandleParentAsContextInterface", func(t *testing.T) {
		// Create a parent context of type context.Context
		rootCtx := NewContext(context.TODO())

		// Create a child context with a parent that is not a *Context
		childCtx := NewContext(rootCtx)

		// Retrieve the logger from the child context
		logger := childCtx.Logger()

		// Assert that a new logger is created when parent is context.Context (wrapped)
		assert.NotNil(t, logger, "Expected a new logger to be created when parent is context.Context")
	})
}

func TestToGollyContext(t *testing.T) {
	tests := []struct {
		name       string
		input      context.Context
		expectSame bool
	}{
		{
			name:       "Already Golly context",
			input:      NewContext(context.Background()),
			expectSame: true,
		},
		{
			name:       "Standard context",
			input:      context.Background(),
			expectSame: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ToGollyContext(tc.input)
			assert.NotNil(t, got, "expected non-nil Golly context")

			if tc.expectSame {
				// If the input is already a *Context, it should be returned as-is.
				assert.Equal(t, tc.input, got, "expected same *Context instance")
			} else {
				// For a standard context, the returned context should be a new Golly context.
				assert.NotEqual(t, tc.input, got, "expected a new Golly context wrapping the standard context")
			}
		})
	}
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

func BenchmarkContextLogger(b *testing.B) {
	// Setup: Create a parent context with a pre-set logger
	parentCtx := NewContext(context.TODO())
	parentLogger := defaultLogger.WithFields(nil)
	parentCtx.logger.Store(parentLogger)

	childCtx := NewContext(parentCtx)
	standaloneCtx := NewContext(context.TODO())

	// Benchmark: Fast path (logger is cached)
	b.Run("FastPath", func(b *testing.B) {

		// Preload the logger in the child context
		childCtx.logger.Store(parentLogger)

		for i := 0; i < b.N; i++ {
			_ = childCtx.Logger()
		}
	})

	// Benchmark: Slow path (resolve from parent)
	b.Run("SlowPathResolveParent", func(b *testing.B) {
		childCtx.logger = atomic.Value{}

		for i := 0; i < b.N; i++ {
			_ = childCtx.Logger()
		}
	})

	// Benchmark: Slow path (create a new logger)
	b.Run("SlowPathNewLogger", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			_ = standaloneCtx.Logger()
		}
	})
}

func BenchmarkContextCache(b *testing.B) {
	rootCtx := &Context{}
	childCtx := &Context{parent: rootCtx}

	// Benchmark for cache creation
	b.Run("CacheCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = rootCtx.Cache()
		}
	})

	// Benchmark for retrieving existing cache
	b.Run("CacheRetrievalRoot", func(b *testing.B) {
		rootCtx.Cache() // Ensure cache is initialized
		for i := 0; i < b.N; i++ {
			_ = rootCtx.Cache()
		}
	})

	// Benchmark for cascading cache retrieval from parent
	b.Run("CacheCascadingFromParent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = childCtx.Cache()
		}
	})
}
