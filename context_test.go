package golly

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextChildGrowth(t *testing.T) {
	parent := t.Context()

	var wg sync.WaitGroup
	gctx := NewContext(parent)

	var initialChildrenCount int
	childrenPtr := atomic.LoadPointer(&gctx.children)
	if childrenPtr != nil {
		initialChildrenCount = len(*(*[]canceler)(childrenPtr))
	}

	// Simulate spawning multiple child contexts
	for range 100 {
		wg.Go(func() {

			WithValue(gctx, "key", "val")
			time.Sleep(10 * time.Millisecond) // Simulate work
		})
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
		key       any
		value     any
		parentKey any
		parentVal any
		expectVal any
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

	for range 15 {
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

func TestContext_LoggerLeak(t *testing.T) {
	// Re-route default logger for the test
	buf := new(bytes.Buffer)
	oldLogger := defaultLogger

	defaultLogger = NewLogger()
	defaultLogger.SetOutput(buf)
	defaultLogger.SetFormatter(&TextFormatter{DisableColors: true})

	defer func() {
		defaultLogger = oldLogger
	}()

	// Step 1: Create a context with many fields to expand the pooled entry's fields slice
	ctx1 := WithLoggerFields(NewContext(nil), Fields{
		"field1": "val1",
		"field2": "val2",
		"field3": "val3",
		"field4": "val4",
	})
	ctx1.Logger().Info("Log 1")

	output1 := buf.String()
	assert.Contains(t, output1, "field1=val1")
	assert.Contains(t, output1, "field2=val2")
	assert.Contains(t, output1, "field3=val3")
	assert.Contains(t, output1, "field4=val4")
	buf.Reset()

	// Step 2: Create a second context with ONE field.
	// If the bug exists, Pass 1 walks to root, finds 1 field.
	// Pass 2 walks from root, finds 0 fields.
	// e.fields has length 1, but index 0 contains "field1" from previous log!

	ctx2 := WithLoggerField(NewContext(nil), "newfield", "newval")
	ctx2.Logger().Info("Log 2")

	output2 := buf.String()
	assert.Contains(t, output2, "newfield=newval")

	// If the fix works, Log 2 should NOT contain field1
	assert.NotContains(t, output2, "field1=val1")
}

// Simulate an external library wrapping the context
type externalKey string

func ExternalLibWrap(ctx context.Context, key externalKey, val string) context.Context {
	return context.WithValue(ctx, key, val)
}

func TestContext_MixedWrapping(t *testing.T) {
	var dbKey = &ContextKey{}
	var requestIdKey = &ContextKey{}

	dbVal := "the-db"

	// Base Golly Context with DB
	var ctx context.Context = WithValue(context.Background(), dbKey, dbVal)

	// Wrap with "External" stdlib context
	ctx = ExternalLibWrap(ctx, "std-key", "std-val")

	// Wrap again with Golly
	// WithValue accepts context.Context, returns *Context
	// We assign back to interface to keep chain generic
	ctx = WithValue(ctx, requestIdKey, "123")

	// Verify DB is visible from top
	val := ctx.Value(dbKey)
	assert.Equal(t, dbVal, val, "DB value should be visible through stdlib wrapper")

	// Verify Detach works if the top is Golly
	if gctx, ok := ctx.(*Context); ok {
		detached := gctx.Detach()
		valDetached := detached.Value(dbKey)
		assert.Equal(t, dbVal, valDetached, "DB value should be visible in Detached context through wrapper")
	}
}

func TestContextDetach(t *testing.T) {
	t.Run("Detached context preserves values", func(t *testing.T) {
		ctx := NewContext(context.Background())
		ctx = WithValue(ctx, "user_id", "123")
		ctx = WithValue(ctx, "tenant_id", "org-456")

		detached := ctx.Detach()

		assert.Equal(t, "123", detached.Value("user_id"))
		assert.Equal(t, "org-456", detached.Value("tenant_id"))
	})

	t.Run("Detached context is independent from cancellation", func(t *testing.T) {
		parent, cancel := context.WithCancel(context.Background())
		ctx := NewContext(parent)
		ctx = WithValue(ctx, "data", "important")

		detached := ctx.Detach()

		// Cancel original
		cancel()
		time.Sleep(10 * time.Millisecond)

		// Original should be cancelled
		assert.Error(t, ctx.Err())
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Fatal("Original context should be cancelled")
		}

		// Detached should NOT be cancelled
		assert.NoError(t, detached.Err())
		select {
		case <-detached.Done():
			t.Fatal("Detached context should NOT be cancelled")
		default:
			// Expected
		}

		// Value still accessible
		assert.Equal(t, "important", detached.Value("data"))
	})

	t.Run("Detached context preserves application", func(t *testing.T) {
		app, err := NewTestApplication(Options{Name: "TestApp"})
		require.NoError(t, err)

		ctx := NewContext(context.Background())
		ctx.application = app

		detached := ctx.Detach()

		assert.Equal(t, app, detached.Application())
		assert.Equal(t, "TestApp", detached.Application().Name)
	})

	t.Run("Detached context with deadline independence", func(t *testing.T) {
		parent, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		ctx := NewContext(parent)
		ctx = WithValue(ctx, "job_id", "job-123")

		detached := ctx.Detach()

		// Wait for parent to timeout
		time.Sleep(100 * time.Millisecond)

		// Parent should be done
		assert.Error(t, ctx.Err())

		// Detached should still be alive
		assert.NoError(t, detached.Err())
		assert.Equal(t, "job-123", detached.Value("job_id"))
	})

	t.Run("Use case: async projection handling", func(t *testing.T) {
		// Simulate request context with identity
		reqCtx, cancel := context.WithCancel(context.Background())
		ctx := NewContext(reqCtx)
		ctx = WithValue(ctx, "user_id", "user-123")
		ctx = WithValue(ctx, "tenant_id", "tenant-456")

		// Detach for async work
		asyncCtx := ctx.Detach()

		// Request ends (context cancelled)
		cancel()
		time.Sleep(10 * time.Millisecond)

		// Request context is done
		require.Error(t, ctx.Err())

		// But async context still has identity info
		assert.NoError(t, asyncCtx.Err())
		assert.Equal(t, "user-123", asyncCtx.Value("user_id"))
		assert.Equal(t, "tenant-456", asyncCtx.Value("tenant_id"))

		// This is safe to pass to async projection handlers
		processProjection := func(ctx *Context) {
			userID := ctx.Value("user_id")
			tenantID := ctx.Value("tenant_id")
			assert.Equal(t, "user-123", userID)
			assert.Equal(t, "tenant-456", tenantID)
		}

		processProjection(asyncCtx)
	})
}

/****************************************************
 * Benchmarks
 ****************************************************/

// Helper functions for benchmark setup
func buildCtxEmpty() *Context {
	return NewContext(context.TODO())
}

func buildCtxInherited() *Context {
	parentCtx := WithLoggerField(context.TODO(), "foo", "bar")
	return NewContext(parentCtx)
}

func BenchmarkContextLogger(b *testing.B) {
	cases := []struct {
		name string
		ctx  *Context
	}{
		{"Empty", buildCtxEmpty()},
		{"InheritedFields", buildCtxInherited()},
	}

	for _, tc := range cases {

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				e := tc.ctx.Logger()
				_ = e.Fields() // Prevent compiler optimization
				e.Release()
			}
		})
	}
}

func BenchmarkEntryPoolWarm(b *testing.B) {
	b.ReportAllocs()
	ctx := buildCtxEmpty()

	// warm pools outside timer
	for range 10000 {
		e := ctx.Logger()
		e.Release()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := ctx.Logger()
		e.Release()
	}
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

func TestContextCacheSharedWhenChildInitialized(t *testing.T) {
	root := NewContext(context.Background())
	childA := WithValue(root, "a", "1")
	childB := WithValue(root, "b", "2")

	loaderA := childA.Cache()
	loaderB := childB.Cache()
	loaderRoot := root.Cache()

	assert.Same(t, loaderA, loaderB)
	assert.Same(t, loaderA, loaderRoot)
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

func BenchmarkCollectFieldsOnly(b *testing.B) {
	// build a context chain with some fields
	root := &Context{fields: make([]Field, 4)}
	mid := &Context{parent: root, fields: make([]Field, 4)}
	leaf := &Context{parent: mid, fields: make([]Field, 4)}

	b.ReportAllocs()
	var acc []Field

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc = acc[:0]
		_ = leaf.collectFields(acc)
	}
}
