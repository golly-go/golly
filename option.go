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

	// Loaded before code initializers
	// reserved for plugins, database connection and required things
	// business logic should be in Initializers
	// Dependecies []AppFunc

	// After config is loaded this is the last thing initialized
	// prior to starting the service and entering normal run mode
	Initializer AppFunc

	// Preboot are executed place before config is initialized
	// and any initializers are ran, this is right after signals
	// are bound
	Preboot AppFunc

	Plugins []Plugin

	// Services defines the services we are loading into the system
	// By default Web service will be loaded (This is required for any K8s health checks)
	// Though note we do have a standalone version of this which is used for non web run modes
	Services []Service

	// WatchConfig if true will watch the config file for changes and reloaded
	// golly will dispatch a ConfigChanged event when the config file is changed
	WatchConfig bool

	// Standalone if true will run in standalone mode
	// this is good for cli tools that want to provide their own commands and not
	// become confusing with the default commands
	Standalone bool
}
