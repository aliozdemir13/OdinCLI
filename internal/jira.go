package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	CurrentInstance JiraInstance
	LastEntries     []Issues
	EntriesCache    map[string]Issues
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
		return fmt.Errorf(StyleRed("Network Error: %v"), err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf(StyleRed("Jira Error (%d): %s"), resp.StatusCode, string(body))
	}

	if target != nil {
		err := json.Unmarshal(body, target)
		return err
	}
	return nil
}

func FetchIssues(jql string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(StyleRed("Error: No instance selected. Use 'pull ---{{ProjectKey}}' first."))
		return
	}

	// 1. Initialize the Table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"PRIORITY", "TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE"})

	nextPageToken := ""
	isLast := false
	issueCount := 0
	LastEntries = []Issues{}
	EntriesCache = make(map[string]Issues)

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
		var apiData JiraResponse

		if err := performRequest(req, http.StatusOK, &apiData); err != nil {
			fmt.Println(err)
			return
		}

		if len(apiData.Issues) == 0 && issueCount == 0 {
			fmt.Println(StyleYellow("No issues found for this query."))
			return
		}

		for _, issue := range apiData.Issues {
			LastEntries = append(LastEntries, issue)
			EntriesCache[issue.Key] = issue

			// Prepare Styling
			prioIcon := GetPriorityIcon(issue.Fields.Priority.Name)
			issueType := issue.Fields.IssueType.Name
			issueKey := StyleGreen(issue.Key)
			issueSummary := issue.Fields.Summary
			issueStatus := StyleYellow(issue.Fields.Status.Name)

			assignee := issue.Fields.Assignee.Name
			if assignee == "" {
				assignee = "Unassigned"
			}
			assignee = StyleDim(assignee)

			// Apply Epic styling logic
			if issueType == "Epic" {
				issueKey = StyleIndigo(issue.Key)
				issueSummary = StyleIndigo(StyleBold(issueSummary))
			}

			// 2. Append Row to Table
			t.AppendRow(table.Row{
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

	// 3. Configure Table Style
	style := table.StyleLight
	style.Options.DrawBorder = false
	style.Options.SeparateColumns = false
	style.Options.SeparateHeader = true
	style.Box.PaddingRight = "  "
	t.SetStyle(style)

	// 4. Set column limit for Summary so it doesn't break the terminal
	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "SUMMARY", WidthMax: 60},
	})

	// 5. Render
	fmt.Println() // Add a little breathing room
	t.Render()
	fmt.Printf("\n"+StyleGreen("Successfully pulled %d issues.")+"\n", issueCount)
}

func FetchComments(issueKey string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)
	req, _ := newRequest("GET", apiURL(path), nil)

	var apiData JiraCommentResponse
	if err := performRequest(req, http.StatusOK, &apiData); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(StyleBold("\n--- Comments for %s (%d) ---\n"), issueKey, apiData.Total)
	for _, c := range apiData.Comments {
		commentText := strings.TrimSpace(ExtractPlainText(c.Body))
		statusTag := ""
		if c.Created != c.Updated {
			statusTag = StyleYellow("[edited at " + c.Updated + "]")
		}

		fmt.Printf("%s | %s %s\n", StyleGreen(c.Author.DisplayName), StyleDim(c.Created), statusTag)
		fmt.Printf("%s\n%s\n", commentText, StyleDim(strings.Repeat("-", 40)))
	}
}

func AddCommentToJira(issueKey string, commentText string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)
	payload := AddCommentRequest{
		Body: JiraDescription{
			Type: "doc", Version: 1,
			Content: []DescriptionNode{{
				Type:    "paragraph",
				Content: []DescriptionNode{{Type: "text", Text: commentText}},
			}},
		},
	}

	req, _ := newRequest("POST", apiURL(path), payload)
	if err := performRequest(req, http.StatusCreated, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(StyleGreen("✔ Comment added successfully to %s\n"), issueKey)
}

func GetAvailableTransitions(issueKey string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)
	req, _ := newRequest("GET", apiURL(path), nil)

	var data JiraTransitionsResponse
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

func SearchUsers(query string) ([]JiraUser, error) {
	path := fmt.Sprintf("/rest/api/3/user/search?query=%s&maxResults=5", url.QueryEscape(query))
	req, _ := newRequest("GET", apiURL(path), nil)

	var users []JiraUser
	err := performRequest(req, http.StatusOK, &users)
	return users, err
}

