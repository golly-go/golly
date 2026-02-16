package golly

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataLoaderSet(t *testing.T) {
	loader := NewDataLoader()
	loader.Set("key1", "Hello")

	called := false
	value, err := loader.Fetch("key1", func() (any, error) {
		called = true
		return "Hello", nil
	})

	assert.False(t, called)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", value)
}

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

func TestDataLoaderFetchSingleFlight(t *testing.T) {
	loader := NewDataLoader()
	var calls int32

	startFetch := make(chan struct{})
	releaseFetch := make(chan struct{})

	fetchFn := func() (any, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(startFetch)
		}
		<-releaseFetch
		return "value", nil
	}

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make(chan any, goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			value, err := loader.Fetch("key", fetchFn)
			assert.NoError(t, err)
			results <- value
		}()
	}

	<-startFetch
	close(releaseFetch)
	wg.Wait()
	close(results)

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
	for value := range results {
		assert.Equal(t, "value", value)
	}
}

func TestDataLoaderFirstToStartWins(t *testing.T) {
	loader := NewDataLoader()

	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})

	fetchFn := func() (any, error) {
		close(fetchStarted)
		<-releaseFetch
		return "fetch", nil
	}

	resultCh := make(chan any, 1)
	go func() {
		value, err := loader.Fetch("key", fetchFn)
		assert.NoError(t, err)
		resultCh <- value
	}()

	<-fetchStarted
	loader.Set("key", "set") // This should NOT override the in-flight fetch
	close(releaseFetch)

	result := <-resultCh
	assert.Equal(t, "fetch", result, "First to start (fetch) should win over Set()")
}

func TestDataLoaderHas(t *testing.T) {
	loader := NewDataLoader()

	assert.False(t, loader.Has("missing"))

	loader.Set("key", "value")
	assert.True(t, loader.Has("key"))
}

func TestDataLoaderGet(t *testing.T) {
	loader := NewDataLoader()

	// Get on missing key
	val, ok := loader.Get("missing")
	assert.False(t, ok)
	assert.Nil(t, val)

	// Get on set key
	loader.Set("key", "value")
	val, ok = loader.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Get on in-flight key returns false
	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})

	go func() {
		loader.Fetch("inflight", func() (any, error) {
			close(fetchStarted)
			<-releaseFetch
			return "result", nil
		})
	}()

	<-fetchStarted
	val, ok = loader.Get("inflight")
	assert.False(t, ok, "Get should return false for in-flight keys")
	assert.Nil(t, val)
	close(releaseFetch)
}

func TestDataLoaderGetWait(t *testing.T) {
	loader := NewDataLoader()

	// GetWait on missing key
	val, ok := loader.GetWait("missing")
	assert.False(t, ok)
	assert.Nil(t, val)

	// GetWait on completed key
	loader.Set("key", "value")
	val, ok = loader.GetWait("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// GetWait waits for in-flight fetch
	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})

	go func() {
		loader.Fetch("inflight", func() (any, error) {
			close(fetchStarted)
			<-releaseFetch
			return "result", nil
		})
	}()

	<-fetchStarted

	resultCh := make(chan any, 1)
	go func() {
		val, ok := loader.GetWait("inflight")
		assert.True(t, ok)
		resultCh <- val
	}()

	close(releaseFetch)
	result := <-resultCh
	assert.Equal(t, "result", result)
}

func TestDataLoaderSetError(t *testing.T) {
	loader := NewDataLoader()

	testErr := errors.New("test error")
	loader.SetError("key", testErr)

	val, err := loader.Fetch("key", func() (any, error) {
		t.Fatal("fetch should not be called")
		return nil, nil
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Nil(t, val)
}

func TestDataLoaderDelete(t *testing.T) {
	loader := NewDataLoader()

	loader.Set("key", "value")
	assert.True(t, loader.Has("key"))

	loader.Delete("key")
	assert.False(t, loader.Has("key"))
}

func TestDataLoaderClear(t *testing.T) {
	loader := NewDataLoader()

	loader.Set("key1", "value1")
	loader.Set("key2", "value2")
	assert.True(t, loader.Has("key1"))
	assert.True(t, loader.Has("key2"))

	loader.Clear()
	assert.False(t, loader.Has("key1"))
	assert.False(t, loader.Has("key2"))
}

func TestGetData(t *testing.T) {
	loader := NewDataLoader()

	// GetData on missing key
	val, ok := GetData[string](loader, "missing")
	assert.False(t, ok)
	assert.Equal(t, "", val)

	// GetData on existing key with correct type
	loader.Set("key", "value")
	val, ok = GetData[string](loader, "key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// GetData with wrong type
	loader.Set("int-key", 42)
	strVal, ok := GetData[string](loader, "int-key")
	assert.False(t, ok)
	assert.Equal(t, "", strVal)

	// GetData with correct type
	intVal, ok := GetData[int](loader, "int-key")
	assert.True(t, ok)
	assert.Equal(t, 42, intVal)
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
	for i := range 100 {
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
		_, _ = loader.Fetch(i, fetchFunc)
	}
}

func BenchmarkDataLoaderGet_CacheHit(b *testing.B) {
	loader := NewDataLoader()
	loader.Set("key", "value")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loader.Get("key")
	}
}

func BenchmarkDataLoaderGet_CacheMiss(b *testing.B) {
	loader := NewDataLoader()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loader.Get("missing")
	}
}

// Benchmark for FetchData generic function (cache hit and miss scenarios).
func BenchmarkFetchData_CacheHit(b *testing.B) {
	loader := NewDataLoader()
	fetchFunc := func() (int, error) {
		return 100, nil
	}

	// Pre-load cache to simulate cache hit
	for i := range 100 {
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
		key := "miss-key"
		_, _ = FetchData(loader, key, fetchFunc)
		loader.Delete(key)
	}
}
