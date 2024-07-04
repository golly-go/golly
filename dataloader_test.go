package golly

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDataLoader(t *testing.T) {
	dl := NewDataLoader()
	assert.NotNil(t, dl)
}

func TestFetch_SimpleKey(t *testing.T) {
	gctx := NewContext(context.TODO())

	fetchFn := func(gctx Context) (string, error) {
		return "data", nil
	}

	result, err := LoadData(gctx, "simpleKey", fetchFn)
	assert.NoError(t, err)
	assert.Equal(t, "data", result)

	// Test cache hit
	cachedResult, err := LoadData(gctx, "simpleKey", fetchFn)
	assert.NoError(t, err)
	assert.Equal(t, "data", cachedResult)
}

func TestFetch_Error(t *testing.T) {
	gctx := NewContext(context.TODO())

	fetchFn := func(gctx Context) (string, error) {
		return "", errors.New("fetch error")
	}

	_, err := LoadData(gctx, "errorKey", fetchFn)

	assert.Error(t, err)
	assert.Equal(t, "fetch error", err.Error())
}