func AssignIssue(issueKey string, accountId string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/assignee", issueKey)
	req, _ := newRequest("PUT", apiURL(path), AssigneePayload{AccountId: accountId})

	if err := performRequest(req, http.StatusNoContent, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(StyleGreen("✔ %s assigned successfully.\n"), issueKey)
}

func FetchEpicChildren(epicKey string) {
	// JQL to find all items belonging to this Epic
	jql := fmt.Sprintf("parent = %s", epicKey)

	// We only need a few fields for the table
	fields := "summary,status,issuetype,priority,assignee"
	path := fmt.Sprintf("/rest/api/3/search/jql?jql=%s&fields=%s&maxResults=100", url.QueryEscape(jql), url.QueryEscape(fields))

	req, _ := newRequest("GET", apiURL(path), nil)
	var apiData JiraResponse
	if err := performRequest(req, http.StatusOK, &apiData); err != nil {
		fmt.Println(err)
		return
	}

	if len(apiData.Issues) == 0 {
		fmt.Println(StyleYellow("No child issues found for this epic."))
		return
	}

	fmt.Printf(StyleBold("\nIssues in Epic %s (%d items):\n\n"),
		StyleIndigo("["+epicKey+"] "+EntriesCache[epicKey].Fields.Summary), len(apiData.Issues))

	// 1. Create the Table Writer
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// 2. Define Headers
	t.AppendHeader(table.Row{"TYPE", "KEY", "SUMMARY", "PRIORITY", "STATUS", "ASSIGNEE"})

	for _, issue := range apiData.Issues {
		// Prepare data
		issueType := issue.Fields.IssueType.Name
		key := StyleGreen(issue.Key)

		summary := issue.Fields.Summary
		// The library handles wrapping, but if you want strict truncation:
		if len(summary) > 40 {
			summary = summary[:37] + "..."
		}

		prio := GetPriorityIcon(issue.Fields.Priority.Name)
		status := StyleYellow(issue.Fields.Status.Name)

		assignee := issue.Fields.Assignee.Name
		if assignee == "" {
			assignee = "Unassigned"
		}
		assignee = StyleDim(assignee)

		// 3. Add Row
		t.AppendRow(table.Row{
			issueType,
			key,
			summary,
			prio,
			status,
			assignee,
		})
	}

	// 4. Style the table to look like your original request
	// We use StyleLight but remove the borders for a "clean" look
	style := table.StyleLight
	style.Options.DrawBorder = false
	style.Options.SeparateColumns = false
	style.Options.SeparateHeader = true
	style.Box.PaddingLeft = ""
	style.Box.PaddingRight = "   " // Matches your spacing
	t.SetStyle(style)

	// 5. Render
	t.Render()
	fmt.Println()
}

func ExtractPlainText(desc JiraDescription) string {
	var builder strings.Builder
	for _, node := range desc.Content {
		walkNodes(node, &builder)
	}
	return builder.String()
}

func walkNodes(node DescriptionNode, b *strings.Builder) {
	if node.Text != "" {
		b.WriteString(node.Text)
	}
	if node.Type == "mention" {
		if val, ok := node.Attrs["text"]; ok {
			b.WriteString(StyleBlue(fmt.Sprintf("%v", val)))
		}
	}
	for _, child := range node.Content {
		walkNodes(child, b)
		if child.Type == "paragraph" || child.Type == "listItem" {
			b.WriteString("\n")
		}
	}
}

func AssignInteractive(issueKey string) {
	fmt.Print(StyleBold("Search user to assign: "))
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
		fmt.Println(StyleRed("No users found matching: " + input))
		return
	}

	bestMatch := users[0]
	recommendation := StyleYellow(bestMatch.DisplayName)
	if strings.HasPrefix(strings.ToLower(bestMatch.DisplayName), strings.ToLower(input)) {
		recommendation = bestMatch.DisplayName[:len(input)] + StyleDim(bestMatch.DisplayName[len(input):])
	}

	fmt.Printf("Match found: %s. Assign %s? (y/n): ", recommendation, StyleGreen(issueKey))
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) == "y" {
		AssignIssue(issueKey, bestMatch.AccountId)
	} else if len(users) > 1 {
		fmt.Println(StyleBold("\nOther matches:"))
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
