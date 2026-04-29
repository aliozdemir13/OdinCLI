// Package models provides data structures for the application
package models

// JiraInstance is the structure of authenticating in jira cloud
type JiraInstance struct {
	Name    string
	BaseURL string
	Email   string
	Token   string
}

// JiraResponse is the structure of the data Jira returns
type JiraResponse struct {
	StartAt       int      `json:"startAt"`
	MaxResults    int      `json:"maxResults"`
	Total         int      `json:"total"`
	Issues        []Issues `json:"issues"`
	NextPageToken string   `json:"nextPageToken"`
	IsLast        bool     `json:"isLast"`
}

// Issues is the structure supporting JiraResponse to parse payload
type Issues struct {
	Key    string `json:"key"`
	ID     string `json:"id"`
	Fields Fields `json:"fields"`
}

// Fields is the structure supporting Issues to parse payload
type Fields struct {
	Summary           string          `json:"summary"`
	Description       JiraDescription `json:"description"`
	ParsedDescription string
	IssueType         struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Priority struct {
		Name string `json:"name"`
	} `json:"priority"`
	Status struct {
		Name           string `json:"name"`
		StatusCategory struct {
			Name      string `json:"name"`
			ColorName string `json:"colorName"`
		} `json:"statusCategory"`
	} `json:"status"`
	Assignee struct {
		Name         string `json:"displayName"`
		EmailAddress string `json:"emailAddress"`
	} `json:"assignee"`
}

// DescriptionNode is the structure supporting JiraDescription to parse payload
type DescriptionNode struct {
	Type    string            `json:"type"`
	Text    string            `json:"text,omitempty"`    // Only present if type is "text"
	Content []DescriptionNode `json:"content,omitempty"` // Recursive for paragraphs/lists
	Attrs   map[string]any    `json:"attrs,omitempty"`   // For localId and other metadata
	Marks   []ADFMark         `json:"marks,omitempty"`
}

// JiraDescription is the structure for parsing the jira payload to display description of the issues
type JiraDescription struct {
	Type    string            `json:"type"`
	Version int               `json:"version"`
	Content []DescriptionNode `json:"content"`
}

// JiraCommentResponse data structure for parsing issue comments response from jira
type JiraCommentResponse struct {
	Comments []struct {
		ID     string `json:"id"`
		Author struct {
			DisplayName string `json:"displayName"`
		} `json:"author"`
		Created string          `json:"created"`
		Updated string          `json:"updated"`
		Body    JiraDescription `json:"body"`
	} `json:"comments"`
	Total int `json:"total"`
}

// JiraTransitionsResponse data structure for parsing transition response from jira
type JiraTransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// Transition data structure for parsing transitions which represents the status. this data structure is used for updating the value in jira
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"` // e.g., "In Progress"
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

// AddCommentRequest ata structure for adding comment to a jira issue via API
type AddCommentRequest struct {
	Body JiraDescription `json:"body"`
}

// AssigneePayload data structure for creating assignment request body
type AssigneePayload struct {
	AccountID string `json:"accountId"`
}

// JiraUser data structure for parsing user information from jira payload
type JiraUser struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

// ProjectConfig data structure for parsing config json inner objects to run the app
type ProjectConfig struct {
	URL   string `json:"url"`
	Email string `json:"email"`
}

// Config data structure for parsing config json file to run the app
type Config struct {
	Projects map[string]ProjectConfig `json:"projects"`
}

// Sprint data structure for parsing sprint from jira payload
type Sprint struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"` // "active", "closed", "future"
}

// CreateIssueRequest data structure for parsing issue request for creating issue post request
type CreateIssueRequest struct {
	Fields CreateFields `json:"fields"`
}

// CreateFields data structure for parsing issue fields for creating issue in jira
type CreateFields struct {
	Project     ProjectReference  `json:"project"`
	Summary     string            `json:"summary"`
	Description JiraDescription   `json:"description"`
	IssueType   IssueTypeName     `json:"issuetype"`
	Parent      *ProjectReference `json:"parent,omitempty"` // Pointer allows null
	Labels      []string          `json:"labels,omitempty"`
}

// ProjectReference data structure for parsing project reference from jira payload
type ProjectReference struct {
	Key string `json:"key"`
}

// IssueTypeName data structure for parsing issue type from jira payload
type IssueTypeName struct {
	Name string `json:"name"`
}

// ADFMark data structure for parsing ADF marks from jira payload
type ADFMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}
