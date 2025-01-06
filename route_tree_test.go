package golly

import (
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTokenizePath(b *testing.B) {
	benchmarks := []struct {
		name string
		path string
	}{
		{"simple path", "/test/1/2/3/test"},
		{"path with variable", "/test/1/2/3/{test}"},
		{"path with variable and matcher", "/test/1/2/3/{test:[1234]}"},
		{"complex path with long string", "/test/1/2/1234123412341234/test"},
		{"path with special characters", "/test/1/2/3/testasdfasdf!@$!@#$123"},
		{"repeated path", "/test/1/2/3/test"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tokenize(bm.path)
			}
		})
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
			re := NewRoute()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				re.Add(bm.path, func() {}, POST|GET|PUT|DELETE)
			}
		})
	}
}

// Benchmark for finding routes
func BenchmarkFindRoute(b *testing.B) {
	r := NewRoute()
	r.Get("/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test", func() {})
	r.Get("/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34", func() {})

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
			for i := 0; i < b.N; i++ {
				FindRoute(r, bm.path)
			}
		})
	}
}
func TestUse(t *testing.T) {
	root := NewRoute()
	root.Add("/test", func() {}, GET)

	t.Run("it should add to the children routes", func(t *testing.T) {
		root.Use(
			func(next HandlerFunc) HandlerFunc {
				return func() {
					next()
				}
			})

		assert.Len(t, root.middleware, 1)
		assert.Len(t, root.children, 1)
		assert.Len(t, root.children[0].middleware, 1)
	})
}

func TestAddHelpers(t *testing.T) {
	root := NewRoute()

	t.Run("it should add using the helper methods", func(t *testing.T) {

		re := root.Put("/test", func() {})
		assert.NotZero(t, re.allowed&PUT)

		root.Post("/test", func() {})
		assert.NotZero(t, re.allowed&POST)

		root.Options("/test", func() {})
		assert.NotZero(t, re.allowed&OPTIONS)

		root.Delete("/test", func() {})
		assert.NotZero(t, re.allowed&DELETE)

		root.Connect("/test", func() {})
		assert.NotZero(t, re.allowed&CONNECT)

		root.Get("/test", func() {})
		assert.NotZero(t, re.allowed&GET)

		root.Patch("/test", func() {})
		assert.NotZero(t, re.allowed&PATCH)

		root.Head("/test", func() {})
		assert.NotZero(t, re.allowed&HEAD)

	})
}

func TestTokenizePathSimple(t *testing.T) {
	t.Run("it should catch the patterns", func(t *testing.T) {
		examples := []string{
			"/test/1/2/3/{test:[1234]}",
			"/test/1/2/3/{test:[//]}",
			"/test/1/2/3/{test}",
			"/test/1/2/3/{test}",
			"/test/1/2/3/test",
			"/test/1/2/3/test/",
		}

		for _, url := range examples {
			tokens := tokenize(url)
			assert.Equal(t, 5, len(tokens))
		}
	})
}

