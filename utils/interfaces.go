package utils

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"strings"
)

func GetBytes(key interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return []byte{}
	}
	return buf.Bytes()
}

func GetType(myvar interface{}) string {
	return GetRawType(myvar).Name()
}

func GetTypeWithPackage(myvar interface{}) string {
	return GetRawType(myvar).String()
}

func GetRawType(myvar interface{}) reflect.Type {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// GetTypeName of given struct
func GetTypeName(source interface{}) (reflect.Type, string) {
	rawType := GetRawType(source)

	name := rawType.String()
	if idx := strings.Index(name, "."); idx >= 0 {
		name = name[idx+1:]
	}

	return rawType, name
}
