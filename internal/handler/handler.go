// Package handler is responsible of briding the jira server to cli
package handler

import (
	"fmt"
	"os"
	"strings"

	"github.com/aliozdemir13/odincli/internal"
	"github.com/aliozdemir13/odincli/internal/models"
	"github.com/aliozdemir13/odincli/internal/style"
	"github.com/aliozdemir13/odincli/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jedib0t/go-pretty/v6/table"
)

// option preferred to be able to increase testability, it is known that this is not the best option but merely a patch before next refactor
// TODO: refactor it and use Dependency Injection
var (
	RunIssueForm = func() (summary, issueType, effort, parent string, err error) {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Summary").Value(&summary).Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("required")
					}
					return nil
				}),
				huh.NewSelect[string]().Title("Issue Type").
					Options(huh.NewOptions("Task", "Bug", "Story", "Sub-task", "Epic")...).
					Value(&issueType),
			),
			huh.NewGroup(
				huh.NewInput().Title("Parent Key (Optional)").Value(&parent),
				huh.NewInput().Title("Story Points (Optional)").Value(&effort),
			),
		)
		err = form.Run()
		return
	}

	RunDescriptionEditor = func() (content string, aborted bool) {
		p := tea.NewProgram(ui.InitialModel())
		m, _ := p.Run()
		finalModel := m.(ui.EditorModel)
		return finalModel.Content, finalModel.Aborted
	}

	RunCommendEditor = func() string {
		p := tea.NewProgram(ui.InitialModel())
		m, err := p.Run()
		if err != nil {
			fmt.Printf("Error running editor: %v", err)
			return ""
		}

		finalModel := m.(ui.EditorModel)

		if finalModel.Aborted || finalModel.Content == "" {
			fmt.Println(style.Yellow("Comment cancelled."))
			return ""
		}

		fmt.Println(style.Dim("Processing markup\n"))

		return finalModel.Content
	}
)

// HandlePull covers pull action and fetches issues from jira
func HandlePull(parts []string, config models.Config, apiKey string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("pull")
		return false
	}

	projectKey := strings.ToUpper(parts[1])

	proj, ok := config.Projects[projectKey]
	if !ok {
		fmt.Printf(style.Red("Project %s not found in config.json\n"), projectKey)
		return false
	}

	internal.CurrentInstance = models.JiraInstance{
		Name:    projectKey,
		BaseURL: proj.URL,
		Email:   proj.Email,
		Token:   apiKey,
	}

	fmt.Printf(style.Dim("Fetching from %s...\n"), internal.CurrentInstance.BaseURL)
	internal.FetchIssues(fmt.Sprintf("project = %s AND statusCategory != Done", projectKey))

	return true
}

// HandleDetails handles the issue details display via calling service function
func HandleDetails(parts []string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("details")
		return false
	}

	if len(internal.LastEntries) == 0 {
		fmt.Println(style.Red("First, pull the issues."))
		return false
	}

	key := strings.ToUpper(parts[1])
	if e, ok := internal.EntriesCache[key]; ok {
		fmt.Printf("\n%s - %s %s | %s (%s)\n", style.GetPriorityIcon(e.Fields.Priority.Name), style.Green("["+e.Key+"]"), style.Bold(e.Fields.Summary), style.Yellow(e.Fields.Status.StatusCategory.Name), style.Dim(e.Fields.Assignee.Name))
		fmt.Println(strings.Repeat("-", 40))
		if e.Fields.ParsedDescription == "" {
			e.Fields.ParsedDescription = models.ParseADF(e.Fields.Description)
		}
		if e.Fields.ParsedDescription == "" {
			fmt.Println(style.Dim("No description provided."))
		} else {
			fmt.Println(e.Fields.ParsedDescription)
		}
	} else {
		fmt.Println(style.Red("Issue not found in current pull."))
		return false
	}

	fmt.Println(style.Dim(style.Yellow("Fetching comments for " + parts[1] + "...")))
	internal.FetchComments(key)

	return true
}

// HandleAddComment covers the comment addition via markdown editor and jira service
func HandleAddComment(parts []string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("addComment")
		return false
	}

	c := RunCommendEditor()
	if c == "" {
		return false
	}

	fmt.Printf(style.Dim("Posting comment to %s...\n"), parts[1])

	fmt.Printf(style.Dim("Posting comment  %s...\n"), c)
	internal.AddCommentToJira(parts[1], c)

	return true
}

// HandleFilter is the handler for data filtering and displaying filter results in a table grid
func HandleFilter(parts []string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("filter")
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
		style.PrintCommandUsage("filter")
		return false
	}
	searchKey := strings.ToLower(strings.Trim(params[1], "\""))

	fmt.Printf(style.Dim(style.Yellow("Filtering local results for %s: %s\n")), field, searchKey)

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
			issueKey := style.Green(e.Key)
			summary := e.Fields.Summary
			if e.Fields.IssueType.Name == "Epic" {
				issueKey = style.Indigo(e.Key)
				summary = style.Indigo(style.Bold(summary))
			}

			assignee := e.Fields.Assignee.Name
			if assignee == "" {
				assignee = "Unassigned"
			}

			t.AppendRow(table.Row{
				style.GetPriorityIcon(e.Fields.Priority.Name),
				issueKey,
				summary,
				style.Yellow(e.Fields.Status.Name),
				style.Dim(assignee),
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
		fmt.Println(style.Red("No matching issues found in last pull."))
	}

	return true
}

