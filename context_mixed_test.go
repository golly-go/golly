package golly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
