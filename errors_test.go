package golly

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		code       uint
		cause      error
		extensions []map[string]any
		wantStatus int
		wantCause  error
		wantExt    map[string]any
	}{
		{
			name:       "Without extensions",
			code:       http.StatusBadRequest,
			cause:      errors.New("test error"),
			extensions: nil,
			wantStatus: http.StatusBadRequest,
			wantCause:  errors.New("test error"),
			wantExt:    nil,
		},
		{
			name:       "Without extensions and nil cause",
			code:       http.StatusNotFound,
			cause:      nil,
			extensions: nil,
			wantStatus: http.StatusNotFound,
			wantCause:  nil,
			wantExt:    nil,
		},
		{
			name:  "With single extension map",
			code:  http.StatusInternalServerError,
			cause: errors.New("server error"),
			extensions: []map[string]any{
				{"key1": "value1", "key2": 42},
			},
			wantStatus: http.StatusInternalServerError,
			wantCause:  errors.New("server error"),
			wantExt:    map[string]any{"key1": "value1", "key2": 42},
		},
		{
			name:  "With multiple extension maps",
			code:  http.StatusUnauthorized,
			cause: errors.New("unauthorized"),
			extensions: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2", "key3": 123},
			},
			wantStatus: http.StatusUnauthorized,
			wantCause:  errors.New("unauthorized"),
			wantExt:    map[string]any{"key1": "value1", "key2": "value2", "key3": 123},
		},
		{
			name:  "With overlapping keys in extension maps",
			code:  http.StatusConflict,
			cause: errors.New("conflict"),
			extensions: []map[string]any{
				{"key1": "first"},
				{"key1": "second", "key2": "value2"},
			},
			wantStatus: http.StatusConflict,
			wantCause:  errors.New("conflict"),
			wantExt:    map[string]any{"key1": "second", "key2": "value2"}, // Last value wins
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.cause, tt.extensions...)

			assert.NotNil(t, err)
			assert.Equal(t, tt.wantStatus, err.Status())
			assert.Equal(t, http.StatusText(int(tt.code)), err.statusText)

			if tt.wantCause == nil {
				assert.Nil(t, err.Unwrap())
			} else {
				assert.Equal(t, tt.wantCause.Error(), err.Unwrap().Error())
			}

			if tt.wantExt == nil {
				assert.Nil(t, err.Extensions())
			} else {
				assert.Equal(t, tt.wantExt, err.Extensions())
			}
		})
	}
}

func TestError_Status(t *testing.T) {
	tests := []struct {
		name   string
		code   uint
		expect int
	}{
		{"Status 200", http.StatusOK, http.StatusOK},
		{"Status 400", http.StatusBadRequest, http.StatusBadRequest},
		{"Status 404", http.StatusNotFound, http.StatusNotFound},
		{"Status 500", http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, nil)
			assert.Equal(t, tt.expect, err.Status())
		})
	}
}

func TestError_Message(t *testing.T) {
	tests := []struct {
		name    string
		message string
		expect  string
	}{
		{
			name:    "With message",
			message: "test message",
			expect:  "test message",
		},
		{
			name:    "Empty message",
			message: "",
			expect:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(http.StatusBadRequest, nil)
			err.message = tt.message
			assert.Equal(t, tt.expect, err.Message())
		})
	}
}

