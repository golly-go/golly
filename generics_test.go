package golly

import (
	"fmt"
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

func TestAny(t *testing.T) {
	tests := []struct {
		name      string
		input     []int
		predicate func(int) bool
		expected  bool
	}{
		{"Has even number", []int{1, 2, 3}, func(x int) bool { return x%2 == 0 }, true},
		{"No even numbers", []int{1, 3, 5}, func(x int) bool { return x%2 == 0 }, false},
		{"Empty slice", []int{}, func(x int) bool { return x%2 == 0 }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Any(tt.input, tt.predicate))
		})
	}
}

func TestMap(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		fn       func(int) int
		expected []int
	}{
		{"Double values", []int{1, 2, 3}, func(x int) int { return x * 2 }, []int{2, 4, 6}},
		{"Square values", []int{1, 2, 3}, func(x int) int { return x * x }, []int{1, 4, 9}},
		{"Empty slice", []int{}, func(x int) int { return x * 2 }, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Map(tt.input, tt.fn))
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     []int
		predicate func(int) bool
		expected  []int
	}{
		{"Filter evens", []int{1, 2, 3, 4}, func(x int) bool { return x%2 == 0 }, []int{2, 4}},
		{"No matches", []int{1, 3, 5}, func(x int) bool { return x%2 == 0 }, []int{}},
		{"Empty slice", []int{}, func(x int) bool { return x%2 == 0 }, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Filter(tt.input, tt.predicate))
		})
	}
}

func TestMapWithIndex(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		fn       func(int, int) string
		expected []string
	}{
		{"Index with values", []int{10, 20}, func(x, i int) string { return fmt.Sprintf("%d:%d", i, x) }, []string{"0:10", "1:20"}},
		{"Empty slice", []int{}, func(x, i int) string { return fmt.Sprintf("%d:%d", i, x) }, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MapWithIndex(tt.input, tt.fn))
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		expect []int
	}{
		{
			name:   "empty slice",
			input:  []int{},
			expect: []int(nil),
		},
		{
			name:   "single element",
			input:  []int{1},
			expect: []int{1},
		},
		{
			name:   "multiple duplicates",
			input:  []int{1, 2, 2, 3, 1, 3},
			expect: []int{1, 2, 3},
		},
		{
			name:   "already unique",
			input:  []int{1, 2, 3, 4, 5},
			expect: []int{1, 2, 3, 4, 5},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// ***************************************************************************
// *  Benches
// ***************************************************************************

func BenchmarkAny(b *testing.B) {
	data := make([]int, 1000)
	for i := 0; i < len(data); i++ {
		data[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Any(data, func(x int) bool { return x%999 == 0 })
	}
}

func BenchmarkMap(b *testing.B) {
	data := make([]int, 1000)
	for i := 0; i < len(data); i++ {
		data[i] = i
	}

	b.Run("1000 deep slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Map(data, func(x int) int { return x * 2 })
		}
	})
}

func BenchmarkFilter(b *testing.B) {
	data := make([]int, 1000)
	for i := 0; i < len(data); i++ {
		data[i] = i
	}

	b.Run("1000 deep slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Filter(data, func(x int) bool { return x%2 == 0 })
		}
	})
}

func BenchmarkMapWithIndex(b *testing.B) {
	data := make([]int, 1000)

	b.Run("1000 deep slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = MapWithIndex(data, func(x, i int) int { return x })
		}
	})

	d2 := make([]int, 100)
	b.Run("100 small slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = MapWithIndex(d2, func(x, i int) int { return x })
		}
	})

}

func BenchmarkFind(b *testing.B) {
	numbers := make([]int, 1000)
	for i := range numbers {
		numbers[i] = i
	}

	b.Run("1000 deep slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = Find(numbers, func(n int) bool { return n == i%1000 })
		}
	})

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

func generateTestSlice(size int, hasDuplicates bool) []int {
	// If hasDuplicates, repeat some subset of numbers
	// else fill with unique values 0..size-1
	out := make([]int, size)
	for i := 0; i < size; i++ {
		if hasDuplicates {
			// This will force repeats.
			// For example, for 100 items: 0, 1, 2, ... 49, 0, 1, 2, ... 49
			out[i] = i % (size / 2)
		} else {
			out[i] = i
		}
	}
	return out
}

func BenchmarkUnique(b *testing.B) {
	benchmarks := []struct {
		name  string
		array []int
	}{
		{"SmallNoDup", generateTestSlice(10, false)},
		{"SmallDupp", generateTestSlice(10, true)},
		{"AverageSize - 1000 (NoDups)", generateTestSlice(1000, true)},
		{"AverageSize - 1000 (Dups)", generateTestSlice(1000, true)},
		{"LargeNoDup", generateTestSlice(100000, false)},
		{"LargeDup", generateTestSlice(100000, true)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name+" Map Based", func(b *testing.B) {
			// We only benchmark the Unique call itself.
			// b.ResetTimer() ensures no overhead is measured from slice generation above.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Unique(bm.array)
			}
		})
	}

}
