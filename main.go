package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"log_tracker/internal"

	"github.com/joho/godotenv"
)

func handlePull(parts []string, config internal.Config, apiKey string) bool {
	if len(parts) < 2 {
		fmt.Println(internal.StyleYellow("Usage: pull ---{{ProjectKey}}"))
		return false
	}

	projectKey := strings.ToUpper(parts[1])

	proj, ok := config.Projects[projectKey]
	if !ok {
		fmt.Printf(internal.StyleRed("Project %s not found in config.json\n"), projectKey)
		return false
	}

	internal.CurrentInstance = internal.JiraInstance{
		Name:    projectKey,
		BaseURL: proj.URL,
		Email:   proj.Email,
		Token:   apiKey,
	}

	fmt.Printf(internal.StyleDim("Fetching from %s...\n"), internal.CurrentInstance.BaseURL)
	internal.FetchIssues(fmt.Sprintf("project = %s AND statusCategory != Done", projectKey))

	return true
}

func handleDetails(parts []string) bool {
	if len(parts) < 2 {
		fmt.Println("Usage: details ---{{Key}}")
		return false
	}

	if len(internal.LastEntries) == 0 {
		fmt.Println(internal.StyleRed("First, pull the issues."))
		return false
	}

	key := strings.ToUpper(parts[1])
	if e, ok := internal.EntriesCache[key]; ok {
		fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
		fmt.Println(strings.Repeat("-", 40))
		if e.Fields.ParsedDescription == "" {
			e.Fields.ParsedDescription = internal.ExtractPlainText(e.Fields.Description)
		}
		if e.Fields.ParsedDescription == "" {
			fmt.Println(internal.StyleDim("No description provided."))
		} else {
			fmt.Println(e.Fields.ParsedDescription)
		}
	} else {
		fmt.Println(internal.StyleRed("Issue not found in current pull."))
		return false
	}

	fmt.Println(internal.StyleDim(internal.StyleYellow("Fetching comments for " + parts[1] + "...")))
	internal.FetchComments(key)

	return true
}

func handleAddComment(parts []string) bool {
	if len(parts) < 2 {
		fmt.Println("Usage: addComment {{Key}} \"Your comment\"")
		return false
	}
	params := strings.SplitN(parts[1], " ", 2)
	if len(params) < 2 {
		fmt.Println("Usage: addComment {{Key}} \"Your comment\"")
		return false
	}

	fmt.Printf(internal.StyleDim("Posting comment to %s...\n"), params[0])

	fmt.Printf(internal.StyleDim("Posting comment to %s...\n"), params[1])
	internal.AddCommentToJira(params[0], params[1])

	return true
}

func handleFilter(parts []string) bool {
	if len(parts) < 2 {
		fmt.Println("Usage: filter ---status In Progress")
		fmt.Println("Usage: filter ---prio High")
		return false
	}
	params := strings.SplitN(parts[1], " ", 2)
	if len(params) < 2 {
		fmt.Println("Usage: filter ---status In Progress")
		fmt.Println("Usage: filter ---prio High")
		return false
	}

	searchKey := strings.ToLower(strings.Trim(params[1], "\""))
	field := strings.ToLower(strings.Trim(params[0], "\""))
	fmt.Printf(internal.StyleDim(internal.StyleYellow("Filtering local results for %s: %s\n")), field, searchKey)

	found := false
	for _, e := range internal.LastEntries {
		if field == "status" && strings.ToLower(e.Fields.Status.StatusCategory.Name) == searchKey {
			fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		} else if field == "prio" && strings.ToLower(e.Fields.Priority.Name) == searchKey {
			fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		}
	}
	if !found {
		fmt.Println(internal.StyleRed("Issue not found in last pull."))
	}

	return true
}

func handleMyIssues() bool {
	found := false
	for _, e := range internal.LastEntries {
		if strings.EqualFold(e.Fields.Assignee.EmailAddress, internal.CurrentInstance.Email) {
			fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
			fmt.Println(strings.Repeat("-", 40))
			found = true
		}
	}
	if !found {
		fmt.Println(internal.StyleRed("Issue not found in last pull."))
		return false
	}
	return true
}

