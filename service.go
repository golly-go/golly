package golly

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

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

type Initializer interface {
	Initialize(*Application) error
}

// Service defines a self-contained component within the Golly framework that runs independently and can be started or stopped separately from the main application.
// While services usually align with the application's lifecycle, they can also be started or stopped dynamically, including from the command line (e.g., `golly service service_name`).
type Service interface {
	// Start activates the service, beginning its operation.
	Start() error

	// Stop gracefully halts the service's operation.
	Stop() error

	// IsRunning indicates whether the service is currently active.
	IsRunning() bool
}

type ServiceCommands interface {
	Commands() []*cobra.Command
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

// startService starts a service.
//
// Parameters:
//   - app: The application instance.
//   - service: The service to start.
//
// Returns:
//   - An error if the service fails to start.
//   - nil if the service starts successfully.
//
// we should probably move this to (application struct as it starts to need more and more of it)
func StartService(app *Application, service Service) error {
	app.logger.Tracef("Starting service: %s", getServiceName(service))

	if i, ok := service.(Initializer); ok {
		if err := i.Initialize(app); err != nil {
			return err
		}
	}

	app.Events().Dispatch(
		WithApplication(context.Background(), app),
		&ServiceStarted{Name: getServiceName(service)})

	return service.Start()
}

// StopService stops a specific service and emits the ServiceStopped event
func StopService(app *Application, service Service) error {
	name := getServiceName(service)
	app.logger.Tracef("Stopping service: %s", name)

	if !service.IsRunning() {
		return nil
	}

	if err := service.Stop(); err != nil {
		return fmt.Errorf("error stopping service %s: %w", name, err)
	}

	app.Events().Dispatch(
		WithApplication(context.Background(), app),
		&ServiceStopped{Name: name})

	return nil
}

// stopRunningServices stops all running services.
func stopRunningServices(app *Application) {
	for _, svc := range app.services {
		if !svc.IsRunning() {
			continue
		}
		if err := StopService(app, svc); err != nil {
			app.logger.Error(err)
		}
	}
}

// GetService retrieves a service by name with type safety.
// Tries context first, then falls back to global.
func GetService[T Service](tracker any, name string) T {
	var a *Application

	switch c := tracker.(type) {
	case *Context:
		a = c.Application()
	case *WebContext:
		a = c.Application()
	case *Application:
		a = c
	case context.Context:
		// Try to convert to Golly context
		if gctx, ok := c.(*Context); ok {
			a = gctx.Application()
		} else {
			a = app // fallback to global
		}
	default:
		a = app
	}

	return GetServiceFromApp[T](a, name)
}

// GetServiceFromApp retrieves a service by name directly from an application instance.
func GetServiceFromApp[T Service](app *Application, name string) T {
	var zero T

	if app == nil {
		return zero
	}

	if svc, ok := app.services[name].(T); ok {
		return svc
	}

	return zero
}

/***
 * CLI Commands
 */

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

		return StartService(app, service)
	}
}

// runAllServices creates a CLICommand for running all registered services concurrently.
//
// Returns an error if any service stops unexpectedly before shutdown.
func runAllServices(app *Application, cmd *cobra.Command, args []string) error {
	var eg errgroup.Group

	// Run each service in its own goroutine
	for _, svc := range app.services {
		// Note: StartService handles Initialize() if needed

		fnc := func(svc Service, name string) func() error {
			return func() error {
				if err := StartService(app, svc); err != nil {
					return errors.New("service '" + name + "' terminated unexpectedly: " + err.Error())
				}
				return nil
			}
		}

		eg.Go(fnc(svc, getServiceName(svc)))
	}

	return eg.Wait()
}
