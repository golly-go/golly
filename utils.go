package golly

import "unsafe"

// unsafeBytes converts a string to a []byte without allocating.
func unsafeBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
