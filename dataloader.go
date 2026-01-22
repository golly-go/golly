package golly

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// FetchFunc is a generic function type that returns a value of type T and an error.
type FetchFunc[T any] func() (T, error)

// entry represents a single cached result. If multiple goroutines request the same key,
// only the first will execute the fetch. Others will wait on `ready`.
type entry struct {
	ready chan struct{}
	value any
	err   error
	done  uint32
}

// DataLoader is a concurrency-safe, request-scoped cache with single-flight semantics per key.
// It caches both values and errors intentionally.
type DataLoader struct {
	mu    sync.Mutex
	cache map[any]*entry
}

// NewDataLoader initializes and returns a new instance of DataLoader.
func NewDataLoader() *DataLoader {
	return &DataLoader{
		cache: make(map[any]*entry),
	}
}

// Has returns true if the key exists in the loader (whether loaded or in-flight).
func (dl *DataLoader) Has(key any) bool {
	dl.mu.Lock()
	_, ok := dl.cache[key]
	dl.mu.Unlock()
	return ok
}

// Get returns the value for key if present and fully loaded.
// If the key is in-flight, Get returns (nil, false).
func (dl *DataLoader) Get(key any) (any, bool) {
	dl.mu.Lock()
	e, ok := dl.cache[key]
	dl.mu.Unlock()
	if !ok {
		return nil, false
	}

	// Non-blocking check: only return if loaded.
	select {
	case <-e.ready:
		return e.value, true
	default:
		return nil, false
	}
}

// GetWait returns the cached value for key if present, waiting if the fetch is in-flight.
// If the key does not exist, it returns (nil, false).
func (dl *DataLoader) GetWait(key any) (any, bool) {
	dl.mu.Lock()
	e, ok := dl.cache[key]
	dl.mu.Unlock()
	if !ok {
		return nil, false
	}

	<-e.ready
	return e.value, true
}

// Set sets a value for key and marks it as ready.
// If another goroutine is already fetching or has set this key, the first writer wins.
func (dl *DataLoader) Set(key any, value any) {
	dl.mu.Lock()
	e, ok := dl.cache[key]
	if !ok {
		e = &entry{ready: make(chan struct{})}
		dl.cache[key] = e
	}
	dl.mu.Unlock()

	if atomic.CompareAndSwapUint32(&e.done, 0, 1) {
		e.value = value
		e.err = nil
		close(e.ready)
	}
}

// SetError sets an error for key and marks it as ready.
// This matches the intentional "cache errors" behavior. First writer wins.
func (dl *DataLoader) SetError(key any, err error) {
	dl.mu.Lock()
	e, ok := dl.cache[key]
	if !ok {
		e = &entry{ready: make(chan struct{})}
		dl.cache[key] = e
	}
	dl.mu.Unlock()

	if atomic.CompareAndSwapUint32(&e.done, 0, 1) {
		e.value = nil
		e.err = err
		close(e.ready)
	}
}

// Delete deletes the key from the cache. If the key is in-flight, it is removed from the map,
// but any existing waiters on the entry will still be released when the fetch completes.
func (dl *DataLoader) Delete(key any) {
	dl.mu.Lock()
	delete(dl.cache, key)
	dl.mu.Unlock()
}

// Clear clears the cache.
func (dl *DataLoader) Clear() {
	dl.mu.Lock()
	dl.cache = make(map[any]*entry)
	dl.mu.Unlock()
}

// Fetch retrieves a value by key. If the key is not present, it executes fetchFn exactly once per key,
// caching both value and error, and releases all waiters.
func (dl *DataLoader) Fetch(key any, fetchFn FetchFunc[any]) (any, error) {
	// Fast path: check if present.
	dl.mu.Lock()
	if e, ok := dl.cache[key]; ok {
		dl.mu.Unlock()
		<-e.ready
		return e.value, e.err
	}

	// Miss: create in-flight entry.
	e := &entry{ready: make(chan struct{})}
	dl.cache[key] = e
	dl.mu.Unlock()

	// Compute without holding the lock.
	value, err := fetchFn()

	if atomic.CompareAndSwapUint32(&e.done, 0, 1) {
		e.value = value
		e.err = err
		close(e.ready)
		return value, err
	}

	<-e.ready
	return e.value, e.err
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

// GetData returns the typed value if present and fully loaded, without waiting.
// If the key is in-flight or absent, it returns (zero, false).
func GetData[T any](loader *DataLoader, key any) (T, bool) {
	var zero T

	value, ok := loader.Get(key)
	if !ok {
		return zero, false
	}

	castedResult, ok := value.(T)
	if !ok {
		return zero, false
	}

	return castedResult, true
}
