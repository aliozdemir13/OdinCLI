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

func FetchIssues(jql string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(StyleRed("Error: No instance selected. Use 'pull ---{{ProjectKey}}' first."))
		return
	}

	// 1. Prepare URL
	apiPath := "/rest/api/3/search/jql"

	baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")
	fullUrl := baseUrl + apiPath

	u, _ := url.Parse(fullUrl)
	q := u.Query()
	q.Set("jql", jql)

	q.Set("maxResults", "100")
	q.Set("startAt", "0")

	q.Set("fields", "summary,description,issuetype,priority,status,assignee,duedate,created")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf(StyleRed("Network Error: %v\n"), err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

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

	LastEntries = []Issues{}
	if len(apiData.Issues) == 0 {
		fmt.Println(StyleYellow("No issues found for this query."))
		return
	}

	fmt.Printf(StyleGreen("Successfully pulled %d issues:\n"), len(apiData.Issues))
	for _, issue := range apiData.Issues {
		LastEntries = append(LastEntries, issue)
		fmt.Printf("%s - [%s] %s | %s (%s)\n",
			//StyleDim(issue.Fields.IssueType.Name),
			GetPriorityIcon(issue.Fields.Priority.Name),
			StyleGreen(issue.Key),
			StyleBold(issue.Fields.Summary),
			StyleYellow(issue.Fields.Status.StatusCategory.Name),
			StyleDim(issue.Fields.Assignee.Name))
	}
}

func GetPriorityIcon(priority string) string {
	switch priority {
	case "Highest":
		return Red + Bold + " [▲▲] " + Reset // Double up
	case "High":
		return Red + "  [▲]  " + Reset // Single up
	case "Medium":
		return Yellow + "  [=]  " + Reset // Equal / Neutral
	case "Low":
		return Blue + "  [▼]  " + Reset // Single down
	case "Lowest":
		return Cyan + " [▼▼] " + Reset // Double down
	default:
		return Dim + "  [-]  " + Reset // Unknown
	}
}

// ExtractPlainText helper remains the same as your previous version
func ExtractPlainText(desc JiraDescription) string {
	var text string
	for _, node := range desc.Content {
		text += parseNode(node)
	}
	return text
}

func parseNode(node DescriptionNode) string {
	var res string

	if node.Text != "" {
		res += node.Text
	}

	if node.Type == "mention" {
		if val, ok := node.Attrs["text"]; ok {
			res += StyleBlue(fmt.Sprintf("%v", val)) // Using your Cyan style for mentions
		}
	}

	for _, child := range node.Content {
		res += parseNode(child)
		if child.Type == "paragraph" || child.Type == "listItem" {
			res += "\n"
		}
	}

	return res
}

func FetchComments(issueKey string) {
	if CurrentInstance.BaseURL == "" {
		fmt.Println(StyleRed("Error: No instance selected."))
		return
	}

	apiPath := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)

	baseUrl := strings.TrimSuffix(CurrentInstance.BaseURL, "/")
	u, _ := url.Parse(baseUrl + apiPath)

	q := u.Query()
	q.Set("maxResults", "50")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf(StyleRed("Request failed: %v\n"), err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf(StyleRed("Error %d: %s\n"), resp.StatusCode, string(body))
		return
	}

	var apiData JiraCommentResponse
	json.Unmarshal(body, &apiData)

	fmt.Printf(StyleBold("\n--- Comments for %s (%d) ---\n"), issueKey, apiData.Total)

	for _, c := range apiData.Comments {
		var plainText string
		var desc JiraDescription
		descBytes, _ := json.Marshal(c.Body)
		json.Unmarshal(descBytes, &desc)
		plainText = ExtractPlainText(desc)

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
	u, _ := url.Parse(baseUrl + apiPath)

	// Even a simple text comment must follow the "doc" -> "paragraph" -> "text" nesting
	payload := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": commentText,
						},
					},
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(CurrentInstance.Email, CurrentInstance.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
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
