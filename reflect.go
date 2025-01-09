package golly

import (
	"reflect"
	"strings"
)

// TypeNoPtr returns the underlying reflect.Type of the provided variable,
// stripping away pointer indirection if present.
// For example, it returns "mypackage.Something" instead of "*mypackage.Something".
// This is useful in event engines or frameworks where pointer notation
// is not necessary and may introduce unwanted complexity.
func TypeNoPtr(myvar interface{}) reflect.Type {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// InfNameNoPackage returns the name of a struct type without the package path.
// For instance, given "mypackage.MyStruct", it will return "MyStruct".
// This simplifies the output for logging, serialization, or routing.
func InfNameNoPackage(source interface{}) string {
	rawType := TypeNoPtr(source)

	name := rawType.String()
	if idx := strings.Index(name, "."); idx >= 0 {
		return name[idx+1:]
	}

	return name
}
