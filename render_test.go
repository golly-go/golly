package golly

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {

	t.Run("RenderJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()
		wctx := NewTestWebContext(req, resp)

		response := map[string]string{"key": "value"}
		Render(wctx, FormatTypeJSON, response)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"key":"value"}`, resp.Body.String())
	})

	t.Run("RenderBytes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()
		wctx := NewTestWebContext(req, resp)

		response := []byte("raw bytes")
		Render(wctx, FormatTypeJSON, response)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.Equal(t, "raw bytes", resp.Body.String())
	})

	t.Run("RenderError", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()
		wctx := NewTestWebContext(req, resp)

		response := fmt.Errorf("an error occurred")
		Render(wctx, FormatTypeJSON, response)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.Equal(t, "an error occurred", resp.Body.String())
	})

	t.Run("RenderHEAD", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/", nil)
		resp := httptest.NewRecorder()
		wctx := NewTestWebContext(req, resp)

		response := map[string]string{"key": "value"}
		Render(wctx, FormatTypeJSON, response)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Empty(t, resp.Body.String())
	})
}

func TestMarshalContent(t *testing.T) {
	t.Run("MarshalString", func(t *testing.T) {
		input := "simple string"
		output, err := marshalContent(input)

		assert.NoError(t, err)
		assert.Equal(t, "simple string", string(output))
	})

	t.Run("MarshalBytes", func(t *testing.T) {
		input := []byte("raw bytes")
		output, err := marshalContent(input)

		assert.NoError(t, err)
		assert.Equal(t, "raw bytes", string(output))
	})

	t.Run("MarshalUnsupportedType", func(t *testing.T) {
		input := struct {
			InvalidField func()
		}{}

		output, err := marshalContent(input)

		assert.Error(t, err)
		assert.Nil(t, output)
	})
}

func BenchmarkRender(b *testing.B) {
	// Prepare test inputs
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	wctx := NewTestWebContext(req, resp)

	format := FormatTypeJSON
	res := map[string]string{"key": "value"}

	// Benchmark various scenarios
	b.Run("OptimizedRender", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Render(wctx, format, res)
		}
	})

	b.Run("RenderWithBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Render(wctx, format, bytes.NewBufferString(`{"key":"value"}`))
		}
	})

	b.Run("RenderWithBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Render(wctx, format, []byte(`{"key":"value"}`))
		}
	})

	b.Run("RenderWithError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Render(wctx, format, fmt.Errorf("an error occurred"))
		}
	})
}

func Benchmark_marshalContent(b *testing.B) {
	b.Run("MarshalString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = marshalContent("example text")
		}
	})

	b.Run("MarshalBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = marshalContent([]byte("example text"))
		}
	})

	b.Run("MarshalStringsBuilder", func(b *testing.B) {
		sb := &strings.Builder{}
		sb.WriteString("example text")
		for i := 0; i < b.N; i++ {
			_, _ = marshalContent(sb)
		}
	})

	b.Run("MarshalBytesBuffer", func(b *testing.B) {
		bb := &bytes.Buffer{}
		bb.WriteString("example text")
		for i := 0; i < b.N; i++ {
			_, _ = marshalContent(bb)
		}
	})

	b.Run("InvalidType", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = marshalContent(123) // Expecting an error
		}
	})
}
