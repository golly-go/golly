package golly

import (
	"errors"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
)

var (
	ErrorServiceNotRegistered = errors.New("Service is not registered")
)

type Namer interface {
	Name() string
}

type Descriptioner interface {
	Description() string
}

// Service defines a self-contained component within the Golly framework that runs independently and can be started or stopped separately from the main application.
// While services usually align with the application's lifecycle, they can also be started or stopped dynamically, including from the command line (e.g., `golly service service_name`).
type Service interface {
	// Initialize prepares the service before it starts running.
	Initialize(*Application) error

	// Start activates the service, beginning its operation.
	Start() error

	// Stop gracefully halts the service's operation.
	Stop() error

	// IsRunning indicates whether the service is currently active.
	IsRunning() bool
}

// serviceMap converts a slice of Service into a map with service names as keys.
// It uses the Namer interface to determine the service name, or falls back
// to the type name if the interface is not implemented.
//
// Parameters:
//   - services: A slice of Service instances to map.
//
// Returns:
//   - A map where keys are service names and values are the corresponding Service instances.
func serviceMap(services []Service) map[string]Service {
	ret := make(map[string]Service)

	// do break on nil
	if len(services) == 0 {
		return ret
	}

	for _, service := range services {
		ret[getServiceName(service)] = service
	}

	return ret
}

// getServiceName retrieves the name of a given service. If the service implements
// the Namer interface, its Name method is used. If the Name method returns an empty
// string or the interface is not implemented, the type name of the service is returned.
//
// Parameters:
//   - service: The Service whose name is to be retrieved.
//
// Returns:
//   - A string representing the name of the service.
func getServiceName(service Service) string {
	if n, ok := service.(Namer); ok {
		if name := n.Name(); name != "" {
			return name
		}
	}
	return TypeNoPtr(service).String()
}

// getServiceDescription retrieves the description of a given service. If the service
// implements the Descriptioner interface, its Description method is used. If the interface
// is not implemented, a default message is returned indicating no description is available.
//
// Parameters:
//   - service: The Service whose description is to be retrieved.
//
// Returns:
//   - A string representing the description of the service, or a default message if no description is available.
func getServiceDescription(service Service) string {
	if d, ok := service.(Descriptioner); ok {
		return d.Description()
	}
	return fmt.Sprintf("No description for %s", getServiceName(service))
}

// listServices creates a CLICommand that lists all registered services.
// The services are printed to the console with their names in a numbered list.
//
// Parameters:
//   - services: A slice of Service instances to be listed.
//
// Returns:
//   - A CLICommand function to execute the listing command within the application context.
func listServices(services []Service) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fmt.Println("Registered Services:")

		for i, service := range services {
			name := getServiceName(service)
			fmt.Printf("\t %d. %s\n", i+1, name)
		}
	}
}

// serviceRun creates a CLICommand for running a specific service by name.
// The function can be extended with logic to start or manage the service.
//
// Parameters:
//   - name: The name of the service to run.
//
// Returns:
//   - A CLICommand function to execute the service run command within the application context.
func serviceRun(name string) CLICommand {
	return func(app *Application, cmd *cobra.Command, args []string) error {
		service, exists := app.services[name]
		if !exists {
			return ErrorServiceNotRegistered
		}

		if err := service.Initialize(app); err != nil {
			return err
		}

		app.events.
			Register(EventShutdown, func(*Context, *Event) { service.Stop() })

		// Add actual service start logic here
		return service.Start()
	}
}

// runAllServices creates a CLICommand for running all registered services concurrently.
//
// Returns an error if any service stops unexpectedly before shutdown.
func runAllServices() CLICommand {
	return func(app *Application, cmd *cobra.Command, args []string) error {
		var wg sync.WaitGroup
		errChan := make(chan error, len(app.services))

		// Run each service in its own goroutine
		for name, svc := range app.services {
			app.logger.Tracef("Starting service: %s", name)

			// Initialize the service
			if err := svc.Initialize(app); err != nil {
				return err
			}

			// Handle shutdown event for this service
			app.events.Register(EventShutdown, func(*Context, *Event) {
				svc.Stop()
			})

			wg.Add(1)
			go func(serviceName string, service Service) {
				defer wg.Done()
				if err := service.Start(); err != nil {
					errChan <- errors.New("service '" + serviceName + "' terminated unexpectedly: " + err.Error())
				}
			}(name, svc)
		}

		// Wait for all services to stop
		wg.Wait()
		close(errChan)

		// Listen for service errors
		for err := range errChan {
			if err != nil {
				app.Shutdown()
				return err
			}
		}

		return nil
	}
}
