package golly

import (
	"fmt"
	"sync"
)

type cacheResult struct {
	err   error
	value any
}

// FetchFunc is a generic function type that returns a value of type T and an error.
type FetchFunc[T any] func() (T, error)

// DataLoader is a concurrency-safe structure that caches and retrieves values based on keys.
// It uses sync.Map internally for efficient thread-safe storage.
type DataLoader struct {
	cache map[any]cacheResult
	mu    sync.RWMutex
}

// NewDataLoader initializes and returns a new instance of DataLoader.
func NewDataLoader() *DataLoader {
	return &DataLoader{
		cache: make(map[any]cacheResult),
	}
}

func (dl *DataLoader) Has(key any) bool {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	_, ok := dl.cache[key]
	return ok
}

func (dl *DataLoader) Get(key any) (any, bool) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	value, ok := dl.cache[key]
	return value.value, ok
}

func (dl *DataLoader) Set(key any, value any) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	dl.cache[key] = cacheResult{value: value, err: nil}
}

func (dl *DataLoader) Delete(key any) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	delete(dl.cache, key)
}

// Fetch attempts to retrieve a value from the cache by key.
// If the key is not found, it calls the provided fetch function to obtain the value, caches it,
// and then returns the result.
func (dl *DataLoader) Fetch(key any, fetchFn FetchFunc[any]) (any, error) {
	// Read lock for cache lookup
	dl.mu.RLock()
	if result, ok := dl.cache[key]; ok {
		dl.mu.RUnlock()
		return result.value, result.err
	}
	dl.mu.RUnlock()

	// Write lock for cache miss
	dl.mu.Lock()

	// Double-check to avoid race condition
	if result, ok := dl.cache[key]; ok {
		dl.mu.Unlock()
		return result.value, result.err
	}

	// Release lock before calling fetchFn to avoid deadlock
	// This allows other goroutines to proceed while we fetch
	dl.mu.Unlock()

	// Call fetch function without holding the lock
	value, err := fetchFn()

	// Re-acquire lock to store the result
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Double-check again in case another goroutine already cached it
	if result, ok := dl.cache[key]; ok {
		return result.value, result.err
	}

	dl.cache[key] = cacheResult{value: value, err: err}

	return value, err
}

// FetchData is a generic function that retrieves typed data from the DataLoader.
// It ensures type safety by casting the cached result to the desired type T.
func FetchData[T any](loader *DataLoader, key any, fetchFn FetchFunc[T]) (T, error) {
	var zero T

	result, err := loader.Fetch(key, func() (any, error) {
		return fetchFn()
	})

	if err != nil {
		return zero, err
	}

	castedResult, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("type assertion to target type failed")
	}

	return castedResult, nil
}

func GetData[T any](loader *DataLoader, key any) (T, bool) {
	var zero T

	if value, ok := loader.Get(key); ok {
		castedResult, ok := value.(T)
		if !ok {
			return zero, false
		}

		return castedResult, true
	}

	return zero, false
}

// Documentation
/*
Package golly provides a lightweight DataLoader to cache and fetch values associated with specific keys.

DataLoader:
- `NewDataLoader() *DataLoader` - Creates a new DataLoader instance.
- `Fetch` - Retrieves data based on a key, stores it in the cache, and returns the result.
- `FetchData` - A generic helper function for typed data fetching, leveraging Go generics.

Caching Mechanism:
- `sync.Map` is used to provide thread-safe concurrent access to the cache.
- If the key is not found, the fetch function is executed, and the result is cached.

Error Handling:
- Errors during fetch operations are propagated to ensure robust error detection.

Usage:
  loader := NewDataLoader()
  result, err := loader.Fetch("key1", func() (interface{}, error) {
      return doSomeWork()
  })

  typedResult, err := FetchData("key2", func() (string, error) {
      return "World", nil
  })
*/
