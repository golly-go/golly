package golly

import (
	"errors"
	"testing"
)

// Test for FetchData using table-driven tests.
func TestFetchData(t *testing.T) {
	loader := NewDataLoader()

	tests := []struct {
		name      string
		key       any
		fetchFn   FetchFunc[any]
		expected  any
		expectErr bool
	}{
		{
			name: "fetch string",
			key:  "key1",
			fetchFn: func() (any, error) {
				return "Hello", nil
			},
			expected:  "Hello",
			expectErr: false,
		},
		{
			name: "fetch int",
			key:  "key2",
			fetchFn: func() (any, error) {
				return 42, nil
			},
			expected:  42,
			expectErr: false,
		},
		{
			name: "fetch with error",
			key:  "key3",
			fetchFn: func() (any, error) {
				return nil, errors.New("fetch failed")
			},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FetchData(loader, tt.key, tt.fetchFn)
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if result != tt.expected {
				t.Errorf("expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}

// ***************************************************************************
// *  Benches
// ***************************************************************************

// Benchmark for DataLoader Fetch method (cache hit and miss scenarios).
func BenchmarkDataLoaderFetch_CacheHit(b *testing.B) {
	loader := NewDataLoader()
	fetchFunc := func() (any, error) {
		return "benchmarkValue", nil
	}

	// Pre-load cache to simulate cache hit
	for i := 0; i < 100; i++ {
		_, _ = loader.Fetch(i, fetchFunc)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i % 100
		_, _ = loader.Fetch(key, fetchFunc)
	}
}

func BenchmarkDataLoaderFetch_CacheMiss(b *testing.B) {
	loader := NewDataLoader()
	fetchFunc := func() (any, error) {
		return "benchmarkValue", nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i + 100 // Ensure cache miss
		_, _ = loader.Fetch(key, fetchFunc)
	}
}

// Benchmark for FetchData generic function (cache hit and miss scenarios).
func BenchmarkFetchData_CacheHit(b *testing.B) {
	loader := NewDataLoader()
	fetchFunc := func() (int, error) {
		return 100, nil
	}

	// Pre-load cache to simulate cache hit
	for i := 0; i < 100; i++ {
		_, _ = FetchData(loader, i, fetchFunc)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i % 100
		_, _ = FetchData(loader, key, fetchFunc)
	}
}

func BenchmarkFetchData_CacheMiss(b *testing.B) {
	loader := NewDataLoader()
	fetchFunc := func() (int, error) {
		return 100, nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i + 100 // Ensure cache miss
		_, _ = FetchData(loader, key, fetchFunc)
	}
}
