// Package style provides styling to the application
package style

import (
	"fmt"
	"io"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	// ColorReset resets style decoration
	ColorReset = "\033[0m"
	// ColorDim dims the text
	ColorDim = "\033[2m"
	// ColorRed colors text as red
	ColorRed = "\033[31m"
	// ColorGreen colors text as green
	ColorGreen = "\033[32m"
	// ColorYellow colors text as yellow
	ColorYellow = "\033[33m"
	// ColorBlue colors text as blue
	ColorBlue = "\033[34m"
	// ColorCyan colors text as cyan
	ColorCyan = "\033[36m"
	// ColorGray colors text as gray
	ColorGray = "\033[90m"
	// ColorIndigo colors text as indigo
	ColorIndigo = "\033[38;5;141m"
	// TextBold styles text as bold
	TextBold = "\033[1m"
)

// PrintHeader returns the logo and the command list for guidance
func PrintHeader(stdout io.Writer) {
	logo := `
  ██████╗  ██████╗  ██╗ ███╗   ██╗
 ██╔═══██╗ ██╔══██╗ ██║ ████╗  ██║
 ██║   ██║ ██║  ██║ ██║ ██╔██╗ ██║
 ██║   ██║ ██║  ██║ ██║ ██║╚██╗██║
 ╚██████╔╝ ██████╔╝ ██║ ██║ ╚████║
  ╚═════╝  ╚═════╝  ╚═╝ ╚═╝  ╚═══╝`
	fmt.Println(ColorCyan + logo + ColorReset)

	PrintCommandList(stdout)
}

// PrintCommandList returns the command list for guidance
func PrintCommandList(stdout io.Writer) {
	fmt.Fprintln(stdout, Dim("\nUsage: pull ---{{ProjectKey}}"))
	fmt.Fprintln(stdout, Dim("Usage: details ---{{IssueKey}}"))
	fmt.Fprintln(stdout, Dim("Usage: details ---epic {{KEY}}"))
	fmt.Fprintln(stdout, Dim("Usage: search ---\"{{phrase}}\""))
	fmt.Fprintln(stdout, Dim("Usage: filter ---status {{Status}}"))
	fmt.Fprintln(stdout, Dim("Usage: filter ---prio {{Priority}}"))
	fmt.Fprintln(stdout, Dim("Usage: filter ---myIssues"))
	fmt.Fprintln(stdout, Dim("Usage: filter ---currentSprint"))
	fmt.Fprintln(stdout, Dim("Usage: filter ---backlog"))
	fmt.Fprintln(stdout, Dim("Usage: filter ---epics"))
	fmt.Fprintln(stdout, Dim("Usage: addComment ---{{IssueKey}}"))
	fmt.Fprintln(stdout, Dim("Usage: status ---{{IssueKey}}"))
	fmt.Fprintln(stdout, Dim("Usage: assign ---{{IssueKey}}"))
	fmt.Fprintln(stdout, Dim("Usage: create"))
	fmt.Fprintln(stdout, Dim("Usage: exit"))
	fmt.Fprintln(stdout, Dim("Usage: help"))
}

// Dim decorates text to dim it
func Dim(t string) string { return ColorDim + t + ColorReset }

// Green decorates text with green color
func Green(t string) string { return ColorGreen + t + ColorReset }

// Yellow decorates text with yellow color
func Yellow(t string) string { return ColorYellow + t + ColorReset }

// Blue decorates text with blue color
func Blue(t string) string { return ColorBlue + t + ColorReset }

// Red decorates text with red color
func Red(t string) string { return ColorRed + t + ColorReset }

// Indigo decorates text with indigo color
func Indigo(t string) string { return ColorIndigo + t + ColorReset }

// Cyan decorates text with cyan color
func Cyan(t string) string { return ColorCyan + t + ColorReset }

// Bold decorates text to make it bold
func Bold(t string) string { return TextBold + t + ColorReset }

// Gray decorates text with gray color
func Gray(t string) string { return ColorGray + t + ColorReset }

// CustomColor maps Jira Hex colors to basic Terminal colors
func CustomColor(text string, hex string) string {
	switch hex {
	case "#bf2600":
		return Red(text) // Reddish
	case "#0747a6":
		return Blue(text) // Bluish
	default:
		return text
	}
}

// GetPriorityIcon returns icons for the jira ticket priorities
func GetPriorityIcon(priority string) string {
	switch priority {
	case "Highest":
		return ColorRed + TextBold + "  [▲▲] " + ColorReset // Double up
	case "High":
		return ColorRed + "  [▲]  " + ColorReset // Single up
	case "Medium":
		return ColorYellow + "  [=]  " + ColorReset // Equal / Neutral
	case "Low":
		return ColorBlue + "  [▼]  " + ColorReset // Single down
	case "Lowest":
		return ColorCyan + "  [▼▼] " + ColorReset // Double down
	default:
		return ColorDim + "  [-]  " + ColorReset // Unknown
	}
}

// PrintCommandUsage is the global handler of the help command display
func PrintCommandUsage(name string) {
	fmt.Println("Usage:")
	switch name {
	case "filter":
		fmt.Println(Yellow("  filter ---status In Progress"))
		fmt.Println(Yellow("  filter ---prio High"))
		fmt.Println(Yellow("  filter ---myIssues"))
		fmt.Println(Yellow("  filter ---currentSprint"))
		fmt.Println(Yellow("  filter ---backlog"))
		fmt.Println(Yellow("  filter ---epics"))

	case "pull":
		fmt.Println(Yellow("  pull ---{{ProjectKey}}"))

	case "details":
		fmt.Println(Yellow("  details ---{{Key}}"))
		fmt.Println(Yellow("  details ---epic {{Key}}"))

	case "addComment":
		fmt.Println(Yellow("  addComment {{Key}}"))

	case "status":
		fmt.Println(Yellow("  status ---{{KEY}}"))

	case "assign":
		fmt.Println(Yellow("  assign ---{{KEY}}"))

	case "search":
		fmt.Println(Yellow("  search ---\"your keyword or phrase\""))
	}
}

// CreateTable returns prettify table view of the information
func CreateTable(header table.Row, body []table.Row, columnConfig []table.ColumnConfig) {
	// Create the Table Writer
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Define Headers
	t.AppendHeader(header)

	for _, row := range body {
		// Add Row
		t.AppendRow(row)
	}

	// Style the table to look like your original request
	// We use StyleLight but remove the borders for a "clean" look
	style := table.StyleLight
	style.Options.DrawBorder = false
	style.Options.SeparateColumns = false
	style.Options.SeparateHeader = true
	style.Box.PaddingLeft = ""
	style.Box.PaddingRight = "   " // Matches your spacing
	t.SetStyle(style)

	if columnConfig != nil {
		t.SetColumnConfigs(columnConfig)
	}

	// Render
	t.Render()
	fmt.Println()
}
