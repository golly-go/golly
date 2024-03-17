package utils

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/google/uuid"
)

var (
	ErrUnsupportedDataType = errors.New("unsupported data type")

	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func SnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func IDField(model interface{}) interface{} {
	value := valueOf(model)

	if v := value.FieldByName("ID"); v.IsValid() {
		switch id := v.Interface().(type) {
		case uuid.UUID:
			return id
		case map[string]interface{}:
			return id["_id"]
		default:
			return id
		}
	}
	return ""
}

func valueOf(obj interface{}) reflect.Value {
	value := reflect.ValueOf(obj)

	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	return value
}

func CollectionName(doc interface{}) (string, error) {

	switch d := doc.(type) {
	case string:
		return d, nil
	default:
		t, e := ResolveToType(doc)

		if e == nil && !t.IsNil() {
			s := SnakeCase(GetType(t.Interface()))

			return pluralize.NewClient().Pluralize(s, 2, false), nil
		}
		return "", e
	}
}

func ResolveToType(toResolve interface{}) (reflect.Value, error) {
	value := reflect.ValueOf(toResolve)

	if value.Kind() == reflect.Ptr && value.IsNil() {
		value = reflect.New(value.Type().Elem())
	}

	modelType := reflect.Indirect(value).Type()

	if modelType.Kind() == reflect.Interface {
		modelType = reflect.Indirect(reflect.ValueOf(toResolve)).Elem().Type()
	}

	for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		if modelType.PkgPath() == "" {
			return value, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, toResolve)
		}
		return value, fmt.Errorf("%w: %s.%s", ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name())
	}

	return reflect.New(modelType), nil
}
