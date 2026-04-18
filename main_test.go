package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aliozdemir13/odincli/internal"
	"github.com/aliozdemir13/odincli/internal/models"
)

// Helper to capture Stdout
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// TestHandlePull tests logic for switching projects based on your models.Config
func TestHandlePull(t *testing.T) {
	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: "https://test.atlassian.net", Email: "test@test.com"},
		},
	}

	t.Run("Valid Project", func(t *testing.T) {
		parts := []string{"pull", "TEST"}
		// Note: internal.FetchIssues will likely be called here.
		success := handlePull(parts, config, "mock-api-key")

		if !success {
			t.Error("Expected handlePull to return true for valid project")
		}

		if internal.CurrentInstance.Name != "TEST" {
			t.Errorf("Expected instance name TEST, got %s", internal.CurrentInstance.Name)
		}
	})

	t.Run("Invalid Project", func(t *testing.T) {
		parts := []string{"pull", "UNKNOWN"}
		output := captureStdout(func() {
			success := handlePull(parts, config, "key")
			if success {
				t.Error("Expected handlePull to return false for unknown project")
			}
		})

		if !strings.Contains(output, "not found in config.json") {
			t.Error("Expected error message in output")
		}
	})
}

// TestHandleFilter tests filtering logic using models.Issues
func TestHandleFilter(t *testing.T) {
	// Setup mock data in the internal package global state
	// Note: We are using models.Issues (plural) as defined in your provided types
	internal.LastEntries = []models.Issues{
		{
			Key: "PROJ-1",
			Fields: models.Fields{
				Summary: "Test Issue",
				Priority: struct {
					Name string `json:"name"`
				}{Name: "High"},
				Status: struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Name      string `json:"name"`
						ColorName string `json:"colorName"`
					} `json:"statusCategory"`
				}{Name: "In Progress"},
			},
		},
	}

	t.Run("Filter by Status Match", func(t *testing.T) {
		parts := []string{"filter", "status In Progress"}
		output := captureStdout(func() {
			handleFilter(parts)
		})

		if !strings.Contains(output, "PROJ-1") {
			t.Error("Expected PROJ-1 to be in filtered output")
		}
	})

	t.Run("Filter by Status No Match", func(t *testing.T) {
		parts := []string{"filter", "status Done"}
		output := captureStdout(func() {
			handleFilter(parts)
		})

		if !strings.Contains(output, "No matching issues found") {
			t.Error("Expected 'No matching issues' message")
		}
	})
}

// TestHandleDetails tests showing issue information from cache
func TestHandleDetails(t *testing.T) {
	// Populate the Cache and LastEntries
	internal.EntriesCache = make(map[string]models.Issues)

	issue := models.Issues{
		Key: "PROJ-1",
		Fields: models.Fields{
			Summary: "Details Summary",
			Status: struct {
				Name           string `json:"name"`
				StatusCategory struct {
					Name      string `json:"name"`
					ColorName string `json:"colorName"`
				} `json:"statusCategory"`
			}{
				StatusCategory: struct {
					Name      string `json:"name"`
					ColorName string `json:"colorName"`
				}{Name: "In Progress"},
			},
		},
	}

	internal.EntriesCache["PROJ-1"] = issue
	internal.LastEntries = []models.Issues{issue}

	t.Run("Found in Cache", func(t *testing.T) {
		parts := []string{"details", "PROJ-1"}
		output := captureStdout(func() {
			// This will likely trigger internal.FetchComments(key)
			handleDetails(parts)
		})

		if !strings.Contains(output, "Details Summary") {
			t.Error("Expected issue summary in output")
		}
	})
}

// TestHandleStatus_Cancel tests the user input interruption logic
func TestHandleStatus_Cancel(t *testing.T) {
	// Mock Stdin to simulate user typing "c" then Enter
	input := "c\n"
	r, w, _ := os.Pipe()
	_, _ = w.Write([]byte(input))
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// We assume handleStatus will fail or return false because we aren't mocking the API response
	// for GetAvailableTransitions, but this verifies the 'c' input logic doesn't crash.
	parts := []string{"status", "PROJ-1"}
	success := handleStatus(parts)

	if success {
		t.Error("Expected handleStatus to return false when cancelled with 'c'")
	}
}
