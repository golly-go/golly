package golly

import (
	"fmt"

	"golang.org/x/sync/errgroup"
)

func listServicesCommand(ctx *Context, args []string) error {
	fmt.Println("Registered Services:")
	for i, svc := range ctx.Application().Services() {
		fmt.Printf("  %d. %s\n", i+1, getServiceName(svc))
	}
	return nil
}

func runAllServicesCommand(ctx *Context, args []string) error {
	app := ctx.Application()
	eg := &errgroup.Group{}

	for _, svc := range app.Services() {
		service := svc // Capture for closure
		eg.Go(func() error {
			if err := StartService(app, service); err != nil {
				return fmt.Errorf("%s failed: %w", getServiceName(service), err)
			}
			return nil
		})
	}

	// Wait for all services (blocks until shutdown or error)
	return eg.Wait()
}

// buildCommandTree constructs the root command with all subcommands
// We have app here, so we can setup service commands!
func buildCommandTree(app *Application, opts Options) *Command {
	root := &Command{
		Name:  opts.Name,
		Short: fmt.Sprintf("%s CLI", opts.Name),
	}

	if opts.Standalone {
		root.Commands = opts.Commands
		return root
	}

	// Build service command with dynamic subcommands
	serviceCmd := &Command{
		Name:  "service",
		Short: "Manage services",
		Commands: []*Command{
			{Name: "list", Short: "List all services", Run: listServicesCommand},
			{Name: "all", Short: "Run all services", Run: runAllServicesCommand},
		},
	}

	// Add individual service start commands
	for _, svc := range app.Services() {
		name := getServiceName(svc)
		service := svc // Capture for closure

		serviceCmd.Commands = append(serviceCmd.Commands, &Command{
			Name:  name,
			Short: fmt.Sprintf("Start %s service", getServiceDescription(service)),
			Run: func(ctx *Context, args []string) error {
				return StartService(ctx.Application(), service)
			},
		})

		// Add service-specific commands if available
		if sc, ok := service.(ServiceCommands); ok {
			serviceCmd.Commands = append(serviceCmd.Commands, sc.Commands()...)
		}
	}

	root.Commands = append(root.Commands, serviceCmd)

	// Add user commands
	root.Commands = append(root.Commands, opts.Commands...)

	return root
}
