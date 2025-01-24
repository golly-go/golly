package golly

// Find searches for the first element in the slice that satisfies the predicate.
// Returns the element and a boolean indicating if it was found.
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	for i := 0; i < len(slice); i++ {
		if predicate(slice[i]) {
			return slice[i], true
		}
	}
	return *new(T), false // Avoid explicit zero value creation
}

// Contains checks if a slice contains the specified element.
func Contains[T comparable](slice []T, element T) bool {
	for i := 0; i < len(slice); i++ {
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
	for i := 0; i < len(slice); i++ {
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

// func Filter[T any](list []T, fn func(T) bool) []T {
// 	ret := []T{}

// 	for i := range list {
// 		if result := fn(list[i]); result {
// 			ret = append(ret, list[i])
// 		}
// 	}

// 	return ret
// }

func Filter[T any](list []T, fn func(T) bool) []T {
	// Estimate capacity based on a fraction of the input size.
	ret := make([]T, 0, len(list)/3)

	for i := range list {
		if fn(list[i]) {
			ret = append(ret, list[i])
		}
	}

	return ret
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

// // Unique removes duplicate elements (preserving first occurrences)
// // by overwriting the input slice in place and returning a subslice.
// func Unique[T comparable](original []T) []T {

// 	// Small optimization: no work if slice length is 0 or 1
// 	if len(original) < 2 {
// 		return original
// 	}

// 	list := make([]T, len(original))

// 	// Use map[T]struct{} to save a bit of space over map[T]bool.
// 	// Pre-size to len(list) to minimize rehashing.
// 	seen := make(map[T]bool, len(list))
// 	copy(list, original)

// 	writeIdx := 0
// 	for pos := range list {
// 		if _, exists := seen[list[pos]]; !exists {
// 			seen[list[pos]] = true
// 			list[writeIdx] = list[pos]
// 			writeIdx++
// 		}
// 	}

// 	// Return only the portion of 'list' that contains unique elements
// 	return list[:writeIdx]
// }

func Unique[T comparable](original []T) []T {
	if len(original) < 2 {
		// Return a copy if you want truly non-destructive
		return append([]T(nil), original...)
	}
	seen := make(map[T]struct{}, len(original))
	result := make([]T, 0, len(original)) // grows by appending

	for _, val := range original {
		if _, found := seen[val]; !found {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}
	return result
}