func TestError_Extensions(t *testing.T) {
	tests := []struct {
		name       string
		extensions []map[string]any
		expect     map[string]any
	}{
		{
			name:       "With extensions",
			extensions: []map[string]any{{"key1": "value1", "key2": 42}},
			expect:     map[string]any{"key1": "value1", "key2": 42},
		},
		{
			name:       "Nil extensions (no extension arg)",
			extensions: nil,
			expect:     nil,
		},
		{
			name:       "Empty extensions map",
			extensions: []map[string]any{{}},
			expect:     map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *Error
			if tt.extensions == nil {
				err = NewError(http.StatusBadRequest, nil)
			} else {
				err = NewError(http.StatusBadRequest, nil, tt.extensions...)
			}
			ext := err.Extensions()

			if tt.expect == nil {
				assert.Nil(t, ext)
			} else {
				assert.Equal(t, tt.expect, ext)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name    string
		message string
		cause   error
		expect  string
	}{
		{
			name:    "With message",
			message: "custom error message",
			cause:   nil,
			expect:  "custom error message",
		},
		{
			name:    "Without message but with cause",
			message: "",
			cause:   errors.New("underlying error"),
			expect:  "underlying error",
		},
		{
			name:    "Message takes precedence over cause",
			message: "explicit message",
			cause:   errors.New("underlying error"),
			expect:  "explicit message",
		},
		{
			name:    "Neither message nor cause",
			message: "",
			cause:   nil,
			expect:  "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(http.StatusBadRequest, tt.cause)
			err.message = tt.message
			assert.Equal(t, tt.expect, err.Error())
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	tests := []struct {
		name   string
		cause  error
		expect error
	}{
		{
			name:   "With cause",
			cause:  errors.New("test cause"),
			expect: errors.New("test cause"),
		},
		{
			name:   "Nil cause",
			cause:  nil,
			expect: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(http.StatusBadRequest, tt.cause)
			unwrapped := err.Unwrap()

			if tt.expect == nil {
				assert.Nil(t, unwrapped)
			} else {
				assert.Equal(t, tt.expect.Error(), unwrapped.Error())
			}
		})
	}
}

func TestError_WithMeta(t *testing.T) {
	tests := []struct {
		name       string
		initialExt []map[string]any
		key        string
		value      any
		expectExt  map[string]any
		expectCopy bool // Whether a new error should be returned
	}{
		{
			name:       "Add to nil extensions",
			initialExt: nil,
			key:        "newKey",
			value:      "newValue",
			expectExt:  map[string]any{"newKey": "newValue"},
			expectCopy: true,
		},
		{
			name:       "Add to existing extensions",
			initialExt: []map[string]any{{"existing": "value"}},
			key:        "newKey",
			value:      "newValue",
			expectExt:  map[string]any{"existing": "value", "newKey": "newValue"},
			expectCopy: true,
		},
		{
			name:       "Overwrite existing key",
			initialExt: []map[string]any{{"key": "old"}},
			key:        "key",
			value:      "new",
			expectExt:  map[string]any{"key": "new"},
			expectCopy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var original *Error
			if tt.initialExt == nil {
				original = NewError(http.StatusBadRequest, nil)
			} else {
				original = NewError(http.StatusBadRequest, nil, tt.initialExt...)
			}
			modified := original.WithMeta(tt.key, tt.value)

			// Verify copy-on-write behavior
			assert.NotSame(t, original, modified, "WithMeta should return a new error instance")

			// Verify original is unchanged
			originalExt := original.Extensions()
			if tt.initialExt == nil {
				assert.Nil(t, originalExt)
			} else {
				assert.Equal(t, tt.initialExt[0], originalExt)
			}

			// Verify modified has the new key-value pair
			assert.Equal(t, tt.expectExt, modified.Extensions())
		})
	}
}

func TestError_WithExtensions(t *testing.T) {
	tests := []struct {
		name                 string
		initialExt           []map[string]any
		newExt               map[string]any
		expectExt            map[string]any
		shouldReturnOriginal bool
	}{
		{
			name:                 "Empty map should return original",
			initialExt:           []map[string]any{{"existing": "value"}},
			newExt:               map[string]any{},
			expectExt:            map[string]any{"existing": "value"},
			shouldReturnOriginal: true,
		},
		{
			name:                 "Add to nil extensions",
			initialExt:           nil,
			newExt:               map[string]any{"key1": "value1", "key2": 42},
			expectExt:            map[string]any{"key1": "value1", "key2": 42},
			shouldReturnOriginal: false,
		},
		{
			name:                 "Merge with existing extensions",
			initialExt:           []map[string]any{{"existing": "value"}},
			newExt:               map[string]any{"new1": "value1", "new2": 42},
			expectExt:            map[string]any{"existing": "value", "new1": "value1", "new2": 42},
			shouldReturnOriginal: false,
		},
		{
			name:                 "Overwrite existing keys",
			initialExt:           []map[string]any{{"key": "old", "keep": "this"}},
			newExt:               map[string]any{"key": "new"},
			expectExt:            map[string]any{"key": "new", "keep": "this"},
			shouldReturnOriginal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var original *Error
			if tt.initialExt == nil {
				original = NewError(http.StatusBadRequest, nil)
			} else {
				original = NewError(http.StatusBadRequest, nil, tt.initialExt...)
			}
			modified := original.WithExtensions(tt.newExt)

			if tt.shouldReturnOriginal {
				// Empty map case - should return original
				assert.Same(t, original, modified, "WithExtensions should return original for empty map")
			} else {
				// Should return a new instance
				assert.NotSame(t, original, modified, "WithExtensions should return a new error instance")

				// Verify original is unchanged
				originalExt := original.Extensions()
				if tt.initialExt == nil {
					assert.Nil(t, originalExt)
				} else {
					assert.Equal(t, tt.initialExt[0], originalExt)
				}
			}

			// Verify modified has expected extensions
			if tt.expectExt == nil {
				assert.Nil(t, modified.Extensions())
			} else {
				assert.Equal(t, tt.expectExt, modified.Extensions())
			}
		})
	}
}

func TestError_HTTPErrorInterface(t *testing.T) {
	t.Run("Error implements HTTPError interface", func(t *testing.T) {
		var httpErr HTTPError = NewError(http.StatusBadRequest, errors.New("test"))
		assert.NotNil(t, httpErr)
		assert.Equal(t, http.StatusBadRequest, httpErr.Status())
	})
}

func TestCopyExt(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]any
		expect map[string]any
	}{
		{
			name:   "Nil map",
			input:  nil,
			expect: map[string]any{},
		},
		{
			name:   "Empty map",
			input:  map[string]any{},
			expect: map[string]any{},
		},
		{
			name:   "Map with values",
			input:  map[string]any{"key1": "value1", "key2": 42},
			expect: map[string]any{"key1": "value1", "key2": 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := copyExt(tt.input)

			assert.Equal(t, tt.expect, result)

			// Verify it's a copy, not the same map
			if len(tt.input) > 0 {
				result["newKey"] = "newValue"
				_, exists := tt.input["newKey"]
				assert.False(t, exists, "Modifying copy should not affect original")
			}
		})
	}
}

func TestError_Chaining(t *testing.T) {
	t.Run("Chaining WithMeta and WithExtensions", func(t *testing.T) {
		original := NewError(http.StatusBadRequest, errors.New("cause"))

		// Chain multiple modifications
		modified := original.
			WithMeta("key1", "value1").
			WithMeta("key2", "value2").
			WithExtensions(map[string]any{"key3": "value3", "key4": 42})

		// Verify original is unchanged
		assert.Nil(t, original.Extensions())

		// Verify all modifications are present
		expected := map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": 42,
		}
		assert.Equal(t, expected, modified.Extensions())
	})

	t.Run("Chaining preserves status code and cause", func(t *testing.T) {
		cause := errors.New("original cause")
		original := NewError(http.StatusNotFound, cause)

		modified := original.WithMeta("key", "value")

		assert.Equal(t, original.Status(), modified.Status())
		assert.Equal(t, original.Unwrap(), modified.Unwrap())
		assert.Equal(t, cause.Error(), modified.Unwrap().Error())
	})
}
