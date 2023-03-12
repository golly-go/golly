package utils

import "github.com/google/uuid"

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func NewUUID() string {
	uid, _ := uuid.NewUUID()
	return uid.String()
}

func CoalesceUUID(s1, s2 uuid.UUID) uuid.UUID {
	if s1 == uuid.Nil {
		return s2
	}
	return s1
}
