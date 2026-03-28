package internal

type Comment struct {
	CreatedDate string
	UpdatedDate string
	Id          string
	CreatedBy   string
	Text        string
}

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

type Issues struct {
	Key    string `json:"key"`
	Id     string `json:"id"`
	Fields struct {
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
	} `json:"fields"`
}

type DescriptionNode struct {
	Type    string            `json:"type"`
	Text    string            `json:"text,omitempty"`    // Only present if type is "text"
	Content []DescriptionNode `json:"content,omitempty"` // Recursive for paragraphs/lists
	Attrs   map[string]any    `json:"attrs,omitempty"`   // For localId and other metadata
}

type JiraDescription struct {
	Type    string            `json:"type"`
	Version int               `json:"version"`
	Content []DescriptionNode `json:"content"`
}

type JiraCommentResponse struct {
	Comments []struct {
		Id     string `json:"id"`
		Author struct {
			DisplayName string `json:"displayName"`
		} `json:"author"`
		Created string          `json:"created"`
		Updated string          `json:"updated"`
		Body    JiraDescription `json:"body"`
	} `json:"comments"`
	Total int `json:"total"`
}

type JiraTransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

type Transition struct {
	Id   string `json:"id"`
	Name string `json:"name"` // e.g., "In Progress"
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

type AddCommentRequest struct {
	Body JiraDescription `json:"body"`
}

type AssigneePayload struct {
	AccountId string `json:"accountId"`
}

type JiraUser struct {
	AccountId   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

type ProjectConfig struct {
	URL   string `json:"url"`
	Email string `json:"email"`
}

type Config struct {
	Projects map[string]ProjectConfig `json:"projects"`
}
