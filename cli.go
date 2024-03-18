package golly

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	ErrorExit = fmt.Errorf("exit")
	ErrorNone = fmt.Errorf("none")

	colors = map[string]int{
		"cyan":    36,
		"magenta": 35,
		"blue":    34,
		"yellow":  33,
		"green":   32,
		"red":     31,
	}
)

type CLIOptions struct {
	LogFile string
}

type CLICommand func(Context, *cobra.Command, []string) error

func Command(command func(Context, *cobra.Command, []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		Run(func(app Application) error {
			gctx := app.NewContext(context.Background())

			if err := command(gctx, cmd, args); err != nil {
				if err == ErrorExit || err == ErrorNone {
					return nil
				}
				return err
			}
			return nil
		})
	}
}

func Credentials(prompt string) (string, string, error) {
	fmt.Printf("\033[1m%s\033[0m \n", prompt)

	username, err := PromptForField("Enter username: ")
	if err != nil {
		return "", "", err
	}

	password, err := PromptForPassword("Enter Password: ")
	if err != nil {
		return "", "", err
	}

	return username, strings.TrimSpace(password), nil
}

func PromptForPassword(prompt string) (string, error) {
	fmt.Printf("\u001b[35m%s\u001b[0m ", prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	password := string(bytePassword)

	return strings.TrimSpace(password), nil
}

func PromptForField(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\u001b[35m%s\u001b[0m ", prompt)

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

func PromptForBoolean(prompt string, deflt bool) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\u001b[35m%s\u001b[0m ", prompt)

	if deflt {
		fmt.Print("[\033[1mY y yes\033[0m|N n no]")
	} else {
		fmt.Print("[Y y yes|\033[1mN n no\033[0m]")
	}

	input, _ := reader.ReadString('\n')

	switch strings.TrimSpace(input) {
	case "Y", "y", "yes":
		return true
	case "N", "n", "no":
		return false
	default:
		return deflt
	}
}

func Say(color string, prompt string, variables ...any) {
	colorCode := 35
	if c, ok := colors[color]; ok {
		colorCode = c
	}

	fmt.Printf("\u001b[%dm%s\u001b[0m\n", colorCode, fmt.Sprintf(prompt, variables...))

}

func PromptForOptionsInt(prompt string, options []string) (int, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\u001b[35m%s\u001b[0m ", prompt)

	for pos, option := range options {
		fmt.Printf("\n\t[\033[1m%d\033[0m %s\n", pos+1, option)
	}

	input, _ := reader.ReadString('\n')
	str := strings.TrimSpace(input)

	if strings.EqualFold(str, "none") {
		return -1, ErrorNone
	}

	if strings.EqualFold(str, "exit") {
		return 0, ErrorExit
	}

	if input == "" {
		fmt.Printf("Invalid option try again")
		return PromptForOptionsInt(prompt, options)
	}

	opt, err := strconv.Atoi(str)
	if err != nil || opt > len(options) || opt == 0 {
		fmt.Println("Error invalid option try again")
		return PromptForOptionsInt(prompt, options)
	}

	return opt - 1, nil
}

func PromptForOptions(prompt string, options []string) (string, error) {
	opt, err := PromptForIndex(prompt, options)
	if err != nil {
		return "", err
	}

	return options[opt], nil
}

func PromptForIndex(prompt string, options []string) (int, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\u001b[35m%s\u001b[0m ", prompt)

	for pos, option := range options {
		fmt.Printf("\n\t[\033[1m%d\033[0m %s\n", pos+1, option)
	}

	input, _ := reader.ReadString('\n')
	str := strings.TrimSpace(input)

	if strings.EqualFold(str, "exit") {
		return -1, ErrorExit
	}

	if input == "" {
		fmt.Printf("Invalid option try again")
		return PromptForIndex(prompt, options)
	}

	opt, err := strconv.Atoi(str)
	if err != nil || opt > len(options) || opt == 0 {
		fmt.Println("Error invalid option try again")
		return PromptForIndex(prompt, options)
	}

	return opt - 1, nil
}
