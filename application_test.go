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

// TestRegisterService tests the RegisterService method
func TestRegisterService(t *testing.T) {
	tests := []struct {
		name             string
		service          Service
		startingServices []Service
		expectErr        bool
		expectedError    error
		expectedCount    int
	}{
		{
			name:             "Register new service successfully",
			service:          &mockService{name: "TestService"},
			startingServices: nil,
			expectErr:        false,
			expectedCount:    1,
		},
		{
			name:             "Register service with duplicate name",
			startingServices: []Service{&mockService{name: "TestService"}},
			service:          &mockService{name: "TestService"},
			expectErr:        true,
			expectedError:    ErrorServiceAlreadyRegistered,
			expectedCount:    1, // Should still be 1, not incremented
		},
		{
			name:             "Register different service when one already exists",
			startingServices: []Service{&mockService{name: "Service1"}},
			service:          &mockService{name: "Service2"},
			expectErr:        false,
			expectedCount:    2,
		},
		{
			name:             "Register service without Namer interface (uses type name)",
			startingServices: []Service{&mockService{name: "Service1"}},
			service:          &mockService{}, // No name set
			expectErr:        false,
			expectedCount:    2, // Service1 + the unnamed service
		},
		{
			name:             "Register service with empty name (falls back to type name)",
			startingServices: []Service{&mockService{name: "Service1"}},
			service:          &mockService{name: ""}, // Empty name
			expectErr:        false,
			expectedCount:    2, // Service1 + the service with empty name (uses type name)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(Options{Services: tt.startingServices})

			err := app.RegisterService(tt.service)

			if tt.expectErr {
				assert.Error(t, err)

				if tt.expectedError != nil {
					assert.Equal(t, tt.expectedError, err)
				}

			} else {
				assert.NoError(t, err)
			}

			assert.Len(t, app.services, tt.expectedCount)
		})
	}
}

// TestRegisterServiceMultipleServices tests registering multiple different services
func TestRegisterServiceMultipleServices(t *testing.T) {
	app := &Application{
		services: make(map[string]Service),
	}

	service1 := &mockService{name: "Service1"}
	service2 := &mockService{name: "Service2"}
	service3 := &mockService{name: "Service3"}

	// Register first service
	err := app.RegisterService(service1)
	assert.NoError(t, err)
	assert.Len(t, app.services, 1)

	// Register second service
	err = app.RegisterService(service2)
	assert.NoError(t, err)
	assert.Len(t, app.services, 2)

	// Register third service
	err = app.RegisterService(service3)
	assert.NoError(t, err)
	assert.Len(t, app.services, 3)

	// Verify all services are registered
	assert.Equal(t, service1, app.services["Service1"])
	assert.Equal(t, service2, app.services["Service2"])
	assert.Equal(t, service3, app.services["Service3"])
}

// TestRegisterServiceDuplicatePreventsRegistration tests that duplicate registration is prevented
func TestRegisterServiceDuplicatePreventsRegistration(t *testing.T) {
	app := &Application{
		services: make(map[string]Service),
	}

	originalService := &mockService{name: "DuplicateService"}
	duplicateService := &mockService{name: "DuplicateService"}

	// Register original service
	err := app.RegisterService(originalService)
	assert.NoError(t, err)
	assert.Len(t, app.services, 1)
	assert.Equal(t, originalService, app.services["DuplicateService"])

	// Try to register duplicate
	err = app.RegisterService(duplicateService)
	assert.Error(t, err)
	assert.Equal(t, ErrorServiceAlreadyRegistered, err)
	assert.Len(t, app.services, 1)
	// Original service should still be there
	assert.Equal(t, originalService, app.services["DuplicateService"])
}

// TestRegisterServiceWithTypeName tests services that don't implement Namer
func TestRegisterServiceWithTypeName(t *testing.T) {
	// Create a service type that doesn't implement Namer
	type unnamedService struct {
		mockService
	}

	app := &Application{
		services: make(map[string]Service),
	}

	service := &unnamedService{}

	err := app.RegisterService(service)
	assert.NoError(t, err)
	assert.Len(t, app.services, 1)

	// Service should be registered with its type name
	serviceName := TypeNoPtr(service).String()
	assert.Contains(t, app.services, serviceName)
	assert.Equal(t, service, app.services[serviceName])
}

// TestRegisterServiceConcurrent tests thread safety of RegisterService
func TestRegisterServiceConcurrent(t *testing.T) {
	app := &Application{
		services: make(map[string]Service),
	}

	const numGoroutines = 100
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines trying to register services with same name
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			service := &mockService{name: "ConcurrentService"}
			err := app.RegisterService(service)
			errors <- err
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		if err == nil {
			successCount++
		} else {
			errorCount++
			assert.Equal(t, ErrorServiceAlreadyRegistered, err)
		}
	}

	// Only one should succeed, rest should get duplicate error
	assert.Equal(t, 1, successCount, "Only one registration should succeed")
	assert.Equal(t, numGoroutines-1, errorCount, "All other registrations should fail")
	assert.Len(t, app.services, 1, "Only one service should be registered")
}

// TestRegisterServiceWithDifferentServiceTypes tests registering different service types
func TestRegisterServiceWithDifferentServiceTypes(t *testing.T) {
	app := &Application{
		services: make(map[string]Service),
	}

	// Register services with different names
	service1 := &mockService{name: "DatabaseService"}
	service2 := &mockService{name: "CacheService"}
	service3 := &mockService{name: "QueueService"}

	err1 := app.RegisterService(service1)
	err2 := app.RegisterService(service2)
	err3 := app.RegisterService(service3)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	assert.Len(t, app.services, 3)
	assert.Equal(t, service1, app.services["DatabaseService"])
	assert.Equal(t, service2, app.services["CacheService"])
	assert.Equal(t, service3, app.services["QueueService"])
}
