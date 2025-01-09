package golly

import "github.com/spf13/cobra"

var (
	options Options
)

type Options struct {
	// Name of the application (the reason this is dynamic)
	// is it can be assigned by Terraform IE: myapp-consumers, myapp-webservice
	Name string

	// Commands Supported commands
	Commands []*cobra.Command

	// After config is loaded this is the last thing initialized
	// prior to starting the service and entering normal run mode
	Initializers []AppFunc

	// Preboots are executed place before config is initialized
	// and any initializers are ran, this is right after signals
	// are bound
	Preboots []AppFunc

	// Services defines the services we are loading into the system
	// By default Web service will be loaded (This is required for any K8s health checks)
	// Though note we do have a standalone version of this which is used for non web run modes
	Services []Service
}
