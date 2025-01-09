package golly

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	defaultServices = []Service{
		&WebService{},
	}

	ErrorServiceNotRegistered = errors.New("Service is not registered")
)

type Namer interface {
	Name() string
}

type Descriptioner interface {
	Description() string
}

// Service this holds a service definition for golly,
// not 100% sure i like the event engine either but
// as i decouple various pieces i flush this out
type Service interface {
	Initialize(*Application) error
	Stop() error
	Start() error
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

	svc := append(defaultServices, services...)
	for _, service := range svc {
		if nmr, ok := service.(Namer); ok {
			ret[nmr.Name()] = service
			continue
		}

		ret[InfNameNoPackage(service)] = service
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
func listServices(services []Service) CLICommand {
	return func(app *Application, cmd *cobra.Command, args []string) error {
		fmt.Println("Registered Services:")

		for i, service := range services {
			name := getServiceName(service)
			fmt.Printf("%d %s\n", i+1, name)
		}

		return nil
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
