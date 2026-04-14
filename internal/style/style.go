package style

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	Reset  = "\033[0m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
	Indigo = "\033[38;5;141m"
	Bold   = "\033[1m"
)

func PrintHeader() {
	logo := `
  ██████╗  ██████╗  ██╗ ███╗   ██╗
 ██╔═══██╗ ██╔══██╗ ██║ ████╗  ██║
 ██║   ██║ ██║  ██║ ██║ ██╔██╗ ██║
 ██║   ██║ ██║  ██║ ██║ ██║╚██╗██║
 ╚██████╔╝ ██████╔╝ ██║ ██║ ╚████║
  ╚═════╝  ╚═════╝  ╚═╝ ╚═╝  ╚═══╝`
	fmt.Println(Cyan + logo + Reset)

	PrintCommandList()
}

func PrintCommandList() {
	fmt.Println(StyleDim("\nUsage: pull ---{{ProjectKey}}"))
	fmt.Println(StyleDim("Usage: details ---{{IssueKey}}"))
	fmt.Println(StyleDim("Usage: search ---{{IssueKey}}"))
	fmt.Println(StyleDim("Usage: filter ---status {{Status}}"))
	fmt.Println(StyleDim("Usage: filter ---prio {{Priority}}"))
	fmt.Println(StyleDim("Usage: filter ---myIssues"))
	fmt.Println(StyleDim("Usage: filter ---currentSprint"))
	fmt.Println(StyleDim("Usage: filter ---backlog"))
	fmt.Println(StyleDim("Usage: filter ---epics"))
	fmt.Println(StyleDim("Usage: addComment ---{{IssueKey}}"))
	fmt.Println(StyleDim("Usage: exit"))
	fmt.Println(StyleDim("Usage: help"))
}

func StyleDim(t string) string    { return Dim + t + Reset }
func StyleGreen(t string) string  { return Green + t + Reset }
func StyleYellow(t string) string { return Yellow + t + Reset }
func StyleBlue(t string) string   { return Blue + t + Reset }
func StyleRed(t string) string    { return Red + t + Reset }
func StyleIndigo(t string) string { return Indigo + t + Reset }
func StyleCyan(t string) string   { return Cyan + t + Reset }
func StyleBold(t string) string   { return Bold + t + Reset }
func StyleGray(t string) string   { return Gray + t + Reset }

// CustomColor maps Jira Hex colors to basic Terminal colors
func CustomColor(text string, hex string) string {
	switch hex {
	case "#bf2600":
		return StyleRed(text) // Reddish
	case "#0747a6":
		return StyleBlue(text) // Bluish
	default:
		return text
	}
}

func GetPriorityIcon(priority string) string {
	switch priority {
	case "Highest":
		return Red + Bold + "  [▲▲] " + Reset // Double up
	case "High":
		return Red + "  [▲]  " + Reset // Single up
	case "Medium":
		return Yellow + "  [=]  " + Reset // Equal / Neutral
	case "Low":
		return Blue + "  [▼]  " + Reset // Single down
	case "Lowest":
		return Cyan + "  [▼▼] " + Reset // Double down
	default:
		return Dim + "  [-]  " + Reset // Unknown
	}
}

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
