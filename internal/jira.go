package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log_tracker/internal/helpers"
	"log_tracker/internal/models"
	"log_tracker/internal/style"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	CurrentInstance models.JiraInstance
	LastEntries     []models.Issues
	EntriesCache    map[string]models.Issues
)

// apiURL combines the base URL with the specific API path
func apiURL(path string) string {
	return strings.TrimSuffix(CurrentInstance.BaseURL, "/") + path
}

// newRequest creates a request with Auth and standard headers
func newRequest(method, path string, bodyData interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if bodyData != nil {
		jsonBytes, _ := json.Marshal(bodyData)
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)
	req.Header.Set("Accept", "application/json")
	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// performRequest executes the request, checks status, and decodes JSON into target
func performRequest(req *http.Request, expectedStatus int, target interface{}) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(style.StyleRed("Network Error: %v"), err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf(style.StyleRed("Jira Error (%d): %s"), resp.StatusCode, string(body))
	}

	if target != nil {
		err := json.Unmarshal(body, target)
		return err
	}
	return nil
}

func FetchIssues(jql string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(style.StyleRed("Error: No instance selected. Use 'pull ---{{ProjectKey}}' first."))
		return
	}

	nextPageToken := ""
	isLast := false
	issueCount := 0
	LastEntries = []models.Issues{}
	EntriesCache = make(map[string]models.Issues)
	// prepare table data
	var tableBody []table.Row

	for !isLast {
		endpoint := apiURL("/rest/api/3/search/jql")
		u, err := url.Parse(endpoint)
		if err != nil {
			fmt.Printf("Error parsing url: %s", err)
			return
		}

		q := u.Query()
		q.Set("jql", jql)
		q.Set("maxResults", "100")
		if nextPageToken != "" {
			q.Set("nextPageToken", nextPageToken)
		}
		q.Set("fields", "summary,description,issuetype,priority,status,assignee,duedate,created")
		u.RawQuery = q.Encode()

		req, _ := newRequest("GET", u.String(), nil)
		var apiData models.JiraResponse

		if err := performRequest(req, http.StatusOK, &apiData); err != nil {
			fmt.Println(err)
			return
		}

		if len(apiData.Issues) == 0 && issueCount == 0 {
			fmt.Println(style.StyleYellow("No issues found for this query."))
			return
		}

		for _, issue := range apiData.Issues {
			LastEntries = append(LastEntries, issue)
			EntriesCache[issue.Key] = issue

			// Prepare Styling
			prioIcon := style.GetPriorityIcon(issue.Fields.Priority.Name)
			issueType := issue.Fields.IssueType.Name
			issueKey := style.StyleGreen(issue.Key)
			issueSummary := issue.Fields.Summary
			issueStatus := style.StyleYellow(issue.Fields.Status.Name)

			assignee := issue.Fields.Assignee.Name
			if assignee == "" {
				assignee = "Unassigned"
			}
			assignee = style.StyleDim(assignee)

			// Apply Epic styling logic
			if issueType == "Epic" {
				issueKey = style.StyleIndigo(issue.Key)
				issueSummary = style.StyleIndigo(style.StyleBold(issueSummary))
			}

			// 2. Append Row to Table
			tableBody = append(tableBody, table.Row{
				prioIcon,
				issueType,
				issueKey,
				issueSummary,
				issueStatus,
				assignee,
			})

			issueCount++
		}
		isLast = apiData.IsLast
		nextPageToken = apiData.NextPageToken
	}

	style.CreateTable(table.Row{"PRIORITY", "TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE"}, tableBody, []table.ColumnConfig{{Name: "SUMMARY", WidthMax: 60}})
	fmt.Printf("\n"+style.StyleGreen("Successfully pulled %d issues.")+"\n", issueCount)
}

func FetchComments(issueKey string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)
	req, _ := newRequest("GET", apiURL(path), nil)

	var apiData models.JiraCommentResponse
	if err := performRequest(req, http.StatusOK, &apiData); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(style.StyleBold("\n--- Comments for %s (%d) ---\n"), issueKey, apiData.Total)
	for _, c := range apiData.Comments {
		commentText := strings.TrimSpace(helpers.ExtractPlainText(c.Body))
		statusTag := ""
		if c.Created != c.Updated {
			statusTag = style.StyleYellow("[edited at " + c.Updated + "]")
		}

		fmt.Printf("%s | %s %s\n", style.StyleGreen(c.Author.DisplayName), style.StyleDim(c.Created), statusTag)
		fmt.Printf("%s\n%s\n", commentText, style.StyleDim(strings.Repeat("-", 40)))
	}
}

func AddCommentToJira(issueKey string, commentText string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)
	payload := models.AddCommentRequest{
		Body: models.JiraDescription{
			Type: "doc", Version: 1,
			Content: []models.DescriptionNode{{
				Type:    "paragraph",
				Content: []models.DescriptionNode{{Type: "text", Text: commentText}},
			}},
		},
	}

	req, _ := newRequest("POST", apiURL(path), payload)
	if err := performRequest(req, http.StatusCreated, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(style.StyleGreen("✔ Comment added successfully to %s\n"), issueKey)
}

