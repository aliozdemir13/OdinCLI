package style

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

// TestStyleWrappers tests the basic ANSI string wrapping functions
func TestStyleWrappers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		function func(string) string
	}{
		{"Dim", "test", Dim + "test" + Reset, StyleDim},
		{"Green", "test", Green + "test" + Reset, StyleGreen},
		{"Yellow", "test", Yellow + "test" + Reset, StyleYellow},
		{"Red", "test", Red + "test" + Reset, StyleRed},
		{"Bold", "test", Bold + "test" + Reset, StyleBold},
		{"Indigo", "test", Indigo + "test" + Reset, StyleIndigo},
		{"Cyan", "test", Cyan + "test" + Reset, StyleCyan},
		{"Gray", "test", Gray + "test" + Reset, StyleGray},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestCustomColor tests the hex-to-terminal color mapping
func TestCustomColor(t *testing.T) {
	tests := []struct {
		text     string
		hex      string
		expected string
	}{
		{"Error", "#bf2600", StyleRed("Error")},
		{"Info", "#0747a6", StyleBlue("Info")},
		{"Normal", "#ffffff", "Normal"}, // Default case
	}

	for _, tt := range tests {
		result := CustomColor(tt.text, tt.hex)
		if result != tt.expected {
			t.Errorf("hex %s: expected %q, got %q", tt.hex, tt.expected, result)
		}
	}
}

// TestGetPriorityIcon tests the Jira priority mapping
func TestGetPriorityIcon(t *testing.T) {
	tests := []struct {
		priority string
		contains string // Check if result contains specific arrow or color
	}{
		{"Highest", "▲▲"},
		{"High", "▲"},
		{"Medium", "="},
		{"Low", "▼"},
		{"Lowest", "▼▼"},
		{"Unknown", "-"},
	}

	for _, tt := range tests {
		result := GetPriorityIcon(tt.priority)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("priority %s: expected icon to contain %q, got %q", tt.priority, tt.contains, result)
		}
	}
}

// Helper function to capture Stdout
func captureOutput(f func()) string {
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

// TestPrintCommandList checks if the command list prints expected usage strings
func TestPrintCommandList(t *testing.T) {
	output := captureOutput(func() {
		PrintCommandList()
	})

	expectedCommands := []string{
		"pull ---{{ProjectKey}}",
		"details ---{{IssueKey}}",
		"exit",
		"help",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("expected output to contain command %q", cmd)
		}
	}
}

// TestPrintHeader checks if the command list prints expected usage strings
func TestPrintHeader(t *testing.T) {
	output := captureOutput(func() {
		PrintHeader()
	})

	expectedCommands := []string{
		"pull ---{{ProjectKey}}",
		"details ---{{IssueKey}}",
		"exit",
		"help",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("expected output to contain command %q", cmd)
		}
	}
}

// TestCreateTable ensures the table function doesn't panic with sample data
func TestCreateTable(t *testing.T) {
	header := table.Row{"ID", "Status"}
	body := []table.Row{
		{"PROJ-1", StyleGreen("Done")},
		{"PROJ-2", StyleYellow("In Progress")},
	}

	// We capture output just to ensure it doesn't crash during rendering
	output := captureOutput(func() {
		CreateTable(header, body, nil)
	})

	if !strings.Contains(output, "PROJ-1") {
		t.Error("Table failed to render row data")
	}
}
