package golly

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteVariables(t *testing.T) {
	// Setup Route Tree with route variables

	root := NewRouteRoot()

	root.Get("/orders", noOpHandler).
		Get("/orders/{orderID}", noOpHandler).
		Get("/orders/{orderID}/items", noOpHandler).
		Get("/orders/{orderID}/items/{itemID}", noOpHandler).
		Get("/named/{named:\\d+}", noOpHandler)

	tests := []struct {
		name           string
		route          *Route
		path           string
		expectedValues map[string]string
	}{
		{
			name: "Multiple Variables in Path",
			path: "/orders/456/items/123",
			expectedValues: map[string]string{
				"orderID": "456",
				"itemID":  "123",
			},
		},
		{
			name: "Multiple Variables in Path",
			path: "/orders/456/items/123",
			expectedValues: map[string]string{
				"orderID": "456",
				"itemID":  "123",
			},
		},
		{
			name: "Single level",
			path: "/orders/789",
			expectedValues: map[string]string{
				"orderID": "789",
			},
		},
		{
			name: "Single level - named",
			path: "/named/789",
			expectedValues: map[string]string{
				"named": "789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := FindRoute(root, tt.path)

			vars := routeVariables(route, pathSegments(tt.path))

			assert.Equal(t, len(tt.expectedValues), vars.Len())
			for key, expected := range tt.expectedValues {
				actual := vars.Get(key)

				assert.Equal(t, expected, actual)
			}
		})
	}
}

func TestUse(t *testing.T) {
	root := NewRouteRoot()
	root.Add("/test", noOpHandler, GET)

	t.Run("it should add to the children routes", func(t *testing.T) {
		root.Use(
			func(next HandlerFunc) HandlerFunc {
				return func(wctx *WebContext) {
					next(wctx)
				}
			})

		assert.Len(t, root.middleware, 1)
		assert.Len(t, root.children, 1)
		// We no longer apply middleware to the children route handler
		// instead we apply downwards as we chain
		assert.Len(t, root.children[0].middleware, 0)
	})
}

func TestAddHelpers(t *testing.T) {
	root := NewRouteRoot()

	t.Run("it should add using the helper methods", func(t *testing.T) {

		root.Put("/test", noOpHandler).
			Post("/test", noOpHandler).
			Options("/test", noOpHandler).
			Delete("/test", noOpHandler).
			Connect("/test", noOpHandler).
			Get("/test", noOpHandler).
			Patch("/test", noOpHandler).
			Head("/test", noOpHandler)

		assert.Len(t, root.children, 1)

		re := root.children[0]

		assert.NotZero(t, re.allowed&PUT)
		assert.NotZero(t, re.allowed&POST)
		assert.NotZero(t, re.allowed&OPTIONS)
		assert.NotZero(t, re.allowed&DELETE)
		assert.NotZero(t, re.allowed&CONNECT)
		assert.NotZero(t, re.allowed&GET)
		assert.NotZero(t, re.allowed&PATCH)
		assert.NotZero(t, re.allowed&HEAD)

	})
}

func TestRouteTree(t *testing.T) {
	root := NewRouteRoot()

	t.Run("it should add a route 5 deep", func(t *testing.T) {
		root.Add("/1/2/3/4/5", noOpHandler, GET)

		assert.Equal(t, 5, root.Length())

		t.Run("it should only update 1 route", func(t *testing.T) {
			root.Add("/1/2/3/4/6", noOpHandler, GET)

			assert.Equal(t, 6, root.Length())

			root.Add("/1/2/3/4/6", noOpHandler, POST)

			assert.Equal(t, 6, root.Length())
		})
	})
}