func GetAvailableTransitions(issueKey string) ([]models.Transition, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)
	req, _ := newRequest("GET", apiURL(path), nil)

	var data models.JiraTransitionsResponse
	err := performRequest(req, http.StatusOK, &data)
	return data.Transitions, err
}

func PerformTransition(issueKey string, transitionId string) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)
	payload := map[string]interface{}{
		"transition": map[string]string{"id": transitionId},
	}

	req, _ := newRequest("POST", apiURL(path), payload)
	return performRequest(req, http.StatusNoContent, nil)
}

func SearchUsers(query string) ([]models.JiraUser, error) {
	path := fmt.Sprintf("/rest/api/3/user/search?query=%s&maxResults=5", url.QueryEscape(query))
	req, _ := newRequest("GET", apiURL(path), nil)

	var users []models.JiraUser
	err := performRequest(req, http.StatusOK, &users)
	return users, err
}

func AssignIssue(issueKey string, accountId string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/assignee", issueKey)
	req, _ := newRequest("PUT", apiURL(path), models.AssigneePayload{AccountId: accountId})

	if err := performRequest(req, http.StatusNoContent, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(style.StyleGreen("✔ %s assigned successfully.\n"), issueKey)
}

func FetchEpicChildren(epicKey string) {
	// JQL to find all items belonging to this Epic
	jql := fmt.Sprintf("parent = %s", epicKey)

	// We only need a few fields for the table
	fields := "summary,status,issuetype,priority,assignee"
	path := fmt.Sprintf("/rest/api/3/search/jql?jql=%s&fields=%s&maxResults=100", url.QueryEscape(jql), url.QueryEscape(fields))

	req, _ := newRequest("GET", apiURL(path), nil)
	var apiData models.JiraResponse
	if err := performRequest(req, http.StatusOK, &apiData); err != nil {
		fmt.Println(err)
		return
	}

	if len(apiData.Issues) == 0 {
		fmt.Println(style.StyleYellow("No child issues found for this epic."))
		return
	}

	fmt.Printf(style.StyleBold("\nIssues in Epic %s (%d items):\n\n"),
		style.StyleIndigo("["+epicKey+"] "+EntriesCache[epicKey].Fields.Summary), len(apiData.Issues))

	// prepare table data
	var tableBody []table.Row

	for _, issue := range apiData.Issues {
		// Prepare data
		issueType := issue.Fields.IssueType.Name
		key := style.StyleGreen(issue.Key)

		summary := issue.Fields.Summary
		// The library handles wrapping, but if you want strict truncation:
		if len(summary) > 40 {
			summary = summary[:37] + "..."
		}

		prio := style.GetPriorityIcon(issue.Fields.Priority.Name)
		status := style.StyleYellow(issue.Fields.Status.Name)

		assignee := issue.Fields.Assignee.Name
		if assignee == "" {
			assignee = "Unassigned"
		}
		assignee = style.StyleDim(assignee)

		// Add Row
		tableBody = append(tableBody, table.Row{
			issueType,
			key,
			summary,
			prio,
			status,
			assignee,
		})
	}

	style.CreateTable(table.Row{"TYPE", "KEY", "SUMMARY", "PRIORITY", "STATUS", "ASSIGNEE"}, tableBody, nil)
}

func AssignInteractive(issueKey string) {
	fmt.Print(style.StyleBold("Search user to assign: "))
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}
	input := scanner.Text()
	if input == "" {
		return
	}

	users, err := SearchUsers(input)
	if err != nil || len(users) == 0 {
		fmt.Println(style.StyleRed("No users found matching: " + input))
		return
	}

	bestMatch := users[0]
	recommendation := style.StyleYellow(bestMatch.DisplayName)
	if strings.HasPrefix(strings.ToLower(bestMatch.DisplayName), strings.ToLower(input)) {
		recommendation = bestMatch.DisplayName[:len(input)] + style.StyleDim(bestMatch.DisplayName[len(input):])
	}

	fmt.Printf("Match found: %s. Assign %s? (y/n): ", recommendation, style.StyleGreen(issueKey))
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) == "y" {
		AssignIssue(issueKey, bestMatch.AccountId)
	} else if len(users) > 1 {
		fmt.Println(style.StyleBold("\nOther matches:"))
		for i, u := range users {
			fmt.Printf("%d) %s\n", i+1, u.DisplayName)
		}
		fmt.Print("Select number (or 'c' to cancel): ")
		var choice int
		fmt.Scanln(&choice)
		if choice > 0 && choice <= len(users) {
			AssignIssue(issueKey, users[choice-1].AccountId)
		}
	}
}
