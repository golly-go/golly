package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	var examples = []struct {
		example string
		result  []string
	}{
		{"1234", []string{"1234"}},
		{"1234,5678", []string{"1234", "5678"}},
		{"1234, 5678", []string{"1234", "5678"}},
		{"1234,5678, 9123", []string{"1234", "5678", "9123"}},
	}

	for _, example := range examples {
		assert.Equal(t, Tokenize(example.example, ','), example.result)
	}
}

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

// Benchmark for ASCIICompair function
func BenchmarkASCIICompair(b *testing.B) {
	tests := []struct {
		name string
		str1 string
		str2 string
	}{
		{"Short strings", "hello", "HELLO"},
		{"Long strings", "LongerTestStringForBenchmark", "longerteststringforbenchmark"},
		{"Mismatch", "MismatchTest", "MistakeTest"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = ASCIICompair(tt.str1, tt.str2)
			}
		})
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"camelCase", "camelCase", "camel_case"},
		{"PascalCase", "PascalCase", "pascal_case"},
		{"AllUppercase", "HTMLParser", "html_parser"},
		{"SingleWord", "Word", "word"},
		{"AcronymWithWord", "JSONData", "json_data"},
		{"MultipleUpper", "HTTPRequestParser", "http_request_parser"},
		{"LowercaseString", "simple", "simple"},
		{"WithNumbers", "User2Profile", "user2_profile"},
		{"ComplexCase", "XMLHTTPRequestHandler", "xmlhttp_request_handler"},
		{"EmptyString", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SnakeCase(tt.input)
			assert.Equal(t, tt.expected, result, "Failed test case: %s", tt.name)
		})
	}
}
