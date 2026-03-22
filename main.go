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
			err := godotenv.Load()
			if err != nil {
				log.Fatal("Error loading .env file")
			}
			apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
			emailAddress := strings.TrimSpace(os.Getenv("EMAIL_ADDRESS"))
			instanceOne := strings.TrimSpace(os.Getenv("INSTANCE_URL_ONE"))
			instanceTwo := strings.TrimSpace(os.Getenv("INSTANCE_URL_TWO"))

			// SET INSTANCE CONFIG HERE
			// TODO create a config file
			switch projectKey {
			case "ONE":
				internal.CurrentInstance = internal.JiraInstance{
					Name:    "ONE",
					BaseURL: instanceOne,
					Email:   emailAddress,
					Token:   apiKey,
				}
			case "TWO":
				internal.CurrentInstance = internal.JiraInstance{
					Name:    "TWO",
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
			found := false
			for _, e := range internal.LastEntries {
				if e.Key == key {
					fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
					fmt.Println(strings.Repeat("-", 40))
					if e.Fields.ParsedDescription == "" {
						e.Fields.ParsedDescription = internal.ExtractPlainText(e.Fields.Description)
					}
					if e.Fields.ParsedDescription == "" {
						fmt.Println(internal.StyleDim("No description provided."))
					} else {
						fmt.Println(e.Fields.ParsedDescription) // Description is already plain text from mapping
					}
					found = true
					break
				}
			}

			fmt.Println(internal.StyleDim(internal.StyleYellow("Fetching comments for " + parts[1] + "...")))
			internal.FetchComments(key)

			if !found {
				fmt.Println(internal.StyleRed("Issue not found in last pull."))
			}

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
				fmt.Println("Usage: filter ---In Progress")
				continue
			}
			status := strings.ToLower(strings.Trim(parts[1], "\""))
			fmt.Printf(internal.StyleDim(internal.StyleYellow("Filtering local results for status: %s\n")), status)

			found := false
			for _, e := range internal.LastEntries {
				if strings.ToLower(e.Fields.Status.StatusCategory.Name) == status {
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

			found := false
			for _, e := range internal.LastEntries {
				if strings.ToLower(e.Key) == searchKey {
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
					found = true
					break
				}
			}

			fmt.Println(internal.StyleDim(internal.StyleYellow("Fetching comments for " + parts[1] + "...")))
			internal.FetchComments(searchKey)

			if !found {
				fmt.Println(internal.StyleRed("Issue not found in last pull."))
			}

		case "myIssues":
			found := false
			for _, e := range internal.LastEntries {
				if e.Fields.Assignee.EmailAddress == internal.CurrentInstance.Email {
					fmt.Printf("\n%s - %s %s | %s (%s)\n", internal.GetPriorityIcon(e.Fields.Priority.Name), internal.StyleGreen("["+e.Key+"]"), internal.StyleBold(e.Fields.Summary), internal.StyleYellow(e.Fields.Status.StatusCategory.Name), internal.StyleDim(e.Fields.Assignee.Name))
					fmt.Println(strings.Repeat("-", 40))
					found = true
				}
			}
			if !found {
				fmt.Println(internal.StyleRed("Issue not found in last pull."))
			}
		case "exit":
			fmt.Println(internal.StyleBlue(internal.StyleBold("Goodbye!")))
			return
		case "help":
			fmt.Println(internal.StyleDim("\nUsage: pull ---{{ProjectKey}}"))
			fmt.Println(internal.StyleDim("Usage: details ---{{IssueKey}}"))
			fmt.Println(internal.StyleDim("Usage: search ---{{IssueKey}}"))
			fmt.Println(internal.StyleDim("Usage: filter ---{{Status}}"))
			fmt.Println(internal.StyleDim("Usage: addComment ---{{IssueKey}} text for the comment"))
			fmt.Println(internal.StyleDim("Usage: myIssues"))
			fmt.Println(internal.StyleDim("Usage: exit"))
			fmt.Println(internal.StyleDim("Usage: help"))

		default:
			fmt.Println(internal.StyleRed("Unknown command. See menu above."))
		}
	}
}
