package golly

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock identity implementation for testing
type testIdentity struct {
	name  string
	valid bool
	err   error
}

func (t *testIdentity) Valid() error {
	if t.valid {
		return nil
	}
	return t.err
}

func (t *testIdentity) IsValid() bool {
	return t.valid
}

func TestIdentityToContext(t *testing.T) {
	t.Run("Sets identity in context", func(t *testing.T) {
		ident := &testIdentity{name: "user1", valid: true}
		ctx := NewContext(context.Background())

		resultCtx := IdentityToContext(ctx, ident)

		retrieved := IdentityFromContext[*testIdentity](resultCtx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "user1", retrieved.name)
		assert.True(t, retrieved.IsValid())
	})

	t.Run("Creates new context if nil", func(t *testing.T) {
		ident := &testIdentity{name: "user2", valid: true}

		resultCtx := IdentityToContext(nil, ident)

		assert.NotNil(t, resultCtx)
		retrieved := IdentityFromContext[*testIdentity](resultCtx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "user2", retrieved.name)
	})
}

func TestIdentityFromContext(t *testing.T) {
	t.Run("Returns identity when present", func(t *testing.T) {
		ident := &testIdentity{name: "user1", valid: true}
		ctx := NewContext(context.Background())
		ctx = IdentityToContext(ctx, ident)

		retrieved := IdentityFromContext[*testIdentity](ctx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "user1", retrieved.name)
	})

	t.Run("Returns zero value when not present", func(t *testing.T) {
		ctx := NewContext(context.Background())

		retrieved := IdentityFromContext[*testIdentity](ctx)
		assert.Nil(t, retrieved)
	})

	t.Run("Returns zero value for wrong type", func(t *testing.T) {
		ident := &testIdentity{name: "user1", valid: true}
		ctx := NewContext(context.Background())
		ctx = IdentityToContext(ctx, ident)

		// Try to retrieve with different type
		type otherIdentity struct{ testIdentity }
		retrieved := IdentityFromContext[*otherIdentity](ctx)
		assert.Nil(t, retrieved)
	})
}

func TestIdentity_Valid(t *testing.T) {
	t.Run("Valid identity returns nil error", func(t *testing.T) {
		ident := &testIdentity{valid: true}
		assert.NoError(t, ident.Valid())
		assert.True(t, ident.IsValid())
	})

	t.Run("Invalid identity returns error", func(t *testing.T) {
		expectedErr := errors.New("invalid identity")
		ident := &testIdentity{valid: false, err: expectedErr}

		assert.Error(t, ident.Valid())
		assert.Equal(t, expectedErr, ident.Valid())
		assert.False(t, ident.IsValid())
	})
}