func TestAddRouteHelpers(t *testing.T) {
	root := NewRouteRoot()

	examples := []string{
		"/",
		"/test/1/2/3/test1234",
		"/test/1/2/3/test",
		"/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test",
		"/test/1/2/3/testING",
		"/test/1/2/3/test1234567",
		"/test/1/2/3/test/1234123/1234/asdf/1234",
	}

	t.Run("it should add GET routes", func(t *testing.T) {
		for _, example := range examples {
			root.Get(example, noOpHandler)
		}

		for _, example := range examples {
			r := FindRoute(root, example)

			assert.NotNil(t, r, "No route found for: %s", example)

			if r.root != nil {
				assert.Equal(t, root, r.root)
			}

			if r.handlers[methodIndex(GET)] != nil {
				// Handler exists
			} else {
				t.Error("Handler should exist")
			}
		}
	})

	t.Run("it should add a path variable and match by that", func(t *testing.T) {
		root.Post("/path/{var}/rando@/{test:[0-2]+}", noOpHandler)

		r := FindRoute(root, "/path/123/rando@/01012")
		assert.NotNil(t, r)
		r = FindRoute(root, "/path/123/rando@/abcd")
		assert.Nil(t, r)
	})

	t.Run("it should create a namespace", func(t *testing.T) {
		var r *Route
		root.Namespace("/test", func(route *Route) {
			r = route
		})

		assert.Equal(t, r.token.value, "test")

		assert.Empty(t, r.Allow())
	})

}

