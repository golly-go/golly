package golly

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Run("InheritParentFields", func(t *testing.T) {
		// Create a parent context and set its logger fields
		parentCtx := WithLoggerField(context.TODO(), "parent", "true")

		// Create a child context inheriting from the parent
		childCtx := NewContext(parentCtx)

		// Retrieve the logger from the child context
		logger := childCtx.Logger()

		// Assert that the parent's fields are present
		assert.NotNil(t, logger)
		fields := logger.Fields()
		assert.Equal(t, "true", fields["parent"])
	})

	t.Run("NewLoggerDefaults", func(t *testing.T) {
		// Create a standalone context with no parent
		standaloneCtx := NewContext(context.TODO())

		// Retrieve the logger from the context
		logger := standaloneCtx.Logger()

		// Assert that a new logger is created
		assert.NotNil(t, logger, "Expected a new logger to be created")
	})

	t.Run("CascadingLoggerUpwards", func(t *testing.T) {
		// Create a parent context and set its logger
		rootCtx := WithLoggerField(context.TODO(), "root", "true")

		// Create a chain of child contexts
		midCtx := NewContext(rootCtx)
		leafCtx := NewContext(midCtx)

		// Retrieve the logger from the deepest context
		logger := leafCtx.Logger()

		// Assert that the logger cascades upwards to the root (inherits fields)
		fields := logger.Fields()
		assert.Equal(t, "true", fields["root"], "Expected root=true in fields")
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

// ... existing tests ...

func BenchmarkContextLogger(b *testing.B) {
	// Setup: Create a parent context with fields
	parentCtx := WithLoggerField(context.TODO(), "foo", "bar")
	childCtx := NewContext(parentCtx)
	standaloneCtx := NewContext(context.TODO())

	// Benchmark: Logger construction from fields (Inherited)
	b.Run("InheritedFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = childCtx.Logger()
		}
	})

	// Benchmark: Logger construction (Empty)
	b.Run("Empty", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = standaloneCtx.Logger()
		}
	})
}

func TestContextCacheSharedWhenRootInitialized(t *testing.T) {
	root := NewContext(context.Background())
	rootLoader := root.Cache()

	childA := WithValue(root, "a", "1")
	childB := WithValue(root, "b", "2")

	loaderA := childA.Cache()
	loaderB := childB.Cache()

	assert.Same(t, rootLoader, loaderA)
	assert.Same(t, rootLoader, loaderB)
}

