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

func Tokenize(s string, delim byte) []string {
	var ret []string
	var ln = len(s)
	var start = 0

	for i := 0; i < ln; i++ {
		switch s[i] {
		case delim:
			ret = append(ret, s[start:i])

			for s[i] == delim || s[i] == ' ' || s[i]+1 == ' ' {
				i++
			}

			start = i
		default:
			if i+1 >= ln {
				ret = append(ret, s[start:])
			}
		}
	}
	return ret
}

// ASCIICompair extremely fast string comparison for simple insenstive
// checks
func ASCIICompair(str1, str2 string) bool {
	if len(str1) != len(str2) {
		return false
	}

	for i := 0; i < len(str1); i++ {
		a := str1[i]
		b := str2[i]

		if 'A' <= a && a <= 'Z' {
			a += 'a' - 'A'
		}

		if 'A' <= b && b <= 'Z' {
			b += 'a' - 'A'
		}

		if b != a {
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
