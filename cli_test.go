package golly

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

// Mock input helper
func mockInput(input string) *bytes.Buffer {
	return bytes.NewBufferString(input + "\n")
}

func TestPrompt(t *testing.T) {
	t.Run("Prompt normal input", func(t *testing.T) {
		input := mockInput("username123")
		SetIn(input)
		defer SetIn(os.Stdin)

		result, err := Prompt("Enter username:", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "username123" {
			t.Errorf("expected 'username123', got '%s'", result)
		}
	})

	t.Run("Prompt exit input", func(t *testing.T) {
		input := mockInput("exit")
		SetIn(input)
		defer SetIn(os.Stdin)

		_, err := Prompt("Enter username:", false)
		if !errors.Is(err, ErrorExit) {
			t.Errorf("expected ErrorExit, got %v", err)
		}
	})
}

func TestPromptForBoolean(t *testing.T) {
	t.Run("Prompt yes response", func(t *testing.T) {
		input := mockInput("y")
		SetIn(input)
		defer SetIn(os.Stdin)

		result := PromptForBoolean("Continue?", false)
		if !result {
			t.Errorf("expected true, got false")
		}
	})

	t.Run("Prompt no response", func(t *testing.T) {
		input := mockInput("n")
		SetIn(input)
		defer SetIn(os.Stdin)

		result := PromptForBoolean("Continue?", true)
		if result {
			t.Errorf("expected false, got true")
		}
	})
}

func TestPromptForOptions(t *testing.T) {
	t.Run("Prompt valid option", func(t *testing.T) {
		input := mockInput("1")
		SetIn(input)
		defer SetIn(os.Stdin)

		options := []string{"Option 1", "Option 2"}
		result, err := PromptForOptions("Choose option:", options)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "Option 1" {
			t.Errorf("expected 'Option 1', got '%s'", result)
		}
	})

	t.Run("Prompt invalid option", func(t *testing.T) {
		input := mockInput("3")
		SetIn(input)
		defer SetIn(os.Stdin)

		options := []string{"Option 1", "Option 2"}
		_, err := PromptForOptions("Choose option:", options)
		if err == nil {
			t.Fatal("expected error for invalid option, got nil")
		}
	})
}

func TestPromptForOptionsInt(t *testing.T) {
	t.Run("Prompt valid option index", func(t *testing.T) {
		input := mockInput("2")
		SetIn(input)
		defer SetIn(os.Stdin)

		options := []string{"Option 1", "Option 2"}
		index, err := PromptForOptionsInt("Choose option:", options)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if index != 1 {
			t.Errorf("expected index 1, got %d", index)
		}
	})

	t.Run("Prompt invalid option index", func(t *testing.T) {
		input := mockInput("5")
		SetIn(input)
		defer SetIn(os.Stdin)

		options := []string{"Option 1", "Option 2"}
		_, err := PromptForOptionsInt("Choose option:", options)
		if err == nil {
			t.Fatal("expected error for invalid option, got nil")
		}
	})
}
