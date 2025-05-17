package golly

import (
	"reflect"
	"testing"
)

// Sample types for testing
type SampleStruct struct{}
type AnotherStruct struct{}

func TestTypeNoPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"Struct without pointer", SampleStruct{}, "golly.SampleStruct"},
		{"Struct with pointer", &SampleStruct{}, "golly.SampleStruct"},
		{"String type", "hello", "string"},
		{"Int type", 123, "int"},
		{"Pointer to int", new(int), "int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TypeNoPtr(tt.input)
			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestInfNameNoPackage(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"Struct without pointer", SampleStruct{}, "SampleStruct"},
		{"Struct with pointer", &SampleStruct{}, "SampleStruct"},
		{"Anonymous struct", struct{}{}, "struct {}"},
		{"Int type", 123, "int"},
		{"Pointer to int", new(int), "int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InfNameNoPackage(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func SampleHandler() {}

func (s SampleStruct) MethodHandler() {}

var anonymousFunc = func() {}

// Interface sample
type HandlerInterface interface {
	Handle()
}

type InterfaceImpl struct{}

func (i InterfaceImpl) Handle() {}

func TestFuncPath(t *testing.T) {
	tests := []struct {
		name             string
		input            interface{}
		expectedContains string
	}{
		{
			name:             "Regular function",
			input:            SampleHandler,
			expectedContains: "SampleHandler",
		},
		{
			name:             "Anonymous function",
			input:            anonymousFunc,
			expectedContains: "github.com/golly-go/golly.init.func1",
		},
		{
			name:             "Struct pointer",
			input:            &SampleStruct{},
			expectedContains: "SampleStruct",
		},
		{
			name:             "Struct value",
			input:            SampleStruct{},
			expectedContains: "SampleStruct",
		},
		{
			name:             "Struct method",
			input:            SampleStruct{}.MethodHandler,
			expectedContains: "MethodHandler",
		},
		{
			name:             "Interface implementation",
			input:            InterfaceImpl{},
			expectedContains: "InterfaceImpl",
		},
		{
			name:             "Nil pointer",
			input:            (*SampleStruct)(nil),
			expectedContains: "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FuncPath(tt.input)
			if !contains(result, tt.expectedContains) {
				t.Errorf("expected result to contain '%s', got '%s'", tt.expectedContains, result)
			}
		})
	}
}

// Helper to check substring presence
func contains(full, sub string) bool {
	return reflect.ValueOf(full).IsValid() && len(full) >= len(sub) && reflect.ValueOf(full).String() != "" && (len(sub) == 0 || (len(sub) > 0 && len(full) > 0 && reflect.ValueOf(full).String() != "" && (reflect.DeepEqual(full, sub) || len(sub) <= len(full) && (string(full) == sub || string(full[0:len(sub)]) == sub || len(full) > len(sub) && contains(full[1:], sub)))))
}
