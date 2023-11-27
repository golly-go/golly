package golly

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		fn       func(int) string
		expected []string
	}{
		{
			name:     "Map to string",
			input:    []int{1, 2, 3},
			fn:       func(n int) string { return fmt.Sprintf("num%d", n) },
			expected: []string{"num1", "num2", "num3"},
		},
		{
			name:     "Map empty",
			input:    []int{},
			fn:       func(n int) string { return fmt.Sprintf("num%d", n) },
			expected: []string{},
		},

		{
			name:     "Map 1",
			input:    []int{1},
			fn:       func(n int) string { return fmt.Sprintf("num%d", n) },
			expected: []string{"num1"},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Map(tc.input, tc.fn)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestMapWithIndex(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		fn       func(int, int) string
		expected []string
	}{
		{
			name:     "Map with index",
			input:    []int{10, 20, 30},
			fn:       func(n int, i int) string { return fmt.Sprintf("%d:%d", i, n) },
			expected: []string{"0:10", "1:20", "2:30"},
		},
		{
			name:     "Map empty",
			input:    []int{},
			fn:       func(n int, i int) string { return "" },
			expected: []string{},
		},

		{
			name:     "Map 1",
			input:    []int{1},
			fn:       func(n int, i int) string { return fmt.Sprintf("%d:%d", i, n) },
			expected: []string{"0:1"},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MapWithIndex(tc.input, tc.fn)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestFilter(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		fn       func(int) bool
		expected []int
	}{
		{
			name:     "Filter even numbers",
			input:    []int{1, 2, 3, 4, 5},
			fn:       func(n int) bool { return n%2 == 0 },
			expected: []int{2, 4},
		},
		{
			name:     "Filter negative numbers",
			input:    []int{-1, 2, -3, 4, -5},
			fn:       func(n int) bool { return n < 0 },
			expected: []int{-1, -3, -5},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.input, tc.fn)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestCompact(t *testing.T) {
	testCases := []struct {
		name     string
		input    []*int
		expected []*int
	}{
		{
			name:     "Remove nil elements",
			input:    []*int{func() *int { v := 1; return &v }(), nil, func() *int { v := 2; return &v }()},
			expected: []*int{func() *int { v := 1; return &v }(), func() *int { v := 2; return &v }()},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Compact(tc.input)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestFind(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		fn       func(int) bool
		expected *int
	}{
		{
			name:     "Find first even number",
			input:    []int{1, 3, 4, 6},
			fn:       func(n int) bool { return n%2 == 0 },
			expected: func() *int { v := 4; return &v }(),
		},
		{
			name:     "Find non-existent",
			input:    []int{1, 3, 5},
			fn:       func(n int) bool { return n%2 == 0 },
			expected: nil,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Find(tc.input, tc.fn)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestFlatten(t *testing.T) {
	testCases := []struct {
		name     string
		input    [][]int
		expected []int
	}{
		{
			name:     "Flatten nested slices",
			input:    [][]int{{1, 2}, {3, 4}, {5}},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "Flatten empty slices",
			input:    [][]int{{}, {}, {}},
			expected: []int{},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Flatten(tc.input)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestEachSuccess(t *testing.T) {
	testCases := []struct {
		name          string
		input         []int
		functionError bool // Indicates whether the function should return an error
		expectError   bool // Indicates whether we expect an error from EachSuccess
		calls         int  // Indicates how many times the function should be called
	}{
		{
			name:          "No errors",
			input:         []int{1, 2, 3},
			functionError: false,
			expectError:   false,
			calls:         3,
		},
		{
			name:          "Error on second element",
			input:         []int{1, 2, 3},
			functionError: true,
			expectError:   true,
			calls:         1,
		},
		{
			name:          "Empty slice",
			input:         []int{},
			functionError: false,
			expectError:   false,
			calls:         0,
		},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result []int

			err := EachSuccess(tc.input, func(x int) error {
				if tc.functionError && x == 2 {
					return assert.AnError
				}

				result = append(result, x+1)
				return nil
			})

			assert.Len(t, result, tc.calls, tc.name)

			if tc.expectError {
				assert.Error(t, err, tc.name)
			} else {
				assert.NoError(t, err, tc.name)
			}
		})
	}
}

func TestEach(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "Increment each element",
			input:    []int{1, 2, 3},
			expected: []int{2, 3, 4},
		},
		{
			name:     "Empty slice",
			input:    []int{},
			expected: []int{},
		},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result []int
			Each(tc.input, func(x int) {
				result = append(result, x+1)
			})
			assert.ElementsMatch(t, tc.expected, result, tc.name)
		})
	}
}

func TestUnique(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Unique strings",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "All unique strings",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "No strings",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "No strings",
			input:    []string{"a"},
			expected: []string{"a"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Unique(tc.input)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestCoalesce(t *testing.T) {
	testCases := []struct {
		name     string
		val1     interface{}
		val2     interface{}
		expected interface{}
	}{
		{
			name:     "First non-zero string",
			val1:     "",
			val2:     "default",
			expected: "default",
		},
		{
			name:     "First non-zero int",
			val1:     0,
			val2:     42,
			expected: 42,
		},
		{
			name:     "Both non-zero",
			val1:     "value",
			val2:     "default",
			expected: "value",
		},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Coalesce(tc.val1, tc.val2)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestEmptyStringFilter(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "Non-empty string",
			input:    "hello",
			expected: false,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := EmptyStringFilter(tc.input)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestNotEmptyStringFilter(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Non-empty string",
			input:    "hello",
			expected: true,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NotEmptyStringFilter(tc.input)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestAsyncForEach(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "Increment each element",
			input:    []int{1, 2, 3},
			expected: []int{2, 3, 4},
		},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var lock sync.Mutex

			result := []int{}

			AsyncForEach(tc.input, func(x int) {
				lock.Lock()
				defer lock.Unlock()

				result = append(result, x+1)
			})

			assert.ElementsMatch(t, tc.expected, result, tc.name)
		})
	}
}

func TestAsyncFilter(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "Filter positive numbers",
			input:    []int{1, -2, 3, -4, 5},
			expected: []int{1, 3, 5},
		},
		{
			name:     "Filter all return nil",
			input:    []int{-1, -2, -3},
			expected: []int{},
		},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AsyncFilter(tc.input, func(x int) *int {
				if x > 0 {
					return &x
				}
				return nil
			})
			assert.ElementsMatch(t, tc.expected, result, tc.name)
		})
	}
}

func TestAsyncMap(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		fn       func(int) string
		expected []string
	}{
		{
			name:     "Map to string",
			input:    []int{1, 2, 3},
			fn:       func(n int) string { return fmt.Sprintf("num%d", n) },
			expected: []string{"num1", "num2", "num3"},
		},
		{
			name:     "Square numbers",
			input:    []int{1, 2, 3, 4},
			fn:       func(n int) string { return fmt.Sprintf("%d", n*n) },
			expected: []string{"1", "4", "9", "16"},
		},
		{
			name:     "Boolean check for even numbers",
			input:    []int{1, 2, 3, 4, 5},
			fn:       func(n int) string { return fmt.Sprintf("%v", n%2 == 0) },
			expected: []string{"false", "true", "false", "true", "false"},
		},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AsyncMap(tc.input, tc.fn)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}
