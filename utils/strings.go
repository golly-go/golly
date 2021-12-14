package utils

import (
	"crypto/sha512"
	"fmt"

	"github.com/google/uuid"
)

const toLower = 'a' - 'A'

// RandomHex returns a random generated hex from rand package
func RandomHex(n int) (string, error) {
	uid, _ := uuid.NewRandom()

	hasher := sha512.New()
	hasher.Write([]byte(uid.String()))

	x := fmt.Sprintf("%x", hasher.Sum(nil))

	return x[0:n], nil
}

// StringSliceContains - keep it dry look for a string in a slice of strings
func StringSliceContains(slice []string, s string) bool {
	for _, str := range slice {
		if str == s {
			return true
		}
	}
	return false
}

type Converter func(string) string

func Convert(s []string, c Converter) []string {
	out := []string{}
	for _, i := range s {
		out = append(out, c(i))
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

func Compair(str1, str2 string) bool {
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
