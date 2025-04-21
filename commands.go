package golly

import (
	"fmt"
	"strings"

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
		Run:   Command(runAllServices()),
	}

	commands = []*cobra.Command{
		serviceCommand,
		{
			Use:   "plugins",
			Short: "List all plugins",
			Run:   Command(listAllPluginsCommand),
		},
		{
			Use:     "routes",
			Short:   "Lists registered routes",
			Aliases: []string{"route"},
			Run: Command(func(app *Application, cmd *cobra.Command, args []string) error {
				fmt.Println("Listing Routes:")
				fmt.Println(strings.Join(buildPath(app.routes, ""), "\n"))
				return nil
			}),
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
		run(func(app *Application) error {
			err := command(app, cmd, args)

			if err != nil && err != ErrorExit && err != ErrorNone {
				return err
			}
			return nil
		})
	}
}

func bindCommands(options Options) *cobra.Command {
	rootCMD := &cobra.Command{}
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

	// loop through our plugins incase they are defining
	// cli commands - i am putting this here for now cause it needs
	// to happen prior to rootCMD.execute
	rootCMD.AddCommand(pluginCommands(options.Plugins)...)

	// Any misc commands defined by the end user
	rootCMD.AddCommand(options.Commands...)

	return rootCMD
}
