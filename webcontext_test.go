package golly

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// /====
// / Tests
// /=====

func TestWebContext(t *testing.T) {
	// Add basic web context tests here if needed
}

func TestMakeRequestID_LongHostname(t *testing.T) {
	// Save original hostname
	originalHostname := hostname
	defer func() { hostname = originalHostname }()

	// Set a long hostname to trigger the buffer overflow condition
	// Buffer in WebContext is 36 bytes.
	// We want hostname + "/" + count + padding > 36
	// If count is 1 digit, we need padding of 5 zeroes.
	// Total needed = len(hostname) + 1 (/) + 1 (cnt) + 5 (pad) = len(hostname) + 7
	// If len(hostname) is 30, total = 37. Cap is 36. Boom.
	hostname = strings.Repeat("a", 30)

	// We expect this to NOT panic now.

	var buf [36]byte
	id := makeRequestID(buf[:])

	// interactive with verify:
	assert.NotEmpty(t, id)
	// format: hostname/000001
	// 30 chars + 1 + 6 = 37 chars
	assert.Equal(t, 37, len(id))
	assert.True(t, strings.HasPrefix(id, hostname+"/"))
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
