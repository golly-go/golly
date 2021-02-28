package utils

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func NewUUID() string {
	return NewUUIDV5().String()
}

func NewUUIDV5() uuid.UUID {
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))

	c := 122
	b := make([]byte, c)
	entropy.Read(b)

	return uuid.NewSHA1(uuid.NameSpaceOID, b)

}
