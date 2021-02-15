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
