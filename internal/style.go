package internal

import "fmt"

const (
	Reset  = "\033[0m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
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
	fmt.Println(StyleDim("Usage: addComment ---{{IssueKey}} text for the comment"))
	fmt.Println(StyleDim("Usage: myIssues"))
	fmt.Println(StyleDim("Usage: exit"))
	fmt.Println(StyleDim("Usage: help"))
}

func StyleDim(t string) string    { return Dim + t + Reset }
func StyleGreen(t string) string  { return Green + t + Reset }
func StyleYellow(t string) string { return Yellow + t + Reset }
func StyleBlue(t string) string   { return Blue + t + Reset }
func StyleRed(t string) string    { return Red + t + Reset }
func StyleCyan(t string) string   { return Cyan + t + Reset }
func StyleBold(t string) string   { return Bold + t + Reset }

func GetPriorityIcon(priority string) string {
	switch priority {
	case "Highest":
		return Red + Bold + " [▲▲] " + Reset // Double up
	case "High":
		return Red + "  [▲]  " + Reset // Single up
	case "Medium":
		return Yellow + "  [=]  " + Reset // Equal / Neutral
	case "Low":
		return Blue + "  [▼]  " + Reset // Single down
	case "Lowest":
		return Cyan + " [▼▼] " + Reset // Double down
	default:
		return Dim + "  [-]  " + Reset // Unknown
	}
}
