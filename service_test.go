package golly

import (
	"reflect"
	"testing"
)

func TestServiceMap(t *testing.T) {
	tests := []struct {
		name     string
		services []Service
		expected map[string]Service
	}{
		{name: "nil", services: nil, expected: map[string]Service{}},
		{name: "empty", services: []Service{}, expected: map[string]Service{}},
		{name: "one service", services: []Service{&WebService{}}, expected: map[string]Service{
			"web": &WebService{},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := serviceMap(test.services)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}