func TestContextCacheNotSharedAcrossSiblingsWithoutParentCache(t *testing.T) {
	root := NewContext(context.Background())
	childA := WithValue(root, "a", "1")
	childB := WithValue(root, "b", "2")

	loaderA := childA.Cache()
	loaderB := childB.Cache()
	loaderRoot := root.Cache()

	assert.NotSame(t, loaderA, loaderB)
	assert.NotSame(t, loaderA, loaderRoot)
	assert.NotSame(t, loaderB, loaderRoot)
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

// --- Cycle Prevention Tests ---

// TestContextLoggerNoCycle ensures Logger() doesn't infinite loop with circular references
func TestContextLoggerNoCycle(t *testing.T) {
	// Create a chain of contexts
	ctx1 := NewContext(context.Background())
	ctx2 := NewContext(ctx1)
	ctx3 := NewContext(ctx2)

	// Artificially create a cycle: ctx3 -> ctx1 (BAD in real code, but we should handle it)
	ctx3.parent = ctx1
	ctx1.parent = ctx3 // Creates cycle: ctx1 <-> ctx3

	// This should not infinite loop or stack overflow
	logger := ctx2.Logger()
	assert.NotNil(t, logger, "Should return a logger even with cycles")
}

// TestContextApplicationNoCycle ensures Application() doesn't infinite loop
func TestContextApplicationNoCycle(t *testing.T) {
	app := &Application{}

	ctx1 := NewContext(context.Background())
	ctx1.application = app
	ctx2 := NewContext(ctx1)
	ctx3 := NewContext(ctx2)

	// Create cycle
	ctx3.parent = ctx1
	ctx1.parent = ctx3

	// Should not infinite loop
	result := ctx2.Application()
	assert.Equal(t, app, result, "Should find application even with cycles")
}

// TestContextDeepChainLimit ensures we stop at maxDepth
func TestContextDeepChainLimit(t *testing.T) {
	// Create a chain deeper than maxDepth (10)
	root := NewContext(context.Background())
	current := root

	for i := 0; i < 15; i++ {
		current = NewContext(current)
	}

	// Should still work, just stop at maxDepth
	logger := current.Logger()
	assert.NotNil(t, logger, "Should return default logger when chain is too deep")
}

// TestContextValueNoCycle ensures Value() doesn't infinite loop with circular references
func TestContextValueNoCycle(t *testing.T) {
	ctx1 := WithValue(context.Background(), "key1", "value1")
	ctx2 := WithValue(ctx1, "key2", "value2")
	ctx3 := WithValue(ctx2, "key3", "value3")

	// Create cycle
	ctx3.parent = ctx1
	ctx1.parent = ctx3

	// Should not infinite loop
	val := ctx2.Value("key1")
	assert.Equal(t, "value1", val, "Should find value even with cycles")

	val = ctx2.Value("nonexistent")
	assert.Nil(t, val, "Should return nil for non-existent keys")
}

// --- Data Flow & Mock Injection Tests ---

type contextKey string

const (
	keyOpenAI contextKey = "openai"
	keyQdrant contextKey = "qdrant"
	keyStripe contextKey = "stripe"
	keyWorkOS contextKey = "workos"
	keyKMS    contextKey = "kms"
	keyEngine contextKey = "engine"
)

// Simulate user's setup functions using golly.WithValue
func withOpenAI(ctx context.Context, val string) context.Context {
	return WithValue(ctx, keyOpenAI, val)
}

func withQdrant(ctx context.Context, val string) context.Context {
	return WithValue(ctx, keyQdrant, val)
}

func withStripe(ctx context.Context, val string) context.Context {
	return WithValue(ctx, keyStripe, val)
}

func withWorkOS(ctx context.Context, val string) context.Context {
	return WithValue(ctx, keyWorkOS, val)
}

func withKMS(ctx context.Context, val string) context.Context {
	return WithValue(ctx, keyKMS, val)
}

func withEngine(ctx context.Context, val string) context.Context {
	return WithValue(ctx, keyEngine, val)
}

// TestDataFlow_ContextChaining replicates the user's SetupContext flow
func TestDataFlow_ContextChaining(t *testing.T) {
	baseCtx := context.Background()

	app, err := NewTestApplication(Options{Name: "data-flow-test"})
	require.NoError(t, err)

	ctx := withOpenAI(baseCtx, "mock-openai")
	ctx = withQdrant(ctx, "mock-qdrant")
	ctx = withStripe(ctx, "mock-stripe")
	ctx = withWorkOS(ctx, "mock-workos")
	ctx = withKMS(ctx, "mock-kms")

	gCtx := ToGollyContext(ctx)
	gCtx.SetApplication(app)
	ctx = gCtx

	ctx = withEngine(ctx, "mock-engine")

	assert.Equal(t, "mock-openai", ctx.Value(keyOpenAI))
	assert.Equal(t, "mock-qdrant", ctx.Value(keyQdrant))
	assert.Equal(t, "mock-stripe", ctx.Value(keyStripe))
	assert.Equal(t, "mock-workos", ctx.Value(keyWorkOS))
	assert.Equal(t, "mock-kms", ctx.Value(keyKMS))
	assert.Equal(t, "mock-engine", ctx.Value(keyEngine))

	finalGCtx := ToGollyContext(ctx)
	assert.Equal(t, app, finalGCtx.Application(), "Application should resolve via parent walk")
}

// TestDataFlow_MixedContextChain verifies Value() propagation through a stdlib context sandwich
// Golly -> Stdlib -> Golly
func TestDataFlow_MixedContextChain(t *testing.T) {
	rootCtx := NewContext(context.Background())
	rootCtx = WithValue(rootCtx, contextKey("golly-root"), "val-root")

	// Middle Stdlib Context (e.g. some middleware or library)
	// This breaks the *Context chain type-assertion, forcing fallback to .Value()
	middleCtx := context.WithValue(rootCtx, contextKey("stdlib-middle"), "val-middle")

	// Top Golly Context
	topCtx := WithValue(middleCtx, contextKey("golly-top"), "val-top")

	// Verification
	// Check immediate value
	assert.Equal(t, "val-top", topCtx.Value(contextKey("golly-top")))

	// Check parent value (traversing through stdlib context)
	assert.Equal(t, "val-middle", topCtx.Value(contextKey("stdlib-middle")))

	// Check root value (traversing through stdlib AND back into golly)
	assert.Equal(t, "val-root", topCtx.Value(contextKey("golly-root")))

	// Check Missing
	assert.Nil(t, topCtx.Value(contextKey("missing")))
}

// TestDataFlow_DetachedChain verifies that Detach() preserves values but cuts cancellation
func TestDataFlow_DetachedChain(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	rootCtx = WithValue(rootCtx, contextKey("val-root"), "root-data")

	// Create Detached Context in the middle
	// This simulates: ctx = golly.ToGollyContext(ctx).Detach()
	gCtx := ToGollyContext(rootCtx)
	detachedCtx := gCtx.Detach()

	// Add more values on top
	finalCtx := WithValue(detachedCtx, contextKey("val-top"), "top-data")

	// Verification

	// Values should still propagate (Detach preserves parent link for Value)
	assert.Equal(t, "top-data", finalCtx.Value(contextKey("val-top")))
	assert.Equal(t, "root-data", finalCtx.Value(contextKey("val-root")), "Detached context should still find parent values")

	// Cancellation should NOT propagate
	cancel() // Cancel root

	// Root should be canceled
	assert.Error(t, rootCtx.Err())

	// Detached/Final should NOT be canceled
	assert.NoError(t, finalCtx.Err(), "Detached context should not be canceled by parent")
	select {
	case <-finalCtx.Done():
		t.Fatal("Final context closed unexpectedly")
	default:
		// OK
	}
}
