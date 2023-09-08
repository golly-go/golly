package golly

import (
	"net/http"
	"testing"
)

func TestHeaderTokens(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		header  string
		want    []string
	}{
		{
			"no header",
			http.Header{},
			"Authorization",
			nil,
		},
		{
			"single token",
			http.Header{"Authorization": []string{"Bearer token"}},
			"Authorization",
			[]string{"Bearer token"},
		},
		{
			"multiple tokens",
			http.Header{"Accept": []string{"text/plain, application/json"}},
			"Accept",
			[]string{"text/plain", "application/json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeaderTokens(tt.headers, tt.header)

			if !compareStringSlices(got, tt.want) {
				t.Errorf("HeaderTokens() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderTokenContains(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		header  string
		value   string
		want    bool
	}{
		{
			"not contains",
			http.Header{"Authorization": []string{"Bearer token"}},
			"Authorization",
			"Basic token",
			false,
		},
		{
			"contains",
			http.Header{"Accept": []string{"text/plain, application/json"}},
			"Accept",
			"text/plain",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeaderTokenContains(tt.headers, tt.header, tt.value)

			if got != tt.want {
				t.Errorf("HeaderTokenContains() = %v; want %v", got, tt.want)
			}
		})
	}
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
