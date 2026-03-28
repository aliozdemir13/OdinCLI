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

	nextPageToken := ""
	isLast := false
	issueCount := 0
	LastEntries = []Issues{}
	EntriesCache = make(map[string]Issues)

	for !isLast {
		// Build the search path with query params
		endpoint := apiURL("/rest/api/3/search/jql")
		u, err := url.Parse(endpoint)
		if err != nil {
			fmt.Printf("Error parsing url: %s", err)
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
			fmt.Printf("%s - [%s] %s | %s (%s)\n",
				GetPriorityIcon(issue.Fields.Priority.Name),
				StyleGreen(issue.Key),
				StyleBold(issue.Fields.Summary),
				StyleYellow(issue.Fields.Status.Name),
				StyleDim(issue.Fields.Assignee.Name))
			issueCount++
		}
		isLast = apiData.IsLast
		nextPageToken = apiData.NextPageToken
	}
	fmt.Printf(StyleGreen("Successfully pulled %d issues.\n"), issueCount)
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
	req, _ := newRequest("GET", path, nil)

	var users []JiraUser
	err := performRequest(req, http.StatusOK, &users)
	return users, err
}

func AssignIssue(issueKey string, accountId string) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/assignee", issueKey)
	req, _ := newRequest("PUT", path, AssigneePayload{AccountId: accountId})

	if err := performRequest(req, http.StatusNoContent, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(StyleGreen("✔ %s assigned successfully.\n"), issueKey)
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
