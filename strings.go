package golly

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrUnsupportedDataType = errors.New("unsupported data type")

	matchFirstCap = regexp.MustCompile("([A-Z]+)([A-Z][a-z])")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

type Converter func(string) string

func Convert(s []string, c Converter) []string {
	out := []string{}
	for pos := range s {
		out = append(out, c(s[pos]))
	}
	return out
}

// ASCIICompair extremely fast string comparison for simple insenstive
// checks
func ASCIICompair(str1, str2 string) bool {
	if len(str1) != len(str2) {
		return false
	}

	for i := 0; i < len(str1); i++ {
		if (str1[i]|0x20 != str2[i]|0x20) && (str1[i]|0x20 >= 'a' && str1[i]|0x20 <= 'z') {
			return false
		}
	}
	return true
}

// SnakeCase converts a camelCase or PascalCase string to snake_case.
func SnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