func TestTokenizePath(t *testing.T) {
	t.Run("it should tokenize paths correctly", func(t *testing.T) {
		tests := []struct {
			url      string
			expected []RouteToken
		}{
			{
				url: "/test/1/2/3/{test:[1234]}",
				expected: []RouteToken{
					RoutePath{Path: "test"},
					RoutePath{Path: "1"},
					RoutePath{Path: "2"},
					RoutePath{Path: "3"},
					RouteVariable{Name: "test", Matcher: "[1234]"},
				},
			},
			{
				url: "/test/1/2/3/{test:[//]}",
				expected: []RouteToken{
					RoutePath{Path: "test"},
					RoutePath{Path: "1"},
					RoutePath{Path: "2"},
					RoutePath{Path: "3"},
					RouteVariable{Name: "test", Matcher: "[//]"},
				},
			},
			{
				url: "/test/1/2/3/{test}",
				expected: []RouteToken{
					RoutePath{Path: "test"},
					RoutePath{Path: "1"},
					RoutePath{Path: "2"},
					RoutePath{Path: "3"},
					RouteVariable{Name: "test"},
				},
			},
			{
				url: "/test/1/2/3/test",
				expected: []RouteToken{
					RoutePath{Path: "test"},
					RoutePath{Path: "1"},
					RoutePath{Path: "2"},
					RoutePath{Path: "3"},
					RoutePath{Path: "test"},
				},
			},
			{
				url: "/simple/path",
				expected: []RouteToken{
					RoutePath{Path: "simple"},
					RoutePath{Path: "path"},
				},
			},
			{
				url:      "/",
				expected: []RouteToken{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.url, func(t *testing.T) {
				tokens := tokenize(tt.url)

				if len(tokens) != len(tt.expected) {
					t.Errorf("Expected %d tokens, got %d for path %s", len(tt.expected), len(tokens), tt.url)
					return
				}

				for i, token := range tokens {
					expectedToken := tt.expected[i]

					// Check if both tokens are of the same type
					if fmt.Sprintf("%T", token) != fmt.Sprintf("%T", expectedToken) {
						t.Errorf("Token type mismatch at index %d: expected %T, got %T", i, expectedToken, token)
					}

					// Verify token value
					if token.Value() != expectedToken.Value() {
						t.Errorf("Value mismatch at index %d: expected %s, got %s", i, expectedToken.Value(), token.Value())
					}

					// Verify matcher for RouteVariable
					if v, ok := token.(RouteVariable); ok {
						expVar := expectedToken.(RouteVariable)
						if v.Matcher != expVar.Matcher {
							t.Errorf("Matcher mismatch at index %d: expected %s, got %s", i, expVar.Matcher, v.Matcher)
						}
					}
				}
			})
		}
	})
}

func TestRouteTree(t *testing.T) {
	root := NewRoute()

	t.Run("it should add a route 5 deep", func(t *testing.T) {
		root.Add("/1/2/3/4/5", func() {}, GET)

		assert.Equal(t, 5, root.Length())

		t.Run("it should only update 1 route", func(t *testing.T) {
			root.Add("/1/2/3/4/6", func() {}, GET)

			assert.Equal(t, 6, root.Length())

			root.Add("/1/2/3/4/6", func() {}, POST)

			assert.Equal(t, 6, root.Length())
		})
	})
}

func TestAddRouteHelpers(t *testing.T) {
	root := NewRoute()

	examples := []string{
		"/test/1/2/3/test1234",
		"/test/1/2/3/test",
		"/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test",
		"/test/1/2/3/testING",
		"/test/1/2/3/test1234567",
		"/test/1/2/3/test/1234123/1234/asdf/1234",
	}

	t.Run("it should add GET routes", func(t *testing.T) {
		for _, example := range examples {
			root.Get(example, func() {})
		}

		for _, example := range examples {

			r := FindRoute(root, example)
			fmt.Printf("%s %#v\n", example, r)

			assert.NotNil(t, r)

			if _, ok := r.handlers[GET]; ok {
				assert.True(t, ok)
			}
		}
	})

	t.Run("it should add Match routes", func(t *testing.T) {
		for _, example := range examples {
			root.Match(example, func() {}, http.MethodGet, http.MethodOptions)
		}

		for _, example := range examples {

			r := FindRoute(root, example)

			allowed := r.Allow()

			assert.Len(t, allowed, 2)
			assert.Contains(t, allowed, "GET")
			assert.Contains(t, allowed, "OPTIONS")

			assert.NotNil(t, r)

		}
	})

	t.Run("it should add a path variable and match by that", func(t *testing.T) {
		root.Post("/path/{var}/rando@/{test:[0-2]+}", func() {})

		r := FindRoute(root, "/path/123/rando@/01012")
		assert.NotNil(t, r)
		r = FindRoute(root, "/path/123/rando@/abcd")
		assert.Nil(t, r)
	})

	t.Run("it should create a namespace", func(t *testing.T) {
		r := root.Namespace("/test", func(r *Route) {
			assert.Equal(t, r.Token.Value(), "test")
		})
		assert.Empty(t, r.Allow())
	})

}

// Test function for buildPath
// Test function for buildPath
func TestBuildPath(t *testing.T) {
	tests := []struct {
		name     string
		route    *Route
		prefix   string
		expected []string
	}{
		{
			name: "Static path with GET method",
			route: &Route{
				Token:   RoutePath{Path: "users"},
				allowed: GET,
			},
			expected: []string{"[GET] /users"},
		},
		{
			name: "Veradic path with POST method",
			route: &Route{
				Token:   RouteVariable{Name: "id", Matcher: "[0-9]+"},
				allowed: POST,
			},
			expected: []string{"[POST] /{id:[0-9]+}"},
		},
		{
			name: "Nested routes with mixed methods",
			route: &Route{
				Token:   RoutePath{Path: "api"},
				allowed: GET,
				children: []*Route{
					{
						Token:   RoutePath{Path: "v1"},
						allowed: POST,
						children: []*Route{
							{
								Token:   RouteVariable{Name: "userID", Matcher: "[0-9]+"},
								allowed: GET | PUT,
							},
						},
					},
				},
			},
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
				Token:   RoutePath{Path: "empty"},
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
