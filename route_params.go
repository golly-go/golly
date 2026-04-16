package golly

import (
	"reflect"
	"strings"
)

// ParamSource indicates where a parameter originates.
type ParamSource string

const (
	ParamSourceInput  ParamSource = "input"
	ParamSourceQuery  ParamSource = "query"
	ParamSourceOutput ParamSource = "output"
	ParamSourcePath   ParamSource = "path"
)

// RouteParam describes a single declared parameter on a route.
type RouteParam struct {
	Name     string
	Type     string
	Required bool
	Source   ParamSource
}

type RouteParamSet []RouteParam

// RouteDoc holds routing documentation metadata.
type RouteDoc struct {
	description string
	params      RouteParamSet
}

// Input sets the input schema by reflecting over the provided struct instance.
func (d *RouteDoc) Input(v any) *RouteDoc {
	if d == nil {
		d = &RouteDoc{}
	}
	d.params = append(d.params, paramsFromAny(v, ParamSourceInput)...)
	return d
}

// Query sets the query schema by reflecting over the provided struct instance.
func (d *RouteDoc) Query(v any) *RouteDoc {
	if d == nil {
		d = &RouteDoc{}
	}
	d.params = append(d.params, paramsFromAny(v, ParamSourceQuery)...)
	return d
}

// Output sets the output schema by reflecting over the provided struct instance.
func (d *RouteDoc) Output(v any) *RouteDoc {
	if d == nil {
		d = &RouteDoc{}
	}
	d.params = append(d.params, paramsFromAny(v, ParamSourceOutput)...)
	return d
}

// Describe initializes a RouteDoc with a description.
func Describe(desc string) *RouteDoc { return &RouteDoc{description: desc} }

// Input is a convenience starting point for RouteDoc without a description.
func Input(v any) *RouteDoc { return (&RouteDoc{}).Input(v) }

// Query is a convenience starting point for RouteDoc without a description.
func Query(v any) *RouteDoc { return (&RouteDoc{}).Query(v) }

// Output is a convenience starting point for RouteDoc without a description.
func Output(v any) *RouteDoc { return (&RouteDoc{}).Output(v) }

func paramsFromAny(v any, source ParamSource) RouteParamSet {
	if v == nil {
		return nil
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	return paramsFromType(t, source)
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

// formatRouteDoc formats a RouteDoc for display in the route list.
func formatRouteDoc(doc *RouteDoc) (string, string, string, string) {
	if doc == nil {
		return "-", "-", "-", ""
	}

	var inParts []string
	var outParts []string
	var queryParts []string

	if len(doc.params) > 0 {
		for _, p := range doc.params {
			modifier := "?"
			if p.Required {
				modifier = "*"
			}
			str := p.Name + ": " + p.Type + modifier

			switch p.Source {
			case ParamSourceOutput:
				outParts = append(outParts, str)
			case ParamSourceQuery:
				queryParts = append(queryParts, str)
			default:
				inParts = append(inParts, str)
			}
		}
	}

	queryStr, inStr, outStr := "-", "-", "-"

	if len(queryParts) > 0 {
		queryStr = "[" + strings.Join(queryParts, ", ") + "]"
	}
	if len(inParts) > 0 {
		inStr = "[" + strings.Join(inParts, ", ") + "]"
	}
	if len(outParts) > 0 {
		outStr = "[" + strings.Join(outParts, ", ") + "]"
	}

	descStr := ""
	if doc.description != "" {
		descStr = "\"" + doc.description + "\""
	}
	return queryStr, inStr, outStr, descStr
}
