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

	commands = []*cobra.Command{
		serviceCommand,
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

	services := append(options.Services, defaultServices...)

	// Add "list-services" command
	listServiceCommand.Run = Command(listServices(services))
	serviceCommand.AddCommand(listServiceCommand)

	// Add individual service commands
	for _, service := range services {
		name := getServiceName(service)
		description := getServiceDescription(service)

		serviceCommand.AddCommand(&cobra.Command{
			Use:   name,
			Short: description,
			Run:   Command(serviceRun(name)),
		})
	}

	// Add other commands and options
	rootCMD.AddCommand(commands...)

	// put this here for now
	for _, plugin := range options.Plugins {
		rootCMD.AddCommand(plugin.Commands()...)
	}

	rootCMD.AddCommand(options.Commands...)

	return rootCMD
}
