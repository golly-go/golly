package golly

import (
	"cmp"
	"slices"
)

// Find searches for the first element in the slice that satisfies the predicate.
// Returns the element and a boolean indicating if it was found.
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	for i := range slice {
		if predicate(slice[i]) {
			return slice[i], true
		}
	}
	return *new(T), false // Avoid explicit zero value creation
}

// Contains checks if a slice contains the specified element.
func Contains[T comparable](slice []T, element T) bool {
	for i := range slice {
		if slice[i] == element {
			return true
		}
	}
	return false
}

// Any checks if any element in the slice satisfies the given predicate function.
//
// Parameters:
//   - slice: A slice of type `T` to evaluate.
//   - predicate: A function that takes an element of type `T` and returns a boolean.
//
// Returns:
//   - true if at least one element satisfies the predicate, false otherwise.
//
// Example:
//
//	numbers := []int{1, 2, 3, 4}
//	hasEven := Any(numbers, func(n int) bool { return n%2 == 0 }) // true
func Any[T any](slice []T, predicate func(T) bool) bool {
	for i := range slice {
		if predicate(slice[i]) {
			return true
		}
	}
	return false
}

func Map[T any, R any](list []T, fn func(T) R) []R {
	ret := make([]R, 0, len(list))

	for pos := range list {
		ret = append(ret, fn(list[pos]))
	}

	return ret
}

// Filter filters elements of the input slice based on a predicate function.
//
// Parameters:
//   - list: A slice of type `T` to filter.
//   - fn: A function that takes an element of type `T` and returns a boolean.
//
// Returns:
//   - A slice of type `T` containing only the elements that satisfy the predicate.
//
// Example:
//   numbers := []int{1, 2, 3, 4}
//   evens := Filter(numbers, func(n int) bool { return n%2 == 0 }) // []int{2, 4}

func Filter[T any](in []T, keep func(T) bool) []T {
	out := make([]T, len(in))
	copy(out, in) // 1 alloc, fast memcpy

	w := 0
	for _, v := range out {
		if keep(v) {
			out[w] = v
			w++
		}
	}
	return out[:w]
}

// MapWithIndex applies a transformation function to each element of the slice, passing its index.
//
// Parameters:
//   - list: A slice of type `T` to transform.
//   - fn: A function that takes an element of type `T` and its index, and returns a value of type `R`.
//
// Returns:
//   - A slice of type `R` containing the transformed elements.
//
// Example:
//   numbers := []int{10, 20, 30}
//   result := MapWithIndex(numbers, func(value int, index int) string {
//     return fmt.Sprintf("%d: %d", index, value)
//   })
//   // result: []string{"0: 10", "1: 20", "2: 30"}

func MapWithIndex[T any, R any](list []T, fn func(T, int) R) []R {
	ret := make([]R, 0, len(list))

	for pos := range list {
		ret = append(ret, fn(list[pos], pos))
	}

	return ret
}

// Unique removes duplicate elements from a slice in-place.
//
// ⚠️ WARNING: This function SORTS the input slice and DOES NOT preserve original order!
// The returned slice will be sorted in ascending order with duplicates removed.
// If you need to preserve order, use a map-based approach instead.
//
// Example:
//
//	in := []int{3, 1, 2, 1, 3}
//	result := Unique(in)  // Returns: [1, 2, 3] (sorted, unique)
//
// Performance: O(n log n) time, 0 allocations (sorts in-place).
func Unique[T cmp.Ordered](in []T) []T {
	if len(in) < 2 {
		return in
	}
	slices.Sort(in) // In-place, O(n log n)
	w := 1
	for i := 1; i < len(in); i++ {
		if in[i] != in[i-1] {
			in[w] = in[i]
			w++
		}
	}
	return in[:w]
}

// func Unique[T comparable](in []T) []T {
// 	seen := make(map[T]struct{}, len(in))

// 	w := 0
// 	for _, v := range in {
// 		if _, ok := seen[v]; ok {
// 			continue
// 		}
// 		seen[v] = struct{}{}
// 		in[w] = v
// 		w++
// 	}
// 	return in[:w]
// }
