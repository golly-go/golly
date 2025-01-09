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
// Contains checks if a slice contains the specified element.
func Contains[T comparable](slice []T, element T) bool {
	for i := 0; i < len(slice); i++ {
		if slice[i] == element {
			return true
		}
	}
	return false
}

func Has[T any](slice []T, predicate func(T) bool) bool {
	for i := 0; i < len(slice); i++ {
		if predicate(slice[i]) {
			return true
		}
	}
	return false
}
