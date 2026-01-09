package golly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGollyDetach_Repro(t *testing.T) {
	const dbKey ContextKey = "database"
	dbVal := "mock_db_connection"

	// 1. Base with Value
	ctx := context.Background()
	ctx = WithValue(ctx, dbKey, dbVal)

	// Verify base
	assert.Equal(t, dbVal, ctx.Value(dbKey))

	// 2. Wrap with Application (simulating middleware/app setup)
	// WithApplication uses NewContext which sets parent
	ctx = WithApplication(ctx, &Application{})

	// 3. Wrap with Engine (simulating eventsource)
	engineKey := ContextKey("engine")
	ctx = WithValue(ctx, engineKey, "engine")

	// 4. Detach
	gctx := ToGollyContext(ctx)
	detached := gctx.Detach()

	// 5. Verify Detached has DB
	// golly.Context.Value should find it walking up the new chain
	assert.Equal(t, dbVal, detached.Value(dbKey), "Detached context LOST the database key!")
}
