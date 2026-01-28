package golly

import (
	"context"
	"fmt"
	"strings"
)

// CommandFunc is the handler signature for commands
type CommandFunc func(*Context, []string) error

// Command represents a CLI command with subcommands
type Command struct {
	Name     string
	Usage    string // "create <orgID> <name> [description]" - auto-parses to MinArgs/MaxArgs
	Short    string
	Long     string
	MinArgs  int
	MaxArgs  int // 0 = unlimited
	Run      CommandFunc
	Commands []*Command
}

// NewCommand creates a new command with fluent API
func NewCommand(name, short string) *Command {
	return &Command{Name: name, Short: short}
}

// Add adds a subcommand and returns parent for chaining
func (c *Command) Add(name, usage string, run CommandFunc) *Command {
	sub := &Command{Name: name, Usage: usage, Run: run}
	sub.parseUsage()
	c.Commands = append(c.Commands, sub)
	return c // Return parent for chaining more Add calls
}

// ExactArgs is a helper for commands requiring exact arg count
func (c *Command) ExactArgs(n int) *Command {
	c.MinArgs = n
	c.MaxArgs = n
	return c
}

// parseUsage extracts MinArgs/MaxArgs from Usage notation
// <required> args become MinArgs, [optional] args add to MaxArgs
func (c *Command) parseUsage() {
	if c.Usage == "" {
		return
	}

	// Count <required> and [optional] args
	required := strings.Count(c.Usage, "<")
	optional := strings.Count(c.Usage, "[")

	// backwards set the name if its not there
	if c.Name == "" {
		parts := strings.Split(c.Usage, " ")
		if len(parts) > 0 {
			c.Name = parts[0]
		}
	}

	// Only auto-set if not manually specified
	if c.MinArgs == 0 && required > 0 {
		c.MinArgs = required
	}

	if c.MaxArgs == 0 && (required+optional) > 0 {
		c.MaxArgs = required + optional
	}
}

// Execute runs the command with given app and args
// App is passed through instead of stored on command for clear data flow
func (c *Command) Execute(app *Application, args []string) error {
	c.parseUsage()

	// Handle help flags FIRST - no app initialization!
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") || args[0] == "help" {
		c.PrintHelp()
		return nil
	}

	// Check for subcommand
	if len(args) > 0 && len(c.Commands) > 0 {
		for _, sub := range c.Commands {
			if sub.Name == args[0] {
				return sub.Execute(app, args[1:])
			}
		}
	}

	// No run function = show help
	if c.Run == nil {
		c.PrintHelp()
		return nil
	}

	// Ensure app is initialized before running command
	if err := ensureAppReady(app); err != nil {
		return err
	}

	// Validate args
	if c.MinArgs > 0 && len(args) < c.MinArgs {
		return fmt.Errorf("%s: requires at least %d argument(s)", c.Name, c.MinArgs)
	}
	if c.MaxArgs > 0 && len(args) > c.MaxArgs {
		return fmt.Errorf("%s: accepts at most %d argument(s)", c.Name, c.MaxArgs)
	}

	// Create context and run
	ctx := NewContextWithApplication(context.Background(), app)

	return c.Run(ctx, args)
}

// PrintHelp displays usage and available commands
func (c *Command) PrintHelp() {
	if c.Usage != "" {
		fmt.Printf("Usage: %s\n\n", c.Usage)
	} else if c.Name != "" {
		fmt.Printf("Usage: %s\n\n", c.Name)
	}

	if c.Short != "" {
		fmt.Printf("%s\n\n", c.Short)
	}

	if c.Long != "" {
		fmt.Printf("%s\n\n", c.Long)
	}

	if len(c.Commands) > 0 {
		fmt.Println("Available Commands:")
		for _, cmd := range c.Commands {
			fmt.Printf("  %-15s %s\n", cmd.Name, cmd.Short)
		}
		fmt.Println()
	}

	fmt.Println("Flags:")
	fmt.Println("  -h, --help   Show this help message")
}
