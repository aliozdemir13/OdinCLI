package style

import (
	"bytes"
	"io"
	"os"
	"regexp"
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
		{"Dim", "test", ColorDim + "test" + ColorReset, Dim},
		{"Green", "test", ColorGreen + "test" + ColorReset, Green},
		{"Yellow", "test", ColorYellow + "test" + ColorReset, Yellow},
		{"Red", "test", ColorRed + "test" + ColorReset, Red},
		{"Bold", "test", TextBold + "test" + ColorReset, Bold},
		{"Indigo", "test", ColorIndigo + "test" + ColorReset, Indigo},
		{"Cyan", "test", ColorCyan + "test" + ColorReset, Cyan},
		{"Gray", "test", ColorGray + "test" + ColorReset, Gray},
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
		{"Error", "#bf2600", Red("Error")},
		{"Info", "#0747a6", Blue("Info")},
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

// stripANSI removes escape codes like [33m from a string
func stripANSI(str string) string {
	const ansi = `\x1b\[[0-9;]*[a-zA-Z]`
	re := regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}

// TestPrintCommandUsage checks if the command usage prints expected usage strings
func TestPrintCommandUsage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"filter", "filter", "filter ---status In Progress"},
		{"pull", "pull", "pull ---{{ProjectKey}}"},
		{"details", "details", "details ---{{Key}}"},
		{"addComment", "addComment", "addComment {{Key}}"},
		{"status", "status", "status ---{{KEY}}"},
		{"assign", "assign", "assign ---{{KEY}}"},
		{"search", "search", "search ---\"your keyword or phrase\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				PrintCommandUsage(tt.input)
			})

			cleanOutput := stripANSI(output)

			if !strings.Contains(cleanOutput, tt.expected) {
				t.Errorf("\nCommand: %s\nExpected to contain: %q\nActual (cleaned): %q", tt.name, tt.expected, cleanOutput)
			}
		})
	}
}

// TestCreateTable ensures the table function doesn't panic with sample data
func TestCreateTable(t *testing.T) {
	header := table.Row{"ID", "Status"}
	body := []table.Row{
		{"PROJ-1", Green("Done")},
		{"PROJ-2", Yellow("In Progress")},
	}

	// We capture output just to ensure it doesn't crash during rendering
	output := captureOutput(func() {
		CreateTable(header, body, nil)
	})

	if !strings.Contains(output, "PROJ-1") {
		t.Error("Table failed to render row data")
	}
}
