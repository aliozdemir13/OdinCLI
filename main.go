package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aliozdemir13/odincli/internal"
	"github.com/aliozdemir13/odincli/internal/models"
	"github.com/aliozdemir13/odincli/internal/style"
	"github.com/aliozdemir13/odincli/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/joho/godotenv"
)

func handlePull(parts []string, config models.Config, apiKey string) bool {
	if len(parts) < 2 {
		printCommandUsage("pull")
		return false
	}

	projectKey := strings.ToUpper(parts[1])

	proj, ok := config.Projects[projectKey]
	if !ok {
		fmt.Printf(style.StyleRed("Project %s not found in config.json\n"), projectKey)
		return false
	}

	internal.CurrentInstance = models.JiraInstance{
		Name:    projectKey,
		BaseURL: proj.URL,
		Email:   proj.Email,
		Token:   apiKey,
	}

	fmt.Printf(style.StyleDim("Fetching from %s...\n"), internal.CurrentInstance.BaseURL)
	internal.FetchIssues(fmt.Sprintf("project = %s AND statusCategory != Done", projectKey))

	return true
}

func handleDetails(parts []string) bool {
	if len(parts) < 2 {
		printCommandUsage("details")
		return false
	}

	if len(internal.LastEntries) == 0 {
		fmt.Println(style.StyleRed("First, pull the issues."))
		return false
	}

	key := strings.ToUpper(parts[1])
	if e, ok := internal.EntriesCache[key]; ok {
		fmt.Printf("\n%s - %s %s | %s (%s)\n", style.GetPriorityIcon(e.Fields.Priority.Name), style.StyleGreen("["+e.Key+"]"), style.StyleBold(e.Fields.Summary), style.StyleYellow(e.Fields.Status.StatusCategory.Name), style.StyleDim(e.Fields.Assignee.Name))
		fmt.Println(strings.Repeat("-", 40))
		if e.Fields.ParsedDescription == "" {
			e.Fields.ParsedDescription = models.ParseADF(e.Fields.Description)
		}
		if e.Fields.ParsedDescription == "" {
			fmt.Println(style.StyleDim("No description provided."))
		} else {
			fmt.Println(e.Fields.ParsedDescription)
		}
	} else {
		fmt.Println(style.StyleRed("Issue not found in current pull."))
		return false
	}

	fmt.Println(style.StyleDim(style.StyleYellow("Fetching comments for " + parts[1] + "...")))
	internal.FetchComments(key)

	return true
}

func handleAddComment(parts []string) bool {
	if len(parts) < 2 {
		printCommandUsage("addComment")
		return false
	}

	c := handleMarkdownEditor()
	if c == "" {
		return false
	}

	fmt.Printf(style.StyleDim("Posting comment to %s...\n"), parts[1])

	fmt.Printf(style.StyleDim("Posting comment  %s...\n"), c)
	internal.AddCommentToJira(parts[1], c)

	return true
}

func handleFilter(parts []string) bool {
	if len(parts) < 2 {
		printCommandUsage("filter")
		return false
	}
	params := strings.SplitN(parts[1], " ", 2)
	field := strings.ToLower(strings.Trim(params[0], "\""))

	// Handle cases that call other functions immediately
	switch field {
	case "myissues":
		return handleMyIssues()
	case "currentsprint":
		internal.FetchIssues(fmt.Sprintf("project = %s AND statusCategory != Done AND sprint in openSprints()", internal.CurrentInstance.Name))
		return true
	case "backlog":
		internal.FetchIssues(fmt.Sprintf("project = %s AND statusCategory != Done AND sprint is EMPTY", internal.CurrentInstance.Name))
		return true
	case "epics":
		handleEpicsFilter()
		return true
	}

	if len(params) < 2 {
		printCommandUsage("filter")
		return false
	}
	searchKey := strings.ToLower(strings.Trim(params[1], "\""))

	fmt.Printf(style.StyleDim(style.StyleYellow("Filtering local results for %s: %s\n")), field, searchKey)

	//Initialize Table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"PRIORITY", "KEY", "SUMMARY", "STATUS", "ASSIGNEE"})

	foundCount := 0
	for _, e := range internal.LastEntries {
		match := false
		if field == "status" && strings.ToLower(e.Fields.Status.Name) == searchKey {
			match = true
		} else if field == "prio" && strings.ToLower(e.Fields.Priority.Name) == searchKey {
			match = true
		}

		if match {
			foundCount++

			// Styling logic
			issueKey := style.StyleGreen(e.Key)
			summary := e.Fields.Summary
			if e.Fields.IssueType.Name == "Epic" {
				issueKey = style.StyleIndigo(e.Key)
				summary = style.StyleIndigo(style.StyleBold(summary))
			}

			assignee := e.Fields.Assignee.Name
			if assignee == "" {
				assignee = "Unassigned"
			}

			t.AppendRow(table.Row{
				style.GetPriorityIcon(e.Fields.Priority.Name),
				issueKey,
				summary,
				style.StyleYellow(e.Fields.Status.Name),
				style.StyleDim(assignee),
			})
		}
	}

	// Render Table if results found
	if foundCount > 0 {
		style := table.StyleLight
		style.Options.DrawBorder = false
		style.Options.SeparateColumns = false
		style.Options.SeparateHeader = true
		style.Box.PaddingRight = "  "
		t.SetStyle(style)

		t.SetColumnConfigs([]table.ColumnConfig{
			{Name: "SUMMARY", WidthMax: 60},
		})

		fmt.Println()
		t.Render()
		fmt.Printf("\nFound %d matches in local cache.\n", foundCount)
	} else {
		fmt.Println(style.StyleRed("No matching issues found in last pull."))
	}

	return true
}

