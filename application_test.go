package golly

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mocks and Helpers
func mockAppFunc(success bool) AppFunc {
	return func(app *Application) error {
		if !success {
			return errors.New("mock error")
		}
		return nil
	}
}

// type testService struct{}

// func (*testService) Initialize(*Application) error { return nil }
// func (*testService) Stop() error                   { return nil }
// func (*testService) Start() error                  { return nil }
// func (*testService) IsRunning() bool               { return false }

// Tests for InitializerChain
func TestInitializerChain(t *testing.T) {
	tests := []struct {
		name         string
		initializers []AppFunc
		expectErr    bool
	}{
		{
			name:         "All initializers succeed",
			initializers: []AppFunc{mockAppFunc(true), mockAppFunc(true)},
			expectErr:    false,
		},
		{
			name:         "One initializer fails",
			initializers: []AppFunc{mockAppFunc(true), mockAppFunc(false), mockAppFunc(true)},
			expectErr:    true,
		},
		{
			name:         "Empty initializer chain",
			initializers: []AppFunc{},
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := AppFuncChain(tt.initializers...)
			app := &Application{}

			err := chain(app)
			if tt.expectErr {
				assert.Error(t, err, "Expected an error but got nil")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
			}
		})
	}
}

func TestRunAppFuncs(t *testing.T) {
	tests := []struct {
		name      string
		funcs     []AppFunc
		expectErr bool
	}{
		{
			name:      "All functions succeed",
			funcs:     []AppFunc{mockAppFunc(true), mockAppFunc(true)},
			expectErr: false,
		},
		{
			name:      "One function fails",
			funcs:     []AppFunc{mockAppFunc(true), mockAppFunc(false), mockAppFunc(true)},
			expectErr: true,
		},
		{
			name:      "Empty function list",
			funcs:     []AppFunc{},
			expectErr: false,
		},
		{
			name:      "All functions fail",
			funcs:     []AppFunc{mockAppFunc(false), mockAppFunc(false)},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &Application{}

			err := runAppFuncs(app, tt.funcs)
			if tt.expectErr {
				assert.Error(t, err, "Expected an error but got nil")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
			}
		})
	}
}

// Tests for NewApplication
func TestNewApplication(t *testing.T) {
	tests := []struct {
		name                  string
		options               Options
		expectedName          string
		expectedLength        int
		expectedServiceLength int
		hasInitializer        bool
	}{
		{
			name:                  "Empty options",
			options:               Options{},
			expectedName:          "",
			expectedLength:        0,
			expectedServiceLength: 0,
			hasInitializer:        false,
		},
		{
			name: "With initializers",
			options: Options{
				Name:        "TestApp",
				Initializer: mockAppFunc(true),
			},
			expectedName:          "TestApp",
			expectedLength:        1,
			expectedServiceLength: 0,
			hasInitializer:        true,
		},
		{
			name: "With services",
			options: Options{
				Name:     "ServiceApp",
				Services: []Service{&mockService{}},
			},
			expectedName:          "ServiceApp",
			expectedLength:        1,
			expectedServiceLength: 1,
			hasInitializer:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(tt.options)

			assert.Equal(t, tt.expectedName, app.Name, "Application name mismatch")
			if tt.hasInitializer {
				assert.NotNil(t, app.initializer)
			} else {
				assert.Nil(t, app.initializer)
			}
			assert.Len(t, app.services, tt.expectedServiceLength, "Unexpected number of services")
		})
	}
}
