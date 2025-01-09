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

	rootCMD := bindCommands(options)
	assert.NotNil(t, rootCMD)
	assert.Len(t, rootCMD.Commands(), 3)
}
