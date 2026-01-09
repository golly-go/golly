package golly

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