func handleEpicsFilter() bool {
	found := false
	for _, e := range internal.LastEntries {
		if strings.ToLower(e.Fields.IssueType.Name) == "epic" {
			fmt.Printf("\n%s - %s %s | %s\n", style.GetPriorityIcon(e.Fields.Priority.Name), style.StyleIndigo("["+e.Key+"]"), style.StyleIndigo(style.StyleBold(e.Fields.Summary)), style.StyleYellow(e.Fields.Status.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		}
	}
	if !found {
		fmt.Println(style.StyleRed("No Epics found in last pull."))
		return false
	}
	return true
}

func handleMyIssues() bool {
	found := false
	for _, e := range internal.LastEntries {
		if strings.EqualFold(e.Fields.Assignee.EmailAddress, internal.CurrentInstance.Email) {
			issueKey := style.StyleGreen("[" + e.Key + "]")
			issueSummary := style.StyleBold(e.Fields.Summary)
			if e.Fields.IssueType.Name == "Epic" {
				issueKey = style.StyleIndigo("[" + e.Key + "]")
				issueSummary = style.StyleIndigo(style.StyleBold(e.Fields.Summary))
			}
			fmt.Printf("\n%s - %s %s | %s (%s)\n", style.GetPriorityIcon(e.Fields.Priority.Name), issueKey, issueSummary, style.StyleYellow(e.Fields.Status.Name), style.StyleDim(e.Fields.Assignee.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		}
	}
	if !found {
		fmt.Println(style.StyleRed("Issue not found in last pull."))
		return false
	}
	return true
}

func handleStatus(parts []string) bool {
	if len(parts) < 2 {
		printCommandUsage("status")
		return false
	}
	issueKey := strings.ToUpper(parts[1])

	// Fetch potential values
	fmt.Println(style.StyleDim("Fetching valid transitions for " + issueKey + "..."))
	transitions, err := internal.GetAvailableTransitions(issueKey)
	if err != nil || len(transitions) == 0 {
		fmt.Println(style.StyleRed("Could not fetch transitions or no moves available."))
		return false
	}

	// Display the "Menu"
	fmt.Println(style.StyleBold("\nSelect new status:"))
	for i, t := range transitions {
		fmt.Printf("%d) %s\n", i+1, style.StyleYellow(t.Name))
	}
	fmt.Print(style.StyleCyan("Choose option (or 'c' to cancel): "))

	// Get user selection
	var choice string
	fmt.Scanln(&choice)

	if choice == "c" {
		return false
	}

	// Convert string choice to index
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx < 1 || idx > len(transitions) {
		fmt.Println(style.StyleRed("Invalid selection."))
		return false
	}

	selected := transitions[idx-1]
	err = internal.PerformTransition(issueKey, selected.Id)
	if err != nil {
		fmt.Println(style.StyleRed("Update failed: ") + err.Error())
	} else {
		fmt.Println(style.StyleGreen("✔ Status changed to " + selected.Name))
	}
	return true
}

func handleAssign(parts []string) bool {
	if len(parts) < 2 {
		printCommandUsage("assign")
		return false
	}

	key := strings.ToUpper(parts[1])

	if issue, ok := internal.EntriesCache[key]; ok {
		internal.AssignInteractive(issue.Key)
	} else {
		fmt.Println(style.StyleRed("Issue not found in current pull."))
	}
	return true
}

func handleSearch(parts []string) bool {
	if len(parts) < 2 {
		printCommandUsage("search")
		return false
	}

	// Remove quotes if the user provided them
	keyword := strings.Trim(parts[1], "\"")

	// Construct the JQL for keyword search
	// text ~ "keyword" searches summary, description, and comments
	jql := fmt.Sprintf("text ~ \"%s\"", keyword)

	fmt.Printf(style.StyleYellow("🔍 Searching for issues containing: \"%s\"...\n"), keyword)

	// This will clear the current pull and replace it with the search results
	internal.FetchIssues(jql)

	return true
}

func handleMarkdownEditor() string {
	p := tea.NewProgram(ui.InitialModel())
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error running editor: %v", err)
		return ""
	}

	finalModel := m.(ui.EditorModel)

	if finalModel.Aborted || finalModel.Content == "" {
		fmt.Println(style.StyleYellow("Comment cancelled."))
		return ""
	}

	fmt.Println(style.StyleDim("Processing markup\n"))

	return finalModel.Content
}

func handleCreateIssue() bool {
	var (
		summary   string
		issueType string
		effort    string
		parent    string
	)

	// Create a multi-step form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Summary").
				Value(&summary).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("Summary is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Issue Type").
				Options(huh.NewOptions("Task", "Bug", "Story", "Sub-task", "Epic")...).
				Value(&issueType),
		),
		huh.NewGroup(
			huh.NewInput().Title("Parent Key (Optional)").Value(&parent),
			huh.NewInput().Title("Story Points (Optional)").Value(&effort),
		),
	)

	err := form.Run()
	if err != nil {
		fmt.Println("Cancelled.")
		return false
	}

	//Launch Bubble Tea Editor for the Description
	fmt.Println(style.StyleIndigo("Opening Editor for Description..."))
	p := tea.NewProgram(ui.InitialModel())
	m, _ := p.Run()
	finalModel := m.(ui.EditorModel)

	if finalModel.Aborted {
		fmt.Println(style.StyleRed("Creation cancelled."))
		return false
	}

	// Build Payload
	payload := models.CreateIssueRequest{}
	payload.Fields.Project.Key = internal.CurrentInstance.Name
	payload.Fields.Summary = summary
	payload.Fields.IssueType.Name = issueType
	payload.Fields.Description = models.MarkdownToADF(finalModel.Content)

	if parent != "" {
		payload.Fields.Parent = &models.ProjectReference{Key: strings.ToUpper(parent)}
	}

	internal.CreateIssueInJira(payload, effort)
	return true
}

func printCommandUsage(name string) {
	fmt.Println("Usage:")
	switch name {
	case "filter":
		fmt.Println(style.StyleYellow("  filter ---status In Progress"))
		fmt.Println(style.StyleYellow("  filter ---prio High"))
		fmt.Println(style.StyleYellow("  filter ---myIssues"))
		fmt.Println(style.StyleYellow("  filter ---currentSprint"))
		fmt.Println(style.StyleYellow("  filter ---backlog"))
		fmt.Println(style.StyleYellow("  filter ---epics"))

	case "pull":
		fmt.Println(style.StyleYellow("  pull ---{{ProjectKey}}"))

	case "details":
		fmt.Println(style.StyleYellow("  details ---{{Key}}"))
		fmt.Println(style.StyleYellow("  details ---epic {{Key}}"))

	case "addComment":
		fmt.Println(style.StyleYellow("  addComment {{Key}}"))

	case "status":
		fmt.Println(style.StyleYellow("  status ---{{KEY}}"))

	case "assign":
		fmt.Println(style.StyleYellow("  assign ---{{KEY}}"))

	case "search":
		fmt.Println(style.StyleYellow("  search ---\"your keyword or phrase\""))
	}
}

func main() {
	style.PrintHeader()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))

	configRaw, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Println(style.StyleRed("✘ Error: config.json not found."))
		fmt.Println(style.StyleDim("Please create a config.json file in the root directory."))
		return
	}

	var config models.Config

	errUnmarshal := json.Unmarshal(configRaw, &config)
	if errUnmarshal != nil {
		fmt.Printf("Error unmarshaling: %v\n", errUnmarshal)
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(style.StyleBlue(style.StyleBold("\nodin waits for your command! > ")))
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ---", 3)
		cmd := parts[0]

		switch cmd {
		case "pull": // pulls the issues from JIRA instance selected
			if !handlePull(parts, config, apiKey) {
				continue
			}

		case "details": // finds a single issue searched and display description, status, subject, assignee and comments of it
			if strings.HasPrefix(parts[1], "epic ") {
				epicKey := strings.ToUpper(strings.TrimPrefix(parts[1], "epic "))
				internal.FetchEpicChildren(epicKey)
				continue
			} else {
				if !handleDetails(parts) {
					continue
				}
			}

		case "addComment": // adding comment to the issue selected
			if !handleAddComment(parts) {
				continue
			}

		case "filter": // filter issues based on given dimension and value. available dimensions: status, priority
			if !handleFilter(parts) {
				continue
			}

		case "search": // search issues with keyword
			if !handleSearch(parts) {
				continue
			}

		case "status": // change status of the issue
			if !handleStatus(parts) {
				continue
			}

		case "assign": // change assignment of the issue
			if !handleAssign(parts) {
				continue
			}

		case "create":
			handleCreateIssue()

		case "exit":
			fmt.Println(style.StyleBlue(style.StyleBold("Goodbye!")))
			return
		case "help":
			style.PrintCommandList()

		default:
			fmt.Println(style.StyleRed("Unknown command. See menu above."))
		}
	}
}
