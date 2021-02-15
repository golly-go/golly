package utils

import (
	"crypto/sha512"
	"fmt"

	"github.com/google/uuid"
)

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
