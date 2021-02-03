package golly

import (
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
		re := NewRouteEntry()
		re.Add("/test/1/2/3/{test:[1234]}", func(c Context) {}, POST|GET|PUT|DELETE)
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
	root := NewRouteEntry()

	t.Run("it should add a route 5 deep", func(t *testing.T) {
		root.Add("/1/2/3/4/5", func(c Context) {}, GET)

		assert.Equal(t, 5, root.Length())

		t.Run("it should only update 1 route", func(t *testing.T) {
			root.Add("/1/2/3/4/6", func(c Context) {}, GET)

			assert.Equal(t, 6, root.Length())

			root.Add("/1/2/3/4/6", func(c Context) {}, POST)

			assert.Equal(t, 6, root.Length())
		})
	})

}
