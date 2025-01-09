package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	numbers := []int{1, 2, 3, 4, 5}
	result, found := Find(numbers, func(n int) bool { return n > 3 })
	assert.True(t, found)
	assert.Equal(t, 4, result)

	result, found = Find(numbers, func(n int) bool { return n > 10 })
	assert.False(t, found)
}

func TestContains(t *testing.T) {
	t.Run("Element present in slice", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5}
		assert.True(t, Contains(numbers, 3))
	})

	t.Run("Element not present in slice", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5}
		assert.False(t, Contains(numbers, 10))
	})

	t.Run("Empty slice", func(t *testing.T) {
		numbers := []int{}
		assert.False(t, Contains(numbers, 1))
	})

	t.Run("Slice with duplicate elements", func(t *testing.T) {
		numbers := []int{1, 2, 3, 3, 5}
		assert.True(t, Contains(numbers, 3))
	})

	t.Run("Slice of strings", func(t *testing.T) {
		strings := []string{"apple", "banana", "cherry"}
		assert.True(t, Contains(strings, "banana"))
		assert.False(t, Contains(strings, "grape"))
	})
}

func BenchmarkFind(b *testing.B) {
	numbers := make([]int, 1000)
	for i := range numbers {
		numbers[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Find(numbers, func(n int) bool { return n == 999 })
	}
}

func BenchmarkContains(b *testing.B) {
	b.Run("Small slice", func(b *testing.B) {
		numbers := []int{1, 2, 3, 4, 5}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Contains(numbers, 3)
		}
	})

	b.Run("Large slice with match", func(b *testing.B) {
		numbers := make([]int, 1000)
		for i := range numbers {
			numbers[i] = i
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Contains(numbers, 999) // Match at the end
		}
	})

	b.Run("Large slice without match", func(b *testing.B) {
		numbers := make([]int, 1000)
		for i := range numbers {
			numbers[i] = i
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Contains(numbers, 1001) // No match
		}
	})
}
