package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommand_ParseUsage(t *testing.T) {
	tests := []struct {
		name        string
		usage       string
		wantMinArgs int
		wantMaxArgs int
	}{
		{
			name:        "required and optional args",
			usage:       "create <orgID> <name> [description]",
			wantMinArgs: 2,
			wantMaxArgs: 3,
		},
		{
			name:        "only required args",
			usage:       "delete <teamID> <userID>",
			wantMinArgs: 2,
			wantMaxArgs: 2,
		},
		{
			name:        "only optional args",
			usage:       "list [limit] [offset]",
			wantMinArgs: 0,
			wantMaxArgs: 2,
		},
		{
			name:        "no args",
			usage:       "status",
			wantMinArgs: 0,
			wantMaxArgs: 0,
		},
		{
			name:        "empty usage",
			usage:       "",
			wantMinArgs: 0,
			wantMaxArgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{Usage: tt.usage}
			cmd.parseUsage()
			assert.Equal(t, tt.wantMinArgs, cmd.MinArgs)
			assert.Equal(t, tt.wantMaxArgs, cmd.MaxArgs)
		})
	}
}

func TestCommand_Execute(t *testing.T) {
	t.Run("executes handler with correct args", func(t *testing.T) {
		var capturedArgs []string
		var capturedCtx *Context

		cmd := &Command{
			Name:  "create",
			Usage: "create <name> <id>",
			Run: func(ctx *Context, args []string) error {
				capturedCtx = ctx
				capturedArgs = args
				return nil
			},
		}

		err := cmd.Execute([]string{"test-name", "123"})

		assert.NoError(t, err)
		assert.NotNil(t, capturedCtx)
		assert.Equal(t, []string{"test-name", "123"}, capturedArgs)
	})

	t.Run("validates minimum args", func(t *testing.T) {
		cmd := &Command{
			Name:  "create",
			Usage: "create <name> <id>",
			Run: func(ctx *Context, args []string) error {
				return nil
			},
		}

		err := cmd.Execute([]string{"only-one"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires at least 2")
	})

	t.Run("validates maximum args", func(t *testing.T) {
		cmd := &Command{
			Name:  "create",
			Usage: "create <name>",
			Run: func(ctx *Context, args []string) error {
				return nil
			},
		}

		err := cmd.Execute([]string{"one", "two", "three"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts at most 1")
	})

	t.Run("routes to subcommand", func(t *testing.T) {
		var executed bool

		root := &Command{
			Name: "teams",
			Commands: []*Command{
				{
					Name:  "create",
					Usage: "create <name>",
					Run: func(ctx *Context, args []string) error {
						executed = true
						assert.Equal(t, []string{"test"}, args)
						return nil
					},
				},
			},
		}

		err := root.Execute([]string{"create", "test"})
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("shows help for unknown subcommand", func(t *testing.T) {
		root := &Command{
			Name:  "teams",
			Short: "Manage teams",
			Commands: []*Command{
				{Name: "create", Short: "Create team"},
			},
		}

		// Unknown subcommands show root help (Run is nil)
		err := root.Execute([]string{"unknown"})
		assert.NoError(t, err) // Help doesn't error
	})
}

func TestCommand_FluentAPI(t *testing.T) {
	t.Run("builds tree with Add chaining", func(t *testing.T) {
		cmd := NewCommand("teams", "Manage teams").
			Add("create", "create <name>", func(ctx *Context, args []string) error {
				return nil
			}).
			Add("delete", "delete <id>", func(ctx *Context, args []string) error {
				return nil
			})

		assert.Equal(t, "teams", cmd.Name)
		assert.Len(t, cmd.Commands, 2)
		assert.Equal(t, "create", cmd.Commands[0].Name)
		assert.Equal(t, 1, cmd.Commands[0].MinArgs)
		assert.Equal(t, "delete", cmd.Commands[1].Name)
		assert.Equal(t, 1, cmd.Commands[1].MinArgs)
	})

	t.Run("ExactArgs helper", func(t *testing.T) {
		cmd := Command{Name: "test"}
		cmd.ExactArgs(3)

		assert.Equal(t, 3, cmd.MinArgs)
		assert.Equal(t, 3, cmd.MaxArgs)
	})
}

func TestCommand_Help(t *testing.T) {
	t.Run("help flag skips execution", func(t *testing.T) {
		executed := false

		cmd := &Command{
			Name:  "test",
			Usage: "test <arg>",
			Run: func(ctx *Context, args []string) error {
				executed = true
				return nil
			},
		}

		err := cmd.Execute([]string{"--help"})
		assert.NoError(t, err)
		assert.False(t, executed, "should not execute handler for --help")

		err = cmd.Execute([]string{"-h"})
		assert.NoError(t, err)
		assert.False(t, executed, "should not execute handler for -h")
	})
}

// Benchmarks
func BenchmarkCommand_Execute(b *testing.B) {
	cmd := &Command{
		Name:  "create",
		Usage: "create <name> <id>",
		Run: func(ctx *Context, args []string) error {
			return nil
		},
	}

	args := []string{"test-name", "123"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.Execute(args)
	}
}

func BenchmarkCommand_SubcommandRouting(b *testing.B) {
	root := NewCommand("teams", "Manage teams").
		Add("create", "create <name>", func(ctx *Context, args []string) error { return nil }).
		Add("update", "update <id> <name>", func(ctx *Context, args []string) error { return nil }).
		Add("delete", "delete <id>", func(ctx *Context, args []string) error { return nil })

	args := []string{"update", "123", "newname"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = root.Execute(args)
	}
}

func BenchmarkCommand_ParseUsage(b *testing.B) {
	cmd := &Command{Usage: "create <orgID> <name> [description] [tags]"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.MinArgs = 0
		cmd.MaxArgs = 0
		cmd.parseUsage()
	}
}