func handleStatus(parts []string) bool {
	if len(parts) < 2 {
		fmt.Println(internal.StyleRed("Usage: status ---{{KEY}}"))
		return false
	}
	issueKey := strings.ToUpper(parts[1])

	// Fetch potential values
	fmt.Println(internal.StyleDim("Fetching valid transitions for " + issueKey + "..."))
	transitions, err := internal.GetAvailableTransitions(issueKey)
	if err != nil || len(transitions) == 0 {
		fmt.Println(internal.StyleRed("Could not fetch transitions or no moves available."))
		return false
	}

	// Display the "Menu"
	fmt.Println(internal.StyleBold("\nSelect new status:"))
	for i, t := range transitions {
		fmt.Printf("%d) %s\n", i+1, internal.StyleYellow(t.Name))
	}
	fmt.Print(internal.StyleCyan("Choose option (or 'c' to cancel): "))

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
		fmt.Println(internal.StyleRed("Invalid selection."))
		return false
	}

	selected := transitions[idx-1]
	err = internal.PerformTransition(issueKey, selected.Id)
	if err != nil {
		fmt.Println(internal.StyleRed("Update failed: ") + err.Error())
	} else {
		fmt.Println(internal.StyleGreen("✔ Status changed to " + selected.Name))
	}
	return true
}

func handleAssign(parts []string) bool {
	if len(parts) < 2 {
		fmt.Println(internal.StyleYellow("Usage: assign ---{{KEY}}"))
		return false
	}

	key := strings.ToUpper(parts[1])

	if issue, ok := internal.EntriesCache[key]; ok {
		internal.AssignInteractive(issue.Key)
	} else {
		fmt.Println(internal.StyleRed("Issue not found in current pull."))
	}
	return true
}

// TODO - clean up the main, it is too messy
func main() {
	internal.PrintHeader()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))

	configRaw, err := os.ReadFile("config.json")
	if err != nil {
		panic(fmt.Sprintf("Error reading file: %s", err))
	}

	var config internal.Config

	errUnmarshal := json.Unmarshal(configRaw, &config)
	if errUnmarshal != nil {
		fmt.Printf("Error unmarshaling: %v\n", errUnmarshal)
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(internal.StyleBlue(internal.StyleBold("\nodin waits for your command! > ")))
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
			if !handleDetails(parts) {
				continue
			}

		case "addComment": // adding comment to the issue selected
			if !handleAddComment(parts) {
				continue
			}

		case "filter": // filter issues based on given dimension and value. available dimensions: status, priority
			if !handleFilter(parts) {
				continue
			}

		case "search": // TODO -> replace with actual text search
			if len(parts) < 2 {
				fmt.Println("Usage: search ---{{Code-123}}")
				continue
			}
			searchKey := strings.ToLower(strings.Trim(parts[1], "\""))

			if e, ok := internal.EntriesCache[searchKey]; ok {
				fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
				fmt.Println(strings.Repeat("-", 40))
				if e.Fields.ParsedDescription == "" {
					e.Fields.ParsedDescription = internal.ExtractPlainText(e.Fields.Description)
				}
				if e.Fields.ParsedDescription == "" {
					fmt.Println(internal.StyleDim("No description provided."))
				} else {
					fmt.Println(e.Fields.ParsedDescription)
				}
			} else {
				fmt.Println(internal.StyleRed("Issue not found in current pull."))
			}

			fmt.Println(internal.StyleDim(internal.StyleYellow("Fetching comments for " + parts[1] + "...")))
			internal.FetchComments(searchKey)

		case "myIssues": // display issues assigned to the configured user
			if !handleMyIssues() {
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

		case "exit":
			fmt.Println(internal.StyleBlue(internal.StyleBold("Goodbye!")))
			return
		case "help":
			internal.PrintCommandList()

		default:
			fmt.Println(internal.StyleRed("Unknown command. See menu above."))
		}
	}
}
