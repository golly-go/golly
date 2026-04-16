package golly

import (
	"reflect"
	"strings"
)

// ParamSource indicates where a parameter originates.
type ParamSource string

const (
	ParamSourceBody  ParamSource = "body"
	ParamSourceQuery ParamSource = "query"
	ParamSourcePath  ParamSource = "path"
)

// RouteParam describes a single declared parameter on a route.
type RouteParam struct {
	Name     string
	Type     string
	Required bool
	Source   ParamSource
}

// RouteParamSet is a slice of RouteParam attached to a route registration.
// Pass it as the last argument to Post/Put/Patch/etc via Params[T]().
type RouteParamSet []RouteParam

// Params inspects T via reflection and returns a RouteParamSet describing
// its exported fields. Field names are resolved from json tags; required
// status is resolved from required:"true" or validate:"required" tags.
//
// Usage:
//
//	r.Post("/create", c.Create, golly.Params[CreateArgs]())
func Params[T any]() RouteParamSet {
	var zero T

	t := reflect.TypeOf(zero)
	if t == nil {
		return nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	return paramsFromType(t, ParamSourceBody)
}

func paramsFromType(t reflect.Type, source ParamSource) RouteParamSet {
	params := make(RouteParamSet, 0, t.NumField())

	for i := range t.NumField() {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		// Resolve name from json tag, falling back to lowercase field name.
		name := field.Tag.Get("json")
		if name == "" {
			name = strings.ToLower(field.Name)
		} else {
			name = strings.Split(name, ",")[0]
			if name == "-" {
				continue // explicitly excluded
			}
		}

		params = append(params, RouteParam{
			Name:     name,
			Type:     field.Type.String(),
			Required: isRouteParamRequired(field),
			Source:   source,
		})
	}

	return params
}

// isRouteParamRequired checks required:"true" and validate:"required" tags.
func isRouteParamRequired(field reflect.StructField) bool {
	if field.Tag.Get("required") == "true" {
		return true
	}

	for _, part := range strings.Split(field.Tag.Get("validate"), ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}

	return false
}

// formatRouteParams formats a RouteParamSet for display in the route list.
// Required params are suffixed with *, optional with ?.
func formatRouteParams(params RouteParamSet) string {
	if len(params) == 0 {
		return ""
	}

	parts := make([]string, 0, len(params))
	for _, p := range params {
		if p.Required {
			parts = append(parts, p.Name+": "+p.Type+"*")
		} else {
			parts = append(parts, p.Name+": "+p.Type+"?")
		}
	}

	return " [" + strings.Join(parts, ", ") + "]"
}
