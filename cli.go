package golly

/**
 * Package golly provides CLI utilities designed to streamline the creation of command-line tools
 * within the golly web framework. This package leverages Cobra to create structured, maintainable,
 * and interactive command-line applications.
 *
 * Overview:
 * Although golly is primarily a web framework, command-line tools are crucial for tasks such as
 * administrative scripts, production debugging, or data management. This CLI utility package
 * consolidates commonly used patterns, reducing boilerplate and accelerating development.
 *
 * Key Concepts:
 * - This package integrates with Cobra to define CLI commands and subcommands.
 * - Supports authentication, interactive prompts, and error handling through reusable utilities.
 * - Prompts for options, text, and passwords can be seamlessly embedded into CLI workflows.
 *
 * CLICommand Usage:
 * Commands are defined using the `Command` wrapper, ensuring that each command is executed within
 * the context of the golly framework. This provides access to context-aware functions and simplifies
 * error handling.
 *
 * Example:
 * The example below shows a Cobra command that deletes users based on email patterns and org ID:
 *
 *     {
 *       Use: "delete-users [emailpattern] [orgID]",
 *       Run: golly.Command(deleteUser),
 *     }
 *
 *     func deleteUser(ctx golly.Context, cmd *cobra.Command, args []string) error {
 *       if len(args) < 2 {
 *         return fmt.Errorf("missing required arguments: emailpattern and orgID")
 *       }
 *       emailPattern := args[0]
 *       orgID := args[1]
 *       fmt.Printf("Deleting users with pattern %s in org %s\n", emailPattern, orgID)
 *       return nil
 *     }
 *
 * In this example:
 * - The command `delete-users` accepts two arguments: an email pattern and an organization ID.
 * - The `deleteUser` function handles the command's execution within the golly context.
 * - The use of `golly.Command` ensures consistent error handling and graceful exits.
 *
 * Prompting for Credentials:
 *     username, password, err := Credentials("Authenticate to continue:")
 *     if err != nil {
 *       return err
 *     }
 *     fmt.Printf("Authenticated as %s\n", username)
 *
 * Selecting Options:
 *     option, err := PromptForOptions("Choose an environment:", []string{"Development", "Staging", "Production"})
 *     if err != nil {
 *       return err
 *     }
 *     fmt.Printf("Selected: %s\n", option)
 *
 * Error Handling:
 * - ErrorExit: Triggered when the user types 'exit' to abort the operation.
 * - ErrorNone: Represents a no-op or successful continuation.
 * - ErrorTriesExceeded: Returned after multiple failed input attempts.
 *
 * Testing CLI Commands:
 * Tests simulate user input by overriding `os.Stdin` with mock readers. This ensures that interactive
 * prompts can be tested without manual intervention.
 *
 * Example:
 *     input := mockInput("username123")
 *     SetIn(input)
 *     result, err := Prompt("Enter username:", false)
 *     if err != nil {
 *       t.Fatalf("unexpected error: %v", err)
 *     }
 *
 * The `SetIn` function redirects standard input to a mock buffer, allowing automated testing of CLI interactions.
 */

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var (
	ErrorExit         = errors.New("exited")
	ErrorNone         = errors.New("no error")
	ErrorTriesExeeded = errors.New("tries exeeded")

	colors = map[string]int{
		"cyan":    36,
		"magenta": 35,
		"blue":    34,
		"yellow":  33,
		"green":   32,
		"red":     31,
	}
)

var (
	reader io.Reader = os.Stdin
)

// SetIn sets the global reader to the provided io.Reader for input redirection.
// This allows overriding standard input (os.Stdin) for testing or automation purposes.
//
// Example:
//
//	input := strings.NewReader("test\n")
//	SetIn(input)
//	defer SetIn(os.Stdin)
func SetIn(r io.Reader) {
	lock.Lock()
	reader = r
	lock.Unlock()
}

// Credentials prompts the user for a username and password.
// This function uses `Prompt` to request user input and `term.ReadPassword` for secure password entry.
//
// Example:
//
//	username, password, err := Credentials("Authenticate to continue")
//	if err != nil {
//	    return err
//	}
//	fmt.Println("Authenticated as:", username)
func Credentials(prompt string) (string, string, error) {
	Say("bold", "%s", prompt)

	username, err := Prompt("Enter username: ", false)
	if err != nil {
		return "", "", err
	}

	password, err := Prompt("Enter password: ", true)
	if err != nil {
		return "", "", err
	}

	return username, password, nil
}

