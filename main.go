// Package main is the main entry point of the app
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aliozdemir13/odincli/internal"
	"github.com/aliozdemir13/odincli/internal/handler"
	"github.com/aliozdemir13/odincli/internal/models"
	"github.com/aliozdemir13/odincli/internal/style"
	"github.com/joho/godotenv"
)

func main() {
	style.PrintHeader()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))

	configRaw, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Println(style.Red("✘ Error: config.json not found."))
		fmt.Println(style.Dim("Please create a config.json file in the root directory."))
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
		fmt.Print(style.Blue(style.Bold("\nodin waits for your command! > ")))
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
			if !handler.HandlePull(parts, config, apiKey) {
				continue
			}

		case "details": // finds a single issue searched and display description, status, subject, assignee and comments of it
			if strings.HasPrefix(parts[1], "epic ") {
				epicKey := strings.ToUpper(strings.TrimPrefix(parts[1], "epic "))
				internal.FetchEpicChildren(epicKey)
				continue
			}
			if !handler.HandleDetails(parts) {
				continue
			}

		case "addComment": // adding comment to the issue selected
			if !handler.HandleAddComment(parts) {
				continue
			}

		case "filter": // filter issues based on given dimension and value. available dimensions: status, priority
			if !handler.HandleFilter(parts) {
				continue
			}

		case "search": // search issues with keyword
			if !handler.HandleSearch(parts) {
				continue
			}

		case "status": // change status of the issue
			if !handler.HandleStatus(parts) {
				continue
			}

		case "assign": // change assignment of the issue
			if !handler.HandleAssign(parts) {
				continue
			}

		case "create":
			handler.HandleCreateIssue()

		case "exit":
			fmt.Println(style.Blue(style.Bold("Goodbye!")))
			return
		case "help":
			style.PrintCommandList()

		default:
			fmt.Println(style.Red("Unknown command. See menu above."))
		}
	}
}
