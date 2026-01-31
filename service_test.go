package golly

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

type namedService struct{ Service }

func (s namedService) Name() string { return "custom" }

type describedService struct{ Service }

func (s describedService) Description() string { return "custom description" }

func TestGetServiceName_Basic(t *testing.T) {
	assert.Equal(t, "web", getServiceName(&WebService{}))
	assert.Equal(t, "custom", getServiceName(namedService{}))
}

func TestGetServiceDescription(t *testing.T) {
	assert.Equal(t, "custom description", getServiceDescription(describedService{}))
	assert.Contains(t, getServiceDescription(&WebService{}), "No description for web")
}

func TestGetService(t *testing.T) {
	testApp, _ := NewTestApplication(Options{})
	web := &WebService{}
	testApp.services["web"] = web

	// From application
	assert.Equal(t, web, GetServiceFromApp[*WebService](testApp, "web"))

	// From context
	ctx := NewContext(context.Background())
	ctx.application = testApp
	assert.Equal(t, web, GetService[*WebService](ctx, "web"))

	// From WebContext
	req, _ := http.NewRequest("GET", "/", nil)
	wctx := NewTestWebContext(req, nil)
	wctx.ctx = ctx
	assert.Equal(t, web, GetService[*WebService](wctx, "web"))

	// From context.Context (using a standard context)
	assert.Equal(t, web, GetService[*WebService](context.Background(), "web"))

	// From context.Context (using a *Context passed as interface)
	var stdCtx context.Context = ctx
	assert.Equal(t, web, GetService[*WebService](stdCtx, "web"))

	// From application tracker itself
	assert.Equal(t, web, GetService[*WebService](testApp, "web"))

	// Unknown tracker type (should fallback to global)
	assert.Equal(t, web, GetService[*WebService]("unknown", "web"))

	// Fallback to global app
	// We set the global app for this test
	oldApp := app.Load()
	app.Store(testApp)
	defer app.Store(oldApp)

	assert.Equal(t, web, GetService[*WebService](nil, "web"))

	// Non-existent
	assert.Nil(t, GetService[*WebService](ctx, "none"))

	// Nil app in GetServiceFromApp
	var nilApp *Application
	assert.Nil(t, GetServiceFromApp[*WebService](nilApp, "web"))
}

type mockLifecycleService struct {
	initialized bool
	started     bool
	stopped     bool
	running     bool
}

func (m *mockLifecycleService) Name() string        { return "lifecycle" }
func (m *mockLifecycleService) Description() string { return "lifecycle mock" }
func (m *mockLifecycleService) Initialize(app *Application) error {
	m.initialized = true
	return nil
}
func (m *mockLifecycleService) Start() error {
	m.started = true
	m.running = true
	return nil
}
func (m *mockLifecycleService) Stop() error {
	m.stopped = true
	m.running = false
	return nil
}
func (m *mockLifecycleService) IsRunning() bool { return m.running }

func TestServiceLifecycle(t *testing.T) {
	testApp, _ := NewTestApplication(Options{})
	svc := &mockLifecycleService{}

	t.Run("StartService", func(t *testing.T) {
		err := StartService(testApp, svc)
		assert.NoError(t, err)
		assert.True(t, svc.initialized)
		assert.True(t, svc.started)
		assert.True(t, svc.running)
	})

	t.Run("StopService", func(t *testing.T) {
		err := StopService(testApp, svc)
		assert.NoError(t, err)
		assert.True(t, svc.stopped)
		assert.False(t, svc.running)
	})

	t.Run("StopService_NotRunning", func(t *testing.T) {
		svc.running = false
		err := StopService(testApp, svc)
		assert.NoError(t, err)
	})

	t.Run("stopRunningServices", func(t *testing.T) {
		svc.running = true
		testApp.services["lifecycle"] = svc
		stopRunningServices(testApp)
		assert.False(t, svc.running)
	})

	t.Run("stopRunningServices_Error", func(t *testing.T) {
		failSvc := &failingStopService{running: true}
		testApp.services["fail"] = failSvc
		// This should just log the error and continue
		stopRunningServices(testApp)
	})

	t.Run("StartService_InitializeError", func(t *testing.T) {
		failSvc := &failingInitializerService{}
		err := StartService(testApp, failSvc)
		assert.Error(t, err)
		assert.Equal(t, "init failed", err.Error())
	})

	t.Run("StopService_StopError", func(t *testing.T) {
		failSvc := &failingStopService{running: true}
		err := StopService(testApp, failSvc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stop failed")
	})
}

type failingInitializerService struct{ mockLifecycleService }

func (m *failingInitializerService) Initialize(*Application) error { return fmt.Errorf("init failed") }

type failingStopService struct {
	mockLifecycleService
	running bool
}

func (m *failingStopService) IsRunning() bool { return m.running }
func (m *failingStopService) Stop() error     { return fmt.Errorf("stop failed") }

func TestServiceCLI(t *testing.T) {
	app, _ := NewTestApplication(Options{})
	svc := &mockLifecycleService{}
	app.services["lifecycle"] = svc

	t.Run("listServices", func(t *testing.T) {
		fnc := listServices([]Service{svc})
		assert.NotNil(t, fnc)
		// We can't easily assert stdout here without redirecting it,
		// but calling it ensures the code is executed.
		fnc(nil, nil)
	})

	t.Run("serviceRun", func(t *testing.T) {
		cmd := serviceRun("lifecycle")
		err := cmd(app, nil, nil)
		assert.NoError(t, err)
		assert.True(t, svc.started)

		// Non-existent
		cmdErr := serviceRun("none")(app, nil, nil)
		assert.ErrorIs(t, cmdErr, ErrorServiceNotRegistered)
	})

	t.Run("runAllServices", func(t *testing.T) {
		svc2 := &mockLifecycleService{}
		app.services["svc2"] = svc2

		// Since runAllServices uses errgroup and might run indefinitely if services don't return,
		// we use a mock service that returns immediately (Start returns nil).
		err := runAllServices(app, nil, nil)
		assert.NoError(t, err)
		assert.True(t, svc2.started)
	})
}
