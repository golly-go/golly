package golly

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTokenizePath(b *testing.B) {
	b.Run("it should catch the patterns", func(b *testing.B) {
		examples := []string{
			"/test/1/2/3/{test:[1234]}",
			"/test/1/2/3/{test}",
			"/test/1/2/X/test",
			"/test/1/2/1234123412341234/test",
			"/test/1/2/3/testasdfasdf!@$!@#$123",
			"/test/1/2/3/test",
			"/test/1/2/3/test",
		}

		for _, url := range examples {
			tokenize(url)
		}
	})
}

func BenchmarkAddPath(b *testing.B) {

	b.Run("it should add handlers for each method supplied", func(b *testing.B) {
		b.ReportAllocs()
		re := NewRoute()
		re.Add("/test/1/2/3/{test:[1234]}", func(c WebContext) {}, POST|GET|PUT|DELETE)
	})
}

func BenchmarkFindRoute(b *testing.B) {
	r := NewRoute()
	r.Get("/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test", func(c WebContext) {})
	r.Get("/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34", func(c WebContext) {})

	b.Run("benchmark search - matching", func(b *testing.B) {
		FindRoute(r, "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/test")
	})

	b.Run("benchmark search - not matching", func(b *testing.B) {
		FindRoute(r, "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X")
	})

	b.Run("benchmark search - long not matching", func(b *testing.B) {
		FindRoute(r, "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34")
	})

	b.Run("benchmark search - long  matching", func(b *testing.B) {
		FindRoute(r, "/test/1/2/3/test/1/2/3/4/5/6/7/8/9/X/1/2/34/123/2134/1234/123412/123412/3412/4123412/34")
	})
}

func TestUse(t *testing.T) {
	root := NewRoute()
	root.Add("/test", func(c WebContext) {}, GET)

	t.Run("it should add to the children routes", func(t *testing.T) {
		root.Use(
			func(next HandlerFunc) HandlerFunc {
				return func(c WebContext) {
					next(c)
				}
			})

		assert.Len(t, root.middleware, 1)
		assert.Len(t, root.Children, 1)
		assert.Len(t, root.Children[0].middleware, 1)
	})
}

func TestAddHelpers(t *testing.T) {
	root := NewRoute()

	t.Run("it should add using the helper methods", func(t *testing.T) {

		re := root.Put("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&PUT)

		root.Post("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&POST)

		root.Options("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&OPTIONS)

		root.Delete("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&DELETE)

		root.Connect("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&CONNECT)

		root.Get("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&GET)

		root.Patch("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&PATCH)

		root.Head("/test", func(c WebContext) {})
		assert.NotZero(t, re.allowed&HEAD)

	})
}

func TestTokenizePath(t *testing.T) {
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

func TestRouteTree(t *testing.T) {
	root := NewRoute()

	t.Run("it should add a route 5 deep", func(t *testing.T) {
		root.Add("/1/2/3/4/5", func(c WebContext) {}, GET)

		assert.Equal(t, 5, root.Length())

		t.Run("it should only update 1 route", func(t *testing.T) {
			root.Add("/1/2/3/4/6", func(c WebContext) {}, GET)

			assert.Equal(t, 6, root.Length())

			root.Add("/1/2/3/4/6", func(c WebContext) {}, POST)

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
			root.Get(example, func(c WebContext) {})
		}

		for _, example := range examples {

			r := FindRoute(root, example)

			assert.NotNil(t, r)
			if _, ok := r.handlers[GET]; ok {
				assert.True(t, ok)
			}
		}
	})

	t.Run("it should add Match routes", func(t *testing.T) {
		for _, example := range examples {
			root.Match(example, func(c WebContext) {}, http.MethodGet, http.MethodOptions)
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
		root.Post("/path/{var}/rando@/{test:[0-2]+}", func(c WebContext) {})

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
