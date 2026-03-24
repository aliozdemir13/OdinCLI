package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	CurrentInstance JiraInstance
	LastEntries     []Issues
)

func prepareQueryCallout(endpoint string, nextPageToken string, jql string) *http.Request {
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

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Printf("Error parsing request: %s", err)
	}
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)
	req.Header.Set("Accept", "application/json")

	return req
}

func FetchIssues(jql string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(StyleRed("Error: No instance selected. Use 'pull ---{{ProjectKey}}' first."))
		return
	}

	// for fetching the additional issues until the end
	nextPageToken := ""
	// for understanding if the pagination end reached
	isLast := false
	// print out the total issues fetched
	issueCount := 0
	LastEntries = []Issues{}

	for !isLast {
		apiPath := "/rest/api/3/search/jql"

		baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")

		req := prepareQueryCallout(baseUrl+apiPath, nextPageToken, jql)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf(StyleRed("Network Error: %v\n"), err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %s", err)
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Printf(StyleRed("Jira API Error (%d): %s\n"), resp.StatusCode, string(body))
			return
		}

		var apiData JiraResponse
		err = json.Unmarshal(body, &apiData)
		if err != nil {
			fmt.Printf(StyleRed("JSON Parse Error: %v\n"), err)
			return
		}

		if len(apiData.Issues) == 0 {
			fmt.Println(StyleYellow("No issues found for this query."))
			return
		}

		for _, issue := range apiData.Issues {
			LastEntries = append(LastEntries, issue)
			fmt.Printf("%s - [%s] %s | %s (%s)\n",
				//StyleDim(issue.Fields.IssueType.Name),
				GetPriorityIcon(issue.Fields.Priority.Name),
				StyleGreen(issue.Key),
				StyleBold(issue.Fields.Summary),
				StyleYellow(issue.Fields.Status.StatusCategory.Name),
				StyleDim(issue.Fields.Assignee.Name))
			issueCount++
		}
		isLast = apiData.IsLast
		nextPageToken = apiData.NextPageToken
	}

	fmt.Printf(StyleGreen("Successfully pulled %d issues:\n"), issueCount)
}

func ExtractPlainText(desc JiraDescription) string {
	var builder strings.Builder

	for _, node := range desc.Content {
		walkNodes(node, &builder)
	}

	return builder.String()
}

func walkNodes(node DescriptionNode, b *strings.Builder) {
	// Handle direct text
	if node.Text != "" {
		b.WriteString(node.Text)
	}

	// Handle mentions
	if node.Type == "mention" {
		if val, ok := node.Attrs["text"]; ok {
			b.WriteString(StyleBlue(fmt.Sprintf("%v", val)))
		}
	}

	// Handle recursive children
	for _, child := range node.Content {
		walkNodes(child, b)

		// Add newlines for block elements
		if child.Type == "paragraph" || child.Type == "listItem" {
			b.WriteString("\n")
		}
	}
}

func FetchComments(issueKey string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(StyleRed("Error: No instance selected."))
		return
	}

	apiPath := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)

	baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")

	req := prepareQueryCallout(baseUrl+apiPath, "", issueKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf(StyleRed("Request failed: %v\n"), err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf(StyleRed("Error %d: %s\n"), resp.StatusCode, string(body))
		return
	}

	var apiData JiraCommentResponse
	json.Unmarshal(body, &apiData)

	fmt.Printf(StyleBold("\n--- Comments for %s (%d) ---\n"), issueKey, apiData.Total)

	for _, c := range apiData.Comments {
		var plainText string
		plainText = ExtractPlainText(c.Body)

		commentEntry := Comment{
			Id:          c.Id,
			CreatedBy:   c.Author.DisplayName,
			CreatedDate: c.Created,
			UpdatedDate: c.Updated,
			Text:        strings.TrimSpace(plainText),
		}

		statusTag := ""
		if c.Created != c.Updated {
			statusTag = StyleYellow("[edited at " + commentEntry.UpdatedDate + "]")
		}

		fmt.Printf("%s | %s %s\n",
			StyleGreen(commentEntry.CreatedBy),
			StyleDim(commentEntry.CreatedDate),
			statusTag,
		)
		fmt.Printf("%s\n", commentEntry.Text)
		fmt.Println(StyleDim(strings.Repeat("-", 40)))
	}
}

func AddCommentToJira(issueKey string, commentText string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(StyleRed("Error: No instance selected. Pull a project first."))
		return
	}

	// Jira Cloud uses /rest/api/3/issue/{key}/comment
	baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")
	apiPath := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)
	u, err := url.Parse(baseUrl + apiPath)
	if err != nil {
		fmt.Printf("Error parsing url: %s", err)
	}

	// Even a simple text comment must follow the "doc" -> "paragraph" -> "text" nesting
	payload := AddCommentRequest{
		Body: JiraDescription{
			Type:    "doc",
			Version: 1,
			Content: []DescriptionNode{
				{
					Type: "paragraph",
					Content: []DescriptionNode{
						{
							Type: "text",
							Text: commentText,
						},
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %s", err)
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Error parsing request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf(StyleRed("Failed to connect: %v\n"), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated { // 201 Created
		fmt.Println(StyleGreen("✔ Comment added successfully to " + issueKey))
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf(StyleRed("Failed to add comment (%d): %s\n"), resp.StatusCode, string(body))
	}
}

func GetAvailableTransitions(issueKey string) ([]Transition, error) {
	apiPath := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)
	baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")

	req, err := http.NewRequest("GET", baseUrl+apiPath, nil)
	if err != nil {
		fmt.Printf("Error parsing request: %s", err)
	}
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data JiraTransitionsResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body: %s", err)
	}
	json.Unmarshal(body, &data)

	return data.Transitions, nil
}

// PerformTransition sends the POST request to move the ticket
func PerformTransition(issueKey string, transitionId string) error {
	apiPath := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)
	baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")

	payload := map[string]interface{}{
		"transition": map[string]string{"id": transitionId},
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %s", err)
	}

	req, err := http.NewRequest("POST", baseUrl+apiPath, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Error parsing request: %s", err)
	}
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Jira returned status %d", resp.StatusCode)
	}
	return nil
}
