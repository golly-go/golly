package golly

import (
	"net/http/httptest"
	"testing"
)

// /====
// / Tests
// /=====

func TestWebContext(t *testing.T) {
	// Add basic web context tests here if needed
}

// /===
// / benchmarks
// /===

func BenchmarkMakeRequestID(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = makeRequestID(nil)
	}
}

func BenchmarkNewWebContext(b *testing.B) {
	req := httptest.NewRequest("GET", "/api/v1/users/123/profile/settings", nil)
	w := httptest.NewRecorder()
	ctx := NewContext(req.Context())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewWebContext(ctx, req, w)
	}
}

func BenchmarkWebContextRequestBody(b *testing.B) {
	body := []byte(`{"key": "value", "data": [1, 2, 3, 4, 5]}`)
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	ctx := NewContext(req.Context())
	wctx := NewWebContext(ctx, req, w)
	wctx.body = body

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wctx.RequestBody()
	}
}
