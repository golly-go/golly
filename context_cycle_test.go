package golly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
