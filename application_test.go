package golly

import (
	"context"
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
	for i := range numGoroutines {
		go func(id int) {
			service := &mockService{name: "ConcurrentService"}
			err := app.RegisterService(service)
			errors <- err
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for range numGoroutines {
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

// TestServices tests the Services() method
func TestServices(t *testing.T) {
	tests := []struct {
		name            string
		services        []Service
		expectedCount   int
		expectedService Service
	}{
		{
			name:          "Empty services",
			services:      nil,
			expectedCount: 0,
		},
		{
			name:          "Single service",
			services:      []Service{&mockService{name: "Service1"}},
			expectedCount: 1,
		},
		{
			name:          "Multiple services",
			services:      []Service{&mockService{name: "Service1"}, &mockService{name: "Service2"}, &mockService{name: "Service3"}},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(Options{Services: tt.services})

			services := app.Services()

			assert.Len(t, services, tt.expectedCount)
			// Verify all services are present
			serviceMap := make(map[string]Service)
			for _, s := range services {
				serviceMap[getServiceName(s)] = s
			}

			for _, expectedSvc := range tt.services {
				name := getServiceName(expectedSvc)
				assert.Contains(t, serviceMap, name)
				assert.Equal(t, expectedSvc, serviceMap[name])
			}
		})
	}
}

// TestChangeState tests the changeState method
func TestChangeState(t *testing.T) {
	tests := []struct {
		name           string
		initialState   ApplicationState
		newState       ApplicationState
		shouldChange   bool
		shouldDispatch bool
	}{
		{
			name:           "Change from starting to running",
			initialState:   StateStarting,
			newState:       StateRunning,
			shouldChange:   true,
			shouldDispatch: true,
		},
		{
			name:           "Change from initialized to running",
			initialState:   StateInitialized,
			newState:       StateRunning,
			shouldChange:   true,
			shouldDispatch: true,
		},
		{
			name:           "Cannot change from shutdown",
			initialState:   StateShutdown,
			newState:       StateRunning,
			shouldChange:   false,
			shouldDispatch: false,
		},
		{
			name:           "Cannot change from errored",
			initialState:   StateErrored,
			newState:       StateRunning,
			shouldChange:   false,
			shouldDispatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(Options{})
			app.state = tt.initialState

			eventDispatched := false
			var dispatchedState ApplicationState

			// Register event handler to verify dispatch
			app.On(EventStateChanged, func(ctx context.Context, data any) {
				eventDispatched = true
				if stateChanged, ok := data.(ApplicationStateChanged); ok {
					dispatchedState = stateChanged.State
				}
			})

			app.changeState(tt.newState)

			if tt.shouldChange {
				assert.Equal(t, tt.newState, app.State())
			} else {
				assert.Equal(t, tt.initialState, app.State())
			}

			if tt.shouldDispatch {
				assert.True(t, eventDispatched, "Event should have been dispatched")
				assert.Equal(t, tt.newState, dispatchedState)
			} else {
				assert.False(t, eventDispatched, "Event should not have been dispatched")
			}
		})
	}
}

// TestInitialize tests the initialize method
func TestInitialize(t *testing.T) {
	tests := []struct {
		name        string
		preboot     AppFunc
		initializer AppFunc
		expectErr   bool
	}{
		{
			name:        "No initializers",
			preboot:     nil,
			initializer: nil,
			expectErr:   false,
		},
		{
			name:        "Only preboot",
			preboot:     mockAppFunc(true),
			initializer: nil,
			expectErr:   false,
		},
		{
			name:        "Only initializer",
			preboot:     nil,
			initializer: mockAppFunc(true),
			expectErr:   false,
		},
		{
			name:        "Both preboot and initializer",
			preboot:     mockAppFunc(true),
			initializer: mockAppFunc(true),
			expectErr:   false,
		},
		{
			name:        "Preboot fails",
			preboot:     mockAppFunc(false),
			initializer: mockAppFunc(true),
			expectErr:   true,
		},
		{
			name:        "Initializer fails",
			preboot:     mockAppFunc(true),
			initializer: mockAppFunc(false),
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(Options{
				Preboot:     tt.preboot,
				Initializer: tt.initializer,
			})

			err := app.initialize()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestOn tests the On method for registering event handlers
func TestOn(t *testing.T) {
	app := NewApplication(Options{})

	callCount := 0
	handler := func(ctx context.Context, data any) {
		callCount++
	}

	// Create a test event type
	type TestEvent struct {
		Name string
	}
	eventName := TypeNoPtr(TestEvent{}).String()

	// Register handler using the event type name
	app.On(eventName, handler)

	// Dispatch event
	app.Events().Dispatch(NewContext(context.Background()), TestEvent{Name: "test"})

	// Verify handler was called
	assert.Equal(t, 1, callCount)

	// Register multiple handlers for same event
	callCount2 := 0
	handler2 := func(ctx context.Context, data any) {
		callCount2++
	}

	app.On(eventName, handler2)

	// Dispatch again
	app.Events().Dispatch(NewContext(context.Background()), TestEvent{Name: "test"})

	// Both handlers should be called
	assert.Equal(t, 2, callCount)
	assert.Equal(t, 1, callCount2)
}

// TestOff tests the Off method for unregistering event handlers
func TestOff(t *testing.T) {
	app := NewApplication(Options{})

	callCount := 0
	handler := func(ctx context.Context, data any) {
		callCount++
	}

	// Create a test event type
	type TestEvent struct {
		Name string
	}
	eventName := TypeNoPtr(TestEvent{}).String()

	// Register handler
	app.On(eventName, handler)

	// Dispatch event
	app.Events().Dispatch(NewContext(context.Background()), TestEvent{Name: "test"})
	assert.Equal(t, 1, callCount)

	// Unregister handler
	app.Off(eventName, handler)

	// Dispatch again
	app.Events().Dispatch(NewContext(context.Background()), TestEvent{Name: "test"})

	// Handler should not be called again
	assert.Equal(t, 1, callCount, "Handler should not be called after unregistering")
}

// TestShutdown tests the Shutdown method
func TestShutdown(t *testing.T) {
	tests := []struct {
		name           string
		initialState   ApplicationState
		shouldShutdown bool
		shouldDispatch bool
	}{
		{
			name:           "Shutdown from running state",
			initialState:   StateRunning,
			shouldShutdown: true,
			shouldDispatch: true,
		},
		{
			name:           "Shutdown from initialized state",
			initialState:   StateInitialized,
			shouldShutdown: true,
			shouldDispatch: true,
		},
		{
			name:           "Shutdown when already shutdown",
			initialState:   StateShutdown,
			shouldShutdown: false,
			shouldDispatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(Options{})
			app.state = tt.initialState

			shutdownDispatched := false
			stateChangedDispatched := false

			// Register event handlers
			app.On(EventShutdown, func(ctx context.Context, data any) {
				shutdownDispatched = true
			})

			app.On(EventStateChanged, func(ctx context.Context, data any) {
				if stateChanged, ok := data.(ApplicationStateChanged); ok {
					if stateChanged.State == StateShutdown {
						stateChangedDispatched = true
					}
				}
			})

			app.Shutdown()

			if tt.shouldShutdown {
				assert.Equal(t, StateShutdown, app.State())
			} else {
				assert.Equal(t, tt.initialState, app.State())
			}

			if tt.shouldDispatch {
				assert.True(t, shutdownDispatched, "Shutdown event should have been dispatched")
				assert.True(t, stateChangedDispatched, "StateChanged event should have been dispatched")
			} else {
				assert.False(t, shutdownDispatched, "Shutdown event should not have been dispatched")
			}
		})
	}
}

// TestShutdownMultipleTimes tests that calling Shutdown multiple times is safe
func TestShutdownMultipleTimes(t *testing.T) {
	app := NewApplication(Options{})
	app.state = StateRunning

	shutdownCount := 0
	app.On(EventShutdown, func(ctx context.Context, data any) {
		shutdownCount++
	})

	// Call shutdown multiple times
	app.Shutdown()
	app.Shutdown()
	app.Shutdown()

	// Should only dispatch once
	assert.Equal(t, 1, shutdownCount)
	assert.Equal(t, StateShutdown, app.State())
}
