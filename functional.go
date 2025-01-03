package golly

import (
	"reflect"
	"sync"
)

func Map[T any, R any](list []T, fn func(T) R) []R {
	return MapWithIndex[T, R](list, func(x T, _ int) R {
		return fn(x)
	})
}

func MapWithIndex[T any, R any](list []T, fn func(T, int) R) []R {
	ret := make([]R, len(list))

	for pos, x := range list {
		ret[pos] = fn(x, pos)
	}

	return ret
}

func Compact[T any](list []T) []T {
	ret := []T{}

	for _, x := range list {
		var c interface{} = x

		val := reflect.ValueOf(c)

		if c == nil || val.Kind() == reflect.Ptr && val.IsNil() {
			continue
		}

		ret = append(ret, x)
	}
	return ret
}

func Filter[T any](list []T, fn func(T) bool) []T {
	ret := []T{}

	for _, x := range list {
		if result := fn(x); result {
			ret = append(ret, x)
		}
	}

	return ret
}

func Find[T any](list []T, fn func(T) bool) *T {
	for _, x := range list {
		if fn(x) {
			return &x
		}
	}
	return nil
}

func Flatten[T any](list [][]T) []T {
	ret := []T{}

	for _, x := range list {
		ret = append(ret, x...)
	}

	return ret
}

// Each runs the given function on each item in the list.
func Each[T any](list []T, fn func(T)) {
	EachSuccess[T](list, func(x T) error {
		fn(x)
		return nil
	})
}

// EachSuccess runs the given function on each item in the list, and returns the first error encountered.
func EachSuccess[T any](list []T, fn func(T) error) error {
	if len(list) == 0 {
		return nil
	}

	for _, x := range list {
		if err := fn(x); err != nil {
			return err
		}
	}
	return nil
}

// Unique returns a list of unique strings
func Unique[T comparable](strs []T) []T {
	if len(strs) == 0 || len(strs) == 1 {
		return strs
	}

	var ret []T
	var mt = map[T]bool{}

	for _, str := range strs {
		if _, found := mt[str]; !found {
			ret = append(ret, str)
			mt[str] = true
		}
	}

	return ret
}

func Coalesce[T any](v1 T, v2 T) T {
	val1 := reflect.ValueOf(v1)

	// Check if val1 is valid (not a zero Value) and also not a zero value of its type
	if !val1.IsValid() || val1.IsZero() {
		return v2
	}
	return v1
}

func EmptyStringFilter(s string) bool    { return len(s) == 0 }
func NotEmptyStringFilter(s string) bool { return !EmptyStringFilter(s) }

//**********************************************************************************************************************
// Async Functions
//**********************************************************************************************************************

// AsyncFilter runs the given function on each item in the list, asynchronously. filtering out nil results.
func AsyncFilter[T any, R any](list []T, fn func(T) *R) []R {
	var wg sync.WaitGroup
	var lock sync.RWMutex

	ret := []R{}

	for _, x := range list {
		wg.Add(1)

		go func(x T) {
			if res := fn(x); res != nil {
				lock.Lock()
				defer lock.Unlock()

				ret = append(ret, *res)
			}
			wg.Done()
		}(x)
	}

	wg.Wait()

	return ret
}

// AsyncForEach runs the given function on each item in the list, asynchronously.
func AsyncForEach[T any](list []T, fn func(T)) {
	var wg sync.WaitGroup

	for _, x := range list {
		wg.Add(1)

		go func(x T) {
			fn(x)
			wg.Done()
		}(x)
	}

	wg.Wait()
}

func AsyncMap[T any, R any](list []T, fn func(T) R) []R {
	var wg sync.WaitGroup
	var lock sync.RWMutex

	ret := make([]R, len(list))

	for pos, x := range list {
		wg.Add(1)

		go func(x T, pos int) {
			res := fn(x)

			lock.Lock()

			ret[pos] = res

			lock.Unlock()

			wg.Done()
		}(x, pos)
	}

	wg.Wait()

	return ret
}
