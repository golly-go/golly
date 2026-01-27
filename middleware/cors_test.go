package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Empty input", "", nil},
		{"Single header", "content-type", []string{"Content-Type"}},
		{"Multiple headers", "content-type, accept, authorization", []string{"Content-Type", "Accept", "Authorization"}},
		{"Headers with spaces", "  content-type  ,  accept ,authorization  ", []string{"Content-Type", "Accept", "Authorization"}},
		{"Mixed case headers", "Content-Type,ACCEPT,Authorization", []string{"Content-Type", "Accept", "Authorization"}},
		{"Duplicate commas", "content-type,,accept,authorization", []string{"Content-Type", "Accept", "Authorization"}},
		{"Trailing commas", "content-type,accept,", []string{"Content-Type", "Accept"}},
		{"Leading commas", ",content-type,accept", []string{"Content-Type", "Accept"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaders(tt.input)
			assert.Equal(t, tt.expected, result, "Expected canonicalized headers")
		})
	}
}

func TestCors_ValidateHeaders(t *testing.T) {
	c := &cors{
		allHeaders: false,
		headers:    []string{"Content-Type", "Authorization"},
	}

	assert.True(t, c.validateHeaders("Content-Type"))
	assert.True(t, c.validateHeaders("content-type"))
	assert.True(t, c.validateHeaders("Content-Type, Authorization"))
	assert.False(t, c.validateHeaders("X-Custom-Header"))
	assert.True(t, c.validateHeaders("")) // Empty is allowed
}

func TestCors_IsOriginAllowed(t *testing.T) {
	c := &cors{
		allOrigins:     false,
		allowedOrigins: []string{"https://example.com"},
		worigins:       []string{"*.example.org"},
	}

	assert.True(t, c.isOriginAllowed("https://example.com"))
	assert.True(t, c.isOriginAllowed("https://EXAMPLE.com"))     // Case insensitive
	assert.True(t, c.isOriginAllowed("https://sub.example.org")) // Wildcard
	assert.False(t, c.isOriginAllowed("https://other.com"))
}

func TestCors_IsMethodAllowed(t *testing.T) {
	c := &cors{
		methods: []string{"GET", "POST"},
	}

	assert.True(t, c.isMethodAllowed("GET"))
	assert.True(t, c.isMethodAllowed("get"))     // Case insensitive
	assert.True(t, c.isMethodAllowed("OPTIONS")) // Always allowed
	assert.False(t, c.isMethodAllowed("DELETE"))
}

/***************************************************
 * Benchmarks
 ***************************************************/

func BenchmarkParseHeaders(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{"Single header", "Content-Type"},
		{"Multiple headers", "Content-Type, Accept, Authorization"},
		{"Headers with spaces", "  Content-Type  ,  Accept ,Authorization  "},
		{"Long header list", "Header1,Header2,Header3,Header4,Header5,Header6,Header7,Header8,Header9,Header10"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = parseHeaders(tt.input)
				// strings.Split(tt.input, ",")
			}
		})
	}
}

func BenchmarkIsOriginAllowed(b *testing.B) {
	c := &cors{
		allOrigins: false,
		allowedOrigins: []string{
			"https://allowed-origin.com",
			"https://another-origin.com",
		},
		worigins: []string{"*.example.com"},
	}

	tests := []struct {
		name   string
		origin string
	}{
		{"Exact match", "https://allowed-origin.com"},
		{"Wildcard match", "sub.example.com"},
		{"Mixed Case Origin", "https://Allowed-Origin.com"},
		{"Non-matching origin", "https://unauthorized.com"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = c.isOriginAllowed(tt.origin)
			}
		})
	}
}

func BenchmarkIsMethodAllowed(b *testing.B) {
	c := &cors{
		methods: []string{"GET", "POST", "OPTIONS"},
	}

	tests := []struct {
		name   string
		method string
	}{
		{"Valid method - GET", "GET"},
		{"Valid method - OPTIONS", "OPTIONS"},
		{"Invalid method - PATCH", "PATCH"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = c.isMethodAllowed(tt.method)
			}
		})
	}
}

func BenchmarkAreHeadersAllowed(b *testing.B) {
	c := &cors{
		allHeaders: false,
		headers:    []string{"Content-Type", "Authorization", "Accept"},
	}

	tests := []struct {
		name    string
		headers []string
	}{
		{"Valid headers", []string{"Content-Type", "Authorization"}},
		{"Invalid headers", []string{"X-Custom-Header"}},
		{"Mixed headers", []string{"Content-Type", "X-Custom-Header"}},
		{"No headers", []string{}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = c.areHeadersAllowed(tt.headers)
			}
		})
	}
}
