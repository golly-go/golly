package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWildcardStringMatch(t *testing.T) {
	tests := []struct {
		name     string
		wcString string
		input    string
		expected bool
	}{
		{"Exact match", "hello", "hello", true},
		{"Prefix wildcard match", "he*", "hello", true},
		{"Suffix wildcard match", "*lo", "hello", true},
		{"Prefix and suffix wildcard match", "h*o", "hello", true},
		{"No match", "hi*", "hello", false},
		{"Wildcard, but not matching suffix", "*z", "hello", false},
		{"Empty wildcard", "", "", true},
		{"Empty input", "abc*", "", false},
		{"Match entire string with *", "*", "anything", true},
		{"Multiple wildcard sections", "h*o*", "howdy", true},
		{"Complex multi-wildcard match", "h*t*h*", "hitchhiker", true},
		{"Multiple * match terminating *", "h*t*h*", "hitch", true},
		{"Wildcard at start, middle, and end", "*ell*", "hello", true},
		{"Only suffix match", "*o", "hello", true},
		{"Partial prefix mismatch", "h*o*", "abc", false},
		{"Wildcard but empty input", "*", "", true},
		{"Empty Pattern", "", "asdfas", false},
		{"Ridiculous Wildcards", "*s*****", "asdfas", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WildcardMatch(tt.wcString, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark for WildcardString.Match
func BenchmarkWildcardStringMatch(b *testing.B) {
	tests := []struct {
		name     string
		wcString string
		input    string
	}{
		{"Exact match", "hello", "hello"},
		{"Prefix wildcard match", "he*", "hello"},
		{"Suffix wildcard match", "*lo", "hello"},
		{"Prefix and suffix wildcard match", "h*o", "hello"},
		{"Wildcard with no match", "*z", "hello"},
		{"Full wildcard", "*", "anything"},
		{"Multiple wildcard", "h*t*h*", "hitch"},
		{"Empty pattern", "", ""},
		{"Long string with wildcard", "prefix*suffix", "prefix_some_long_text_suffix"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				WildcardMatch(tt.wcString, tt.input)
			}
		})
	}
}