func handleEpicsFilter() bool {
	found := false
	for _, e := range internal.LastEntries {
		if strings.ToLower(e.Fields.IssueType.Name) == "epic" {
			fmt.Printf("\n%s - %s %s | %s\n", style.GetPriorityIcon(e.Fields.Priority.Name), style.Indigo("["+e.Key+"]"), style.Indigo(style.Bold(e.Fields.Summary)), style.Yellow(e.Fields.Status.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		}
	}
	if !found {
		fmt.Println(style.Red("No Epics found in last pull."))
		return false
	}
	return true
}

func handleMyIssues() bool {
	found := false
	for _, e := range internal.LastEntries {
		if strings.EqualFold(e.Fields.Assignee.EmailAddress, internal.CurrentInstance.Email) {
			issueKey := style.Green("[" + e.Key + "]")
			issueSummary := style.Bold(e.Fields.Summary)
			if e.Fields.IssueType.Name == "Epic" {
				issueKey = style.Indigo("[" + e.Key + "]")
				issueSummary = style.Indigo(style.Bold(e.Fields.Summary))
			}
			fmt.Printf("\n%s - %s %s | %s (%s)\n", style.GetPriorityIcon(e.Fields.Priority.Name), issueKey, issueSummary, style.Yellow(e.Fields.Status.Name), style.Dim(e.Fields.Assignee.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		}
	}
	if !found {
		fmt.Println(style.Red("Issue not found in last pull."))
		return false
	}
	return true
}

// HandleStatus handles the status change and fetching available transitions to display options
func HandleStatus(parts []string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("status")
		return false
	}
	issueKey := strings.ToUpper(parts[1])

	// Fetch potential values
	fmt.Println(style.Dim("Fetching valid transitions for " + issueKey + "..."))
	transitions, err := internal.GetAvailableTransitions(issueKey)
	if err != nil || len(transitions) == 0 {
		fmt.Println(style.Red("Could not fetch transitions or no moves available."))
		return false
	}

	// Display the "Menu"
	fmt.Println(style.Bold("\nSelect new status:"))
	for i, t := range transitions {
		fmt.Printf("%d) %s\n", i+1, style.Yellow(t.Name))
	}
	fmt.Print(style.Cyan("Choose option (or 'c' to cancel): "))

	// Get user selection
	var choice string
	_, _ = fmt.Scanln(&choice)

	if choice == "c" {
		return false
	}

	// Convert string choice to index
	idx := 0
	_, _ = fmt.Sscanf(choice, "%d", &idx)
	if idx < 1 || idx > len(transitions) {
		fmt.Println(style.Red("Invalid selection."))
		return false
	}

	selected := transitions[idx-1]
	err = internal.PerformTransition(issueKey, selected.ID)
	if err != nil {
		fmt.Println(style.Red("Update failed: ") + err.Error())
	} else {
		fmt.Println(style.Green("✔ Status changed to " + selected.Name))
	}
	return true
}

// HandleAssign covers the issue assignment logic
func HandleAssign(parts []string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("assign")
		return false
	}

	key := strings.ToUpper(parts[1])

	if issue, ok := internal.EntriesCache[key]; ok {
		internal.AssignInteractive(issue.Key)
	} else {
		fmt.Println(style.Red("Issue not found in current pull."))
	}
	return true
}

// HandleSearch handles the text search in jira using the service action
func HandleSearch(parts []string) bool {
	if len(parts) < 2 {
		style.PrintCommandUsage("search")
		return false
	}

	// Remove quotes if the user provided them
	keyword := strings.Trim(parts[1], "\"")

	// Construct the JQL for keyword search
	// text ~ "keyword" searches summary, description, and comments
	jql := fmt.Sprintf("text ~ \"%s\"", keyword)

	fmt.Printf(style.Yellow("🔍 Searching for issues containing: \"%s\"...\n"), keyword)

	// This will clear the current pull and replace it with the search results
	internal.FetchIssues(jql)

	return true
}

// HandleCreateIssue covers the new issue creation using the markdown editor for issue contents
func HandleCreateIssue() bool {

	// Create a multi-step form using function as variable approach
	summary, issueType, effort, parent, err := RunIssueForm()
	if err != nil {
		fmt.Println("Cancelled.")
		return false
	}

	//Launch Bubble Tea Editor for the Description
	fmt.Println(style.Indigo("Opening Editor for Description..."))
	content, aborted := RunDescriptionEditor()
	if aborted {
		fmt.Println(style.Red("Creation cancelled."))
		return false
	}

	// Build Payload
	payload := models.CreateIssueRequest{}
	payload.Fields.Project.Key = internal.CurrentInstance.Name
	payload.Fields.Summary = summary
	payload.Fields.IssueType.Name = issueType
	payload.Fields.Description = models.MarkdownToADF(content)

	if parent != "" {
		payload.Fields.Parent = &models.ProjectReference{Key: strings.ToUpper(parent)}
	}

	err = internal.CreateIssueInJira(payload, effort)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return false
	}
	return true
}
