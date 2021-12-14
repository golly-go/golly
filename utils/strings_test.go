package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	var examples = []struct {
		example string
		result  []string
	}{
		{"1234", []string{"1234"}},
		{"1234,5678", []string{"1234", "5678"}},
		{"1234, 5678", []string{"1234", "5678"}},
		{"1234,5678, 9123", []string{"1234", "5678", "9123"}},
	}

	for _, example := range examples {
		assert.Equal(t, Tokenize(example.example, ','), example.result)
	}
}

func TestCompairStrings(t *testing.T) {
	var examples = []struct {
		str1   string
		str2   string
		result bool
	}{
		{"abcdef", "abcdef", true},
		{"ABCDEF", "abcdef", true},
		{"ABCDEF", "ABCDEF", true},
		{"Abcdef", "Abcdef", true},
		{"AbCdEF", "aBcDeF", true},
		{"abc def", "abc def", true},
		{"abc dEf", "abc def", true},
		{"ABC def", "abc DEF", true},
		{"a", "b", false},
		{"abc", "be", false},
	}

	for _, example := range examples {
		assert.Equal(t, example.result, Compair(example.str1, example.str2), fmt.Sprintf("expected %s == %s", example.str1, example.str2))
	}
}