// Prompt displays a text input prompt to the user and returns the entered value.
// If `hidden` is true, the input is masked (useful for passwords).
// Returns ErrorExit if the user types 'exit'.
//
// Example:
//
//	result, err := Prompt("Enter project name:", false)
//	password, err := Prompt("Enter password:", true)
func Prompt(prompt string, hidden bool) (string, error) {
	fmt.Printf("\u001b[35m%s\u001b[0m ", prompt)

	if hidden {
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // Newline after password input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(bytePassword)), nil
	}

	reader := bufio.NewReader(reader)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	str := strings.TrimSpace(input)
	if strings.EqualFold(str, "exit") {
		return "", ErrorExit
	}

	return str, nil
}

// PromptForBoolean prompts the user for a yes/no response.
// The default response is returned if the user provides no input.
//
// Example:
//
//	continue := PromptForBoolean("Do you want to continue?", true)
func PromptForBoolean(prompt string, deflt bool) bool {
	options := "[Y/N]"
	if deflt {
		options = "[Y/N] (default: Y)"
	} else {
		options = "[Y/N] (default: N)"
	}

	fmt.Printf("\u001b[35m%s %s\u001b[0m ", prompt, options)
	input, _ := bufio.NewReader(reader).ReadString('\n')

	switch strings.TrimSpace(strings.ToLower(input)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return deflt
	}
}

// Say prints a message with color formatting.
// The color can be one of the predefined ANSI colors such as "red", "blue", or "bold".
//
// Example:
//
//	Say("green", "Deployment successful!")
func Say(color string, prompt string, variables ...any) {
	colorCode := colors["magenta"]
	if c, ok := colors[color]; ok {
		colorCode = c
	}
	fmt.Printf("\u001b[%dm%s\u001b[0m\n", colorCode, fmt.Sprintf(prompt, variables...))
}

// PromptForOptions prompts the user to select an option from a list of strings.
// It returns the selected option or an error if the input is invalid.
//
// Example:
//
//	env, err := PromptForOptions("Choose environment:", []string{"Development", "Staging", "Production"})
func PromptForOptions(prompt string, options []string) (string, error) {
	opt, err := PromptForIndex(prompt, options)
	if err != nil {
		return "", err
	}
	return options[opt], nil
}

// PromptForOptionsInt prompts the user to select an option by index from a list of strings.
// It returns the selected index or an error if the input is invalid.
//
// Example:
//
//	index, err := PromptForOptionsInt("Select a region:", []string{"US", "EU", "Asia"})
func PromptForOptionsInt(prompt string, options []string) (int, error) {
	return promptWithOptions(prompt, options, false)
}

// PromptForIndex prompts the user to select an option by index.
// It allows users to select "none" if the `allowNone` flag is true.
//
// Example:
//
//	idx, err := PromptForIndex("Select database type:", []string{"Postgres", "MySQL"})
func PromptForIndex(prompt string, options []string) (int, error) {
	return promptWithOptions(prompt, options, true)
}

// promptWithOptions provides a consolidated prompt handler for selecting options by index.
// It allows up to 3 retries for invalid input before returning an error.
//
// Example:
//
//	idx, err := promptWithOptions("Choose size:", []string{"Small", "Medium", "Large"}, false)
func promptWithOptions(prompt string, options []string, allowNone bool) (int, error) {
	for tries := 0; tries <= 3; tries++ {
		fmt.Printf("\u001b[35m%s\u001b[0m\n", prompt)

		for i, option := range options {
			fmt.Printf("\t[\033[1m%d\033[0m] %s\n", i+1, option)
		}

		reader := bufio.NewReader(reader)
		fmt.Printf("Choose an option: ")

		input, _ := reader.ReadString('\n')
		str := strings.TrimSpace(input)

		switch {
		case strings.EqualFold(str, "exit"):
			return -1, ErrorExit
		case allowNone && strings.EqualFold(str, "none"):
			return -1, ErrorNone
		case str == "":
			fmt.Println("Invalid input, please try again.")
			continue
		}

		opt, err := strconv.Atoi(str)
		if err != nil || opt < 1 || opt > len(options) {
			fmt.Println("Invalid option, try again.")
			continue
		}

		return opt - 1, nil
	}

	return -1, ErrorTriesExeeded
}
