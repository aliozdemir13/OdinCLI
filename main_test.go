package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliozdemir13/odincli/internal/handler"
)

func TestRunApp(t *testing.T) {
	// 1. Create a temporary config.json
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configData := `{"projects": {"TEST": {"url": "http://localhost", "email": "a@b.com"}}}`
	os.WriteFile(configPath, []byte(configData), 0644)

	// 2. Mock the API_KEY environment variable
	t.Setenv("API_KEY", "mock-key")

	t.Run("run Help command", func(t *testing.T) {
		// 3. Simulate user typing "help"
		input := strings.NewReader("help\n")
		var output bytes.Buffer

		// 4. Run the app logic
		err := RunApp(input, &output, ".env", configPath)

		if err != nil {
			t.Fatalf("RunApp failed: %v", err)
		}

		// 5. Assert output contains expected strings
		result := output.String()
		if !strings.Contains(result, "Usage: pull ---{{ProjectKey}}") {
			t.Errorf("Expected output to contain 'Usage: pull ---{{ProjectKey}}' but it is %s", &output)
		}
	})

	t.Run("run Exit command", func(t *testing.T) {
		// Simulate user typing "exit"
		input := strings.NewReader("exit\n")
		var output bytes.Buffer

		// run the app logic
		err := RunApp(input, &output, ".env", configPath)

		if err != nil {
			t.Fatalf("RunApp failed: %v", err)
		}

		// Assert output contains expected strings
		result := output.String()
		if !strings.Contains(result, "Goodbye!") {
			t.Errorf("Expected output to contain 'Goodbye!' but it is %s", result)
		}
	})

	t.Run("Invalid Config", func(t *testing.T) {
		input := strings.NewReader("exit\n")
		var output bytes.Buffer

		// Pass a non-existent config path
		_ = RunApp(input, &output, ".env", "wrong_path.json")

		if !strings.Contains(output.String(), "config.json not found") {
			t.Errorf("Expected error message for missing config but it is %s", &output)
		}
	})

	t.Run("missing .env", func(t *testing.T) {
		// Simulate user typing "exit"
		input := strings.NewReader("exit\n")
		var output bytes.Buffer

		// Run the app logic
		_ = RunApp(input, &output, ".env.example", configPath)

		if !strings.Contains(output.String(), "Error loading .env file") {
			t.Errorf("Expected 'Error loading .env file' but it is %s", &output)
		}
	})

	oldEditor := handler.RunCommendEditor
	oldForm := handler.RunIssueForm
	oldDescription := handler.RunDescriptionEditor

	handler.RunCommendEditor = func() string { return "test content" }
	handler.RunIssueForm = func() (string, string, string, string, error) {
		return "Summary", "Task", "5", "", nil
	}
	handler.RunDescriptionEditor = func() (string, bool) {
		return "Description", false
	}

	// Restore them after the test
	defer func() {
		handler.RunCommendEditor = oldEditor
		handler.RunIssueForm = oldForm
		handler.RunDescriptionEditor = oldDescription
	}()

	t.Run("cover cases", func(t *testing.T) {
		// Simulate user typing "exit"
		input := strings.NewReader("pull ---test\ndetails ---PROJ-1\naddComment ---PROJ-1\nfilter ---myIssues\nsearch ---\"test\"\nstatus ---PROJ-1\nassign ---PROJ-1\ncreate\n")
		var output bytes.Buffer

		// Run the app logic
		err := RunApp(input, &output, ".env", configPath)

		if err != nil {
			t.Errorf("No expected error ")
		}
	})
}
