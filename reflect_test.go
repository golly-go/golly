package golly

import "testing"

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
