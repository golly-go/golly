package golly

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type mockService struct {
	name string
}

func (ms *mockService) Name() string                   { return ms.name }
func (*mockService) IsRunning() bool                   { return false }
func (*mockService) Initialize(app *Application) error { return nil }
func (*mockService) Start() error                      { return nil }
func (*mockService) Stop() error                       { return nil }

func TestGetServiceName(t *testing.T) {
	namedService := &mockService{name: "NamedService"}
	unnamedService := &mockService{}

	assert.Equal(t, "NamedService", getServiceName(namedService))
	assert.NotEmpty(t, getServiceName(unnamedService)) // Should return type name
}

func TestBindCommands(t *testing.T) {
	options := Options{
		Services: []Service{
			&mockService{name: "ServiceA"},
		},
		Commands: []*cobra.Command{
			{Use: "custom", Short: "Custom Command"},
		},
	}

	app := NewApplication(options)
	rootCMD := bindCommands(app, options)
	assert.NotNil(t, rootCMD)
	assert.Len(t, rootCMD.Commands(), 3)
}

func TestSetScheduledServices(t *testing.T) {
	options := Options{
		Services: []Service{
			&mockService{name: "ServiceA"},
			&mockService{name: "ServiceB"},
		},
		Commands: []*cobra.Command{
			{Use: "custom", Short: "Custom Command"},
		},
	}

	app := NewApplication(options)
	rootCMD := bindCommands(app, options)

	// Test 1: executing "myapp service ServiceA"
	cmd, _, err := rootCMD.Find([]string{"service", "ServiceA"})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "ServiceA", cmd.Name())

	app.runningServices = []string{}
	setScheduledServices(app, cmd)
	assert.True(t, app.IsServiceScheduled("ServiceA"))
	assert.False(t, app.IsServiceScheduled("ServiceB"))

	// Test 2: executing "myapp service all"
	cmdAll, _, err := rootCMD.Find([]string{"service", "all"})
	assert.NoError(t, err)
	assert.NotNil(t, cmdAll)

	app.runningServices = []string{}
	setScheduledServices(app, cmdAll)
	assert.True(t, app.IsServiceScheduled("ServiceA"))
	assert.True(t, app.IsServiceScheduled("ServiceB"))

	// Test 3: executing "myapp custom"
	cmdCustom, _, err := rootCMD.Find([]string{"custom"})
	assert.NoError(t, err)
	assert.NotNil(t, cmdCustom)

	app.runningServices = []string{}
	setScheduledServices(app, cmdCustom)
	assert.False(t, app.IsServiceScheduled("ServiceA"))
	assert.False(t, app.IsServiceScheduled("ServiceB"))
}