// Test function for buildPath
func TestBuildPath(t *testing.T) {
	handler := noOpHandler

	tests := []struct {
		name     string
		route    *Route
		prefix   string
		expected []string
	}{
		{
			name:     "Static path with GET method",
			route:    NewRouteRoot().Get("/users", handler),
			expected: []string{"[GET] /users"},
		},
		{
			name:     "Veradic path with POST method",
			route:    NewRouteRoot().Add("/{id:[0-9]+}", handler, POST),
			expected: []string{"[POST] /{id:[0-9]+}"},
		},
		{
			name: "Nested routes with mixed methods",
			route: NewRouteRoot().
				Add("/api", handler, GET).
				Add("/api/v1", handler, POST).
				Add("/api/v1/{userID:[0-9]+}", handler, GET|PUT),
			expected: []string{
				"[GET] /api",
				"[POST] /api/v1",
				"[GET] /api/v1/{userID:[0-9]+}",
				"[PUT] /api/v1/{userID:[0-9]+}",
			},
		},
		{
			name: "No allowed methods",
			route: &Route{
				token:   &RouteToken{value: "empty"},
				allowed: 0,
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPath(tt.route, "")

			sort.Strings(got)         // Sort the actual result
			sort.Strings(tt.expected) // Sort the expected result

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRouteRequest(t *testing.T) {
	tests := []struct {
		name       string
		routes     func(*Route)
		request    *http.Request
		expectCode int
		expectBody string
	}{
		{
			name: "Route found with GET method",
			routes: func(root *Route) {
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request:    httptest.NewRequest(http.MethodGet, "/test", nil),
			expectCode: http.StatusOK,
			expectBody: "GET /test",
		},
		{
			name: "Route not found",
			routes: func(root *Route) {
				// No routes are added
			},
			request:    httptest.NewRequest(http.MethodGet, "/not-found", nil),
			expectCode: http.StatusNotFound,
			expectBody: "",
		},
		{
			name: "Method not allowed",
			routes: func(root *Route) {
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request:    httptest.NewRequest(http.MethodPost, "/test", nil),
			expectCode: http.StatusMethodNotAllowed,
			expectBody: "",
		},
		{
			name: "Route with middleware",
			routes: func(root *Route) {
				root.Use(func(next HandlerFunc) HandlerFunc {
					return func(ctx *WebContext) {
						ctx.writer.Header().Set("X-Middleware", "Executed")
						next(ctx)
					}
				})
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request:    httptest.NewRequest(http.MethodGet, "/test", nil),
			expectCode: http.StatusOK,
			expectBody: "GET /test",
		},
		{
			name: "CORS preflight request",
			routes: func(root *Route) {
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodOptions, "/test", nil)
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
				return req
			}(),
			expectCode: http.StatusOK,
			expectBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(Options{})

			tt.routes(app.routes)

			w := httptest.NewRecorder()
			RouteRequest(app, tt.request, w)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectCode, resp.StatusCode)
			assert.Equal(t, tt.expectBody, string(body))
		})
	}
}

// ***************************************************************************
// *  Benches
// ***************************************************************************

func BenchmarkRouteRequest(b *testing.B) {
	tests := []struct {
		name    string
		setup   func(*Route)
		request *http.Request
	}{
		{
			name: "Route found with GET method",
			setup: func(root *Route) {
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
		},
		{
			name: "Route not found",
			setup: func(root *Route) {
				// No routes are added
			},
			request: httptest.NewRequest(http.MethodGet, "/not-found", nil),
		},
		{
			name: "Method not allowed",
			setup: func(root *Route) {
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request: httptest.NewRequest(http.MethodPost, "/test", nil),
		},
		{
			name: "Route with middleware",
			setup: func(root *Route) {
				root.Use(func(next HandlerFunc) HandlerFunc {
					return func(ctx *WebContext) {
						next(ctx)
					}
				})
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
		},
		{
			name: "CORS preflight request",
			setup: func(root *Route) {
				root.Get("/test", func(ctx *WebContext) {
					ctx.writer.WriteHeader(http.StatusOK)
					_, _ = ctx.writer.Write([]byte("GET /test"))
				})
			},
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodOptions, "/test", nil)
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
				return req
			}(),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			app := NewApplication(Options{})
			tt.setup(app.routes)

			var w *httptest.ResponseRecorder

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				w = httptest.NewRecorder()

				RouteRequest(app, tt.request, w)
			}
		})
	}
}

type BenchMockWriter struct {
	h http.Header
}

func (m *BenchMockWriter) Header() http.Header         { return m.h }
func (m *BenchMockWriter) Write(b []byte) (int, error) { return len(b), nil }
func (m *BenchMockWriter) WriteHeader(statusCode int)  {}

func BenchmarkRouteRequest_ZeroAlloc(b *testing.B) {
	app := NewApplication(Options{})
	app.routes.Get("/test", func(ctx *WebContext) {
		ctx.writer.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	mw := &BenchMockWriter{h: make(http.Header)}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		RouteRequest(app, req, mw)
	}
}

func BenchmarkAddPath(b *testing.B) {
	benchmarks := []struct {
		name string
		path string
	}{
		{"simple path", "/test"},
		{"nested path", "/test/1/2/3"},
		{"path with variable", "/test/1/2/3/{test}"},
		{"path with variable and matcher", "/test/1/2/3/{test:[1234]}"},
		{"long static path", "/test/1/2/3/4/5/6/7/8/9"},
		{"complex path with mix of variables", "/test/{section}/1/{id:[0-9]+}/3/{action}"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			re := NewRouteRoot()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				re.Add(bm.path, noOpHandler, POST|GET|PUT|DELETE)
			}
		})
	}
}

// Benchmark for finding routes
func BenchmarkFindRoute(b *testing.B) {
	r := NewRouteRoot()
	r.Get("/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test", noOpHandler)
	r.Get("/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34", noOpHandler)

	benchmarks := []struct {
		name string
		path string
	}{
		{"matching", "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test"},
		{"not matching", "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X"},
		{"long not matching", "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34"},
		{"long matching", "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				FindRoute(r, bm.path)
			}
		})
	}
}

func BenchmarkRouteVariables(b *testing.B) {
	handler := noOpHandler

	root := NewRouteRoot()
	root.Get("/orders", handler).
		Get("/orders/{orderID}", handler).
		Get("/orders/{orderID}/items", handler).
		Get("/orders/{orderID}/items/{itemID}", handler).
		Get("/named/{named:\\d+}", handler)

	tests := []struct {
		name string
		path string
	}{
		{"Simple path", "/orders/123"},
		{"Nested path", "/orders/123/items/456"},
		{"Long path", "/orders/123/items/456/details/789"},
		{"Single segment", "/orders"},
		{"No match", "/invalid/path"},
	}

	for _, tt := range tests {
		route := FindRoute(root, tt.path)
		path := strings.Split(tt.path, "/")

		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = routeVariables(route, path)
			}
		})
	}
}
