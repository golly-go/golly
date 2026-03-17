package golly

import (
	"net/http"
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

// TestWebContextURLParams verifies that URLParams correctly extracts dynamic
// route variables from the path, including after pool reuse via Reset.
func TestWebContextURLParams(t *testing.T) {
	root := NewRouteRoot()
	root.Delete("/_regression/{organizationID}", noOpHandler)
	root.Get("/users/{userID}/posts/{postID}", noOpHandler)

	t.Run("fresh context - single param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/_regression/019bd7b0-15ee-71e4-ae51-7523f6b9bb02", nil)
		w := httptest.NewRecorder()

		wctx := NewWebContext(req.Context(), req, w)
		wctx.route = FindRoute(root, req.URL.Path)

		params := wctx.URLParams()
		assert.Equal(t, "019bd7b0-15ee-71e4-ae51-7523f6b9bb02", params.Get("organizationID"))
	})

	t.Run("fresh context - multiple params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/42/posts/99", nil)
		w := httptest.NewRecorder()

		wctx := NewWebContext(req.Context(), req, w)
		wctx.route = FindRoute(root, req.URL.Path)

		params := wctx.URLParams()
		assert.Equal(t, "42", params.Get("userID"))
		assert.Equal(t, "99", params.Get("postID"))
	})

	// Regression: Reset with a nil/empty wctx.segments (fresh pool object) and a
	// non-empty segments slice caused copy() to be a no-op, leaving URLParams empty.
	t.Run("reset from shorter path to longer path - bug repro", func(t *testing.T) {
		// Simulate a pool WebContext whose previous request was a short path (2 segments).
		short := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()
		wctx := NewWebContext(short.Context(), short, w)

		// Now reset for a longer path with a dynamic segment.
		long := httptest.NewRequest(http.MethodDelete, "/_regression/019bd7b0-15ee-71e4-ae51-7523f6b9bb02", nil)
		path := long.URL.Path
		stack := make([]string, makePathCount(path))
		pathSegments(stack, path)

		wctx.Reset(long.Context(), long, w, stack)
		wctx.route = FindRoute(root, path)

		params := wctx.URLParams()
		assert.Equal(t, "019bd7b0-15ee-71e4-ae51-7523f6b9bb02", params.Get("organizationID"),
			"URLParams must not be empty after Reset with more segments than previous request")
	})

	t.Run("reset from longer path to shorter path", func(t *testing.T) {
		// Previous request was long (4 segments).
		long := httptest.NewRequest(http.MethodGet, "/users/42/posts/99", nil)
		w := httptest.NewRecorder()
		wctx := NewWebContext(long.Context(), long, w)

		// Reset for a shorter dynamic path (3 segments).
		next := httptest.NewRequest(http.MethodDelete, "/_regression/abc123", nil)
		path := next.URL.Path
		stack := make([]string, makePathCount(path))
		pathSegments(stack, path)

		wctx.Reset(next.Context(), next, w, stack)
		wctx.route = FindRoute(root, path)

		params := wctx.URLParams()
		assert.Equal(t, "abc123", params.Get("organizationID"),
			"URLParams must not bleed stale segments from a previous longer request")
	})

	t.Run("URLParams cached after first call", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/_regression/cached-id", nil)
		w := httptest.NewRecorder()

		wctx := NewWebContext(req.Context(), req, w)
		wctx.route = FindRoute(root, req.URL.Path)

		p1 := wctx.URLParams()
		p2 := wctx.URLParams()
		assert.Same(t, p1, p2, "URLParams should return the same pointer on repeated calls")
	})
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

	var buf [64]byte
	id := makeRequestID(buf[:])

	// interactive with verify:
	assert.NotEmpty(t, id)
	// format: hostname/000001
	// 30 chars + 1 + 6 = 37 chars
	assert.Equal(t, 37, len(id))
	assert.True(t, strings.HasPrefix(id, hostname+"/"))

	// Verify zero allocation by checking if the id string backing array points to our stack buffer
	// Note: this is a bit hacky and unsafe-pointer-y, but for a test it's fine
	// Or even better, run a small benchmark here?
	// Let's just trust the logic: if len+pad <= cap, it slices.
	// Since 37 <= 64, it should slice.
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
