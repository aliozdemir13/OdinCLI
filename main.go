package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"log_tracker/internal"

	"github.com/joho/godotenv"
)

// TODO - clean up the main, it is too messy
func main() {
	internal.PrintHeader()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	emailAddress := strings.TrimSpace(os.Getenv("EMAIL_ADDRESS"))
	instanceOne := strings.TrimSpace(os.Getenv("INSTANCE_URL_ONE"))
	instanceTwo := strings.TrimSpace(os.Getenv("INSTANCE_URL_TWO"))
	projectKeyOne := strings.TrimSpace(os.Getenv("PROJECT_KEY_ONE"))
	projectKeyTwo := strings.TrimSpace(os.Getenv("PROJECT_KEY_TWO"))

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
		case "pull":
			if len(parts) < 2 {
				fmt.Println("Usage: pull ---{{ProjectKey}}")
				continue
			}
			projectKey := strings.ToUpper(parts[1])

			// SET INSTANCE CONFIG HERE
			// TODO create a config file
			switch projectKey {
			case projectKeyOne:
				internal.CurrentInstance = internal.JiraInstance{
					Name:    projectKeyOne,
					BaseURL: instanceOne,
					Email:   emailAddress,
					Token:   apiKey,
				}
			case projectKeyTwo:
				internal.CurrentInstance = internal.JiraInstance{
					Name:    projectKeyTwo,
					BaseURL: instanceTwo,
					Email:   emailAddress,
					Token:   apiKey,
				}
			default:
				fmt.Printf(internal.StyleRed("Project %s not configured in switch case.\n"), projectKey)
				continue
			}

			fmt.Printf(internal.StyleDim("Fetching from %s...\n"), internal.CurrentInstance.BaseURL)
			internal.FetchIssues(fmt.Sprintf("project = %s AND statusCategory != Done", projectKey))

		case "details":
			if len(parts) < 2 {
				fmt.Println("Usage: details ---{{Key}}")
				continue
			}

			if len(internal.LastEntries) == 0 {
				fmt.Println(internal.StyleRed("First, pull the issues."))
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
			}

			fmt.Println(internal.StyleDim(internal.StyleYellow("Fetching comments for " + parts[1] + "...")))
			internal.FetchComments(key)

		case "addComment":
			if len(parts) < 2 {
				fmt.Println("Usage: addComment {{Key}} \"Your comment\"")
				continue
			}
			params := strings.SplitN(parts[1], " ", 2)
			if len(params) < 2 {
				fmt.Println("Usage: addComment {{Key}} \"Your comment\"")
				continue
			}

			fmt.Printf(internal.StyleDim("Posting comment to %s...\n"), params[0])

			fmt.Printf(internal.StyleDim("Posting comment to %s...\n"), params[1])
			internal.AddCommentToJira(params[0], params[1])

		case "filter":
			if len(parts) < 2 {
				fmt.Println("Usage: filter ---status In Progress")
				fmt.Println("Usage: filter ---prio High")
				continue
			}
			params := strings.SplitN(parts[1], " ", 2)
			if len(params) < 2 {
				fmt.Println("Usage: filter ---status In Progress")
				fmt.Println("Usage: filter ---prio High")
				continue
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

		case "search":
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

		case "myIssues":
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
			}
		case "status":
			if len(parts) < 2 {
				fmt.Println(internal.StyleRed("Usage: status ---{{KEY}}"))
				continue
			}
			issueKey := strings.ToUpper(parts[1])

			// Fetch potential values
			fmt.Println(internal.StyleDim("Fetching valid transitions for " + issueKey + "..."))
			transitions, err := internal.GetAvailableTransitions(issueKey)
			if err != nil || len(transitions) == 0 {
				fmt.Println(internal.StyleRed("Could not fetch transitions or no moves available."))
				continue
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
				continue
			}

			// Convert string choice to index
			idx := 0
			fmt.Sscanf(choice, "%d", &idx)
			if idx < 1 || idx > len(transitions) {
				fmt.Println(internal.StyleRed("Invalid selection."))
				continue
			}

			selected := transitions[idx-1]
			err = internal.PerformTransition(issueKey, selected.Id)
			if err != nil {
				fmt.Println(internal.StyleRed("Update failed: ") + err.Error())
			} else {
				fmt.Println(internal.StyleGreen("✔ Status changed to " + selected.Name))
			}

		case "assign":
			if len(parts) < 2 {
				fmt.Println(internal.StyleYellow("Usage: assign ---{{KEY}}"))
				continue
			}

			key := strings.ToUpper(parts[1])

			if issue, ok := internal.EntriesCache[key]; ok {
				internal.AssignInteractive(issue.Key)
			} else {
				fmt.Println(internal.StyleRed("Issue not found in current pull."))
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
