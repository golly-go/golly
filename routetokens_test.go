package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			assert.Equal(t, 6, len(tokens))
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
					{value: "/"},
					{value: "test"},
					{value: "1"},
					{value: "2"},
					{value: "3"},
					{value: "test", matcher: "[1234]"},
				},
			},
			{
				url: "/test/1/2/3/{test:[//]}",
				expected: []RouteToken{
					{value: "/"},
					{value: "test"},
					{value: "1"},
					{value: "2"},
					{value: "3"},
					{value: "test", matcher: "[//]"},
				},
			},
			{
				url: "/test/1/2/3/{test}",
				expected: []RouteToken{
					{value: "/"},
					{value: "test"},
					{value: "1"},
					{value: "2"},
					{value: "3"},
					{value: "test"},
				},
			},
			{
				url: "/test/1/2/3/test",
				expected: []RouteToken{
					{value: "/"},
					{value: "test"},
					{value: "1"},
					{value: "2"},
					{value: "3"},
					{value: "test"},
				},
			},
			{
				url: "/simple/path",
				expected: []RouteToken{
					{value: "/"},
					{value: "simple"},
					{value: "path"},
				},
			},
			{
				url: "/",
				expected: []RouteToken{
					{value: "/"},
				},
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

					assert.Equal(t, token.matcher, expectedToken.matcher)
					assert.Equal(t, token.value, expectedToken.value)

				}
			})
		}
	})
}

// ***************************************************************************
// *  Benches
// ***************************************************************************

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

func BenchmarkSegmentPath(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Root", "/"},
		{"Simple Path", "/users"},
		{"Nested Path", "/api/v1/users"},
		{"Trailing Slash", "/orders/"},
		{"No Leading Slash", "products/items"},
		{"Multiple Slashes", "/api//users"},
		{"Complex Path", "/a/b/c/d/e/f/g"},
		{"Empty Path", ""},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			stack := make([]string, makePathCount(tt.path))

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				pathSegments(stack, tt.path)
			}
		})
	}
}
