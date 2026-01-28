package golly

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	serviceCommand = &cobra.Command{
		Use:              "service",
		Short:            "Start a named service service",
		Aliases:          []string{"services"},
		TraverseChildren: true,
	}

	listServiceCommand = &cobra.Command{
		Use:   "list",
		Short: "List services",
	}

	allServicesCommand = &cobra.Command{
		Use:   "all",
		Short: "List all services",
		Run:   Command(runAllServices),
	}

	commands = []*cobra.Command{
		serviceCommand,
		{
			Use:   "plugins",
			Short: "List all plugins",
			Run:   Command(listAllPluginsCommand),
		},
	}
)

// CLICommand is a function type representing a CLI command handler.
// It receives a golly.Context, a Cobra command, and the command's arguments.
//
// Example:
//
//	func deleteUser(ctx *golly.Context, cmd *cobra.Command, args []string) error {
//	    fmt.Println("Deleting user:", args)
//	    return nil
//	}
type CLICommand func(*Application, *cobra.Command, []string) error

// Command wraps a CLICommand to execute within the golly application context.
// It ensures the command is executed with error handling and context propagation.
//
// Example:
//
//	{
//	    Use: "delete-users [emailpattern] [orgID]",
//	    Run: golly.Command(deleteUser),
//	}
func Command(command CLICommand) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		a := app.Load()
		if a == nil {
			fmt.Println("Error: application not initialized")
			os.Exit(1)
		}

		err := command(a, cmd, args)
		if err != nil && err != ErrorExit && err != ErrorNone {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	}
}

func bindCommands(app *Application, options Options) *cobra.Command {

	rootCMD := &cobra.Command{
		Use: app.Name,
		// Initialize app before ANY command runs
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := initializeApp(app); err != nil {
				app.logger.Fatal(err)
			}
			_ = setupSignals(app)
			app.changeState(StateRunning)
		},
	}

	if options.Standalone {
		if len(options.Commands) > 0 {
			rootCMD.AddCommand(options.Commands...)
		}
		return rootCMD
	}

	services := append(pluginServices(options.Plugins), options.Services...)

	// Add "list-services" command
	listServiceCommand.Run = listServices(services)

	serviceCommand.AddCommand(listServiceCommand)
	serviceCommand.AddCommand(allServicesCommand)

	// Add individual service commands
	for pos := range services {
		name := getServiceName(services[pos])

		serviceCommand.AddCommand(&cobra.Command{
			Use:   name,
			Short: getServiceDescription(services[pos]),
			Run:   Command(serviceRun(name)),
		})
	}

	// Add other non dynamic application commands and options
	rootCMD.AddCommand(commands...)

	// Add service commands
	for pos := range services {
		if sc, ok := services[pos].(ServiceCommands); ok {
			rootCMD.AddCommand(sc.Commands()...)
		}
	}

	// loop through our plugins incase they are defining
	// cli commands - i am putting this here for now cause it needs
	// to happen prior to rootCMD.execute
	rootCMD.AddCommand(pluginCommands(options.Plugins)...)

	// Any misc commands defined by the end user
	rootCMD.AddCommand(options.Commands...)

	return rootCMD
}
