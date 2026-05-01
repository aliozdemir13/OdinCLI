package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aliozdemir13/odincli/internal/handler"
	"github.com/aliozdemir13/odincli/internal/models"
)

// TODO: refactor for table-driven test
func TestRunApp(t *testing.T) {
	// 1. Create a temporary config.json
	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: "https://test.atlassian.net", Email: "test@test.com"},
		},
	}

	t.Run("run Help command", func(t *testing.T) {
		// 3. Simulate user typing "help"
		input := strings.NewReader("help\n")
		var output bytes.Buffer

		// 4. Run the app logic
		err := RunApp(input, &output, config, "mock-key")

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
		err := RunApp(input, &output, config, "mock-key")

		if err != nil {
			t.Fatalf("RunApp failed: %v", err)
		}

		// Assert output contains expected strings
		result := output.String()
		if !strings.Contains(result, "Goodbye!") {
			t.Errorf("Expected output to contain 'Goodbye!' but it is %s", result)
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
		input := strings.NewReader("pull ---test\ndetails ---PROJ-1\naddComment ---PROJ-1\nfilter ---myIssues\nsearch ---\"test\"\nstatus ---PROJ-1\nassign ---PROJ-1\ncreate\nunknown\n")
		var output bytes.Buffer

		// Run the app logic
		err := RunApp(input, &output, config, "mock-key")

		if err != nil {
			t.Errorf("No expected error ")
		}
	})
}
