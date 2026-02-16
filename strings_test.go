package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test for ASCIICompair function
func TestASCIICompair(t *testing.T) {
	tests := []struct {
		name     string
		str1     string
		str2     string
		expected bool
	}{
		{"Identical lowercase", "hello", "hello", true},
		{"Identical uppercase", "HELLO", "HELLO", true},
		{"Mixed case match", "HeLLo", "hElLo", true},
		{"Different lengths", "hello", "worlds", false},
		{"Completely different", "abc", "xyz", false},
		{"One empty string", "test", "", false},
		{"Both empty strings", "", "", true},
		{"Single character match", "A", "a", true},
		{"Single character mismatch", "A", "b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ASCIICompair(tt.str1, tt.str2)
			assert.Equal(t, tt.expected, result, "Failed test case: %s", tt.name)
		})
	}

}

var asciiSink bool

func BenchmarkASCIICompair(b *testing.B) {
	tests := []struct {
		name string
		str1 string
		str2 string
	}{
		{"Exact match", "hello", "hello"},
		{"Case-insensitive match", "Hello", "hello"},
		{"Different lengths", "hello", "hell"},
		{"Completely different strings", "hello", "world"},
		{"Special characters", "!@#$%", "!@#$%"},
		{"Case-insensitive with special characters", "HeLLo!", "hElLo!"},
		{"Numeric characters", "12345", "12345"},
		{"Large strings match", "a very long string that matches exactly", "a very long string that matches exactly"},
		{"Large strings mismatch", "a very long string that matches exactly", "a very long string that differs slightly"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			s1, s2 := tt.str1, tt.str2
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				asciiSink = ASCIICompair(s1, s2)
			}
		})
	}

}

var sink bool

func BenchmarkDiffLenForce(b *testing.B) {
	s1a, s2a := "hello", "hell"
	s1b, s2b := "hello!", "hell" // still different lengths, but not identical pair

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i&1 == 0 {
			sink = ASCIICompair(s1a, s2a)
		} else {
			sink = ASCIICompair(s1b, s2b)
		}
	}
}
