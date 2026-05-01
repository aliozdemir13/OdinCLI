package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aliozdemir13/odincli/internal"
	"github.com/aliozdemir13/odincli/internal/models"
)

// Helper to capture Stdout
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// Helper to mock Stdin
func mockStdin(input string) func() {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write([]byte(input))
		w.Close()
	}()

	return func() { os.Stdin = oldStdin }
}

func mockServerSetup() *httptest.Server {
	// Setup Mock Server
	jqlCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// 1. Mock Search JQL (Used by FetchIssues and FetchEpicChildren)
		case strings.Contains(r.URL.Path, "/search/jql"):
			jql := r.URL.Query().Get("jql")

			// Trigger Error Path
			if strings.Contains(jql, "trigger-error") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Trigger Empty Path
			if strings.Contains(jql, "trigger-empty") {
				json.NewEncoder(w).Encode(map[string]interface{}{"issues": []interface{}{}, "isLast": true})
				return
			}

			// PAGINATION LOGIC
			isLast := true
			nextToken := ""
			if jqlCallCount == 0 {
				isLast = false
				nextToken = "page-2"
				jqlCallCount++
			}

			resp := map[string]interface{}{
				"isLast":        isLast,
				"nextPageToken": nextToken,
				"issues": []map[string]interface{}{
					{
						"key": "PROJ-1",
						"fields": map[string]interface{}{
							"summary":   "Task Item",
							"priority":  map[string]interface{}{"name": "High"},
							"issuetype": map[string]interface{}{"name": "Task"},
							"status":    map[string]interface{}{"name": "To Do"},
							"assignee":  map[string]interface{}{"displayName": ""}, // Hits Unassigned
						},
					},
					{
						"key": "EPIC-1",
						"fields": map[string]interface{}{
							"summary":   "Epic Item make it longer than 40 characters to test the concatenation logic as well",
							"issuetype": map[string]interface{}{"name": "Epic"},
							"priority":  map[string]interface{}{"name": "Medium"},
							"status":    map[string]interface{}{"name": "In Progress"},
							"assignee":  map[string]interface{}{"displayName": "ali"},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)

		// 2. Mock Comments
		case strings.Contains(r.URL.Path, "/comment"):
			if r.Method == "POST" {
				if strings.Contains(r.URL.Path, "negative") {
					w.WriteHeader(http.StatusBadRequest)
				}
				w.WriteHeader(http.StatusCreated)
				return
			}
			resp := map[string]interface{}{
				"total": 1,
				"comments": []map[string]interface{}{
					{
						"id":      "101",
						"author":  map[string]interface{}{"displayName": "Ali"},
						"created": "2023-01-01",
						"updated": "2023-01-02", // Trigger [edited] logic
						"body": map[string]interface{}{
							"type":    "doc",
							"version": 1,
							"content": []interface{}{
								map[string]interface{}{
									"type": "paragraph",
									"content": []interface{}{
										map[string]interface{}{"type": "text", "text": "Comment Text"},
									},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)

		// 3. Mock Transitions
		case strings.Contains(r.URL.Path, "/transitions"):
			if r.Method == "POST" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			resp := map[string]interface{}{
				"transitions": []map[string]interface{}{
					{"id": "1", "name": "Done", "to": map[string]interface{}{"name": "Done"}},
				},
			}
			json.NewEncoder(w).Encode(resp)

		// 4. Mock User Search
		case strings.Contains(r.URL.Path, "/user/search"):
			resp := []map[string]interface{}{
				{"accountId": "acc-1", "displayName": "Ali Ozdemir"},
				{"accountId": "acc-2", "displayName": "Other User"},
			}
			json.NewEncoder(w).Encode(resp)

		// 5. Mock Assignee PUT
		case strings.Contains(r.URL.Path, "/assignee"):
			if strings.Contains(r.URL.Path, "negative") {
				w.WriteHeader(http.StatusBadRequest)
			}
			w.WriteHeader(http.StatusNoContent)

		// 6. Mock Create Issue
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/issue"):
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"key": "NEW-1"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errorMessages": ["Not Found"]}`))
		}
	}))
	return server
}

// TestHandlePull tests logic for switching projects based on your models.Config
func TestHandlePull(t *testing.T) {
	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: "https://test.atlassian.net", Email: "test@test.com"},
		},
	}

	t.Run("Valid Project", func(t *testing.T) {
		parts := []string{"pull", "TEST"}
		// Note: internal.FetchIssues will likely be called here.
		success := HandlePull(parts, config, "mock-api-key")

		if !success {
			t.Error("Expected handlePull to return true for valid project")
		}

		if internal.CurrentInstance.Name != "TEST" {
			t.Errorf("Expected instance name TEST, got %s", internal.CurrentInstance.Name)
		}
	})

	t.Run("Invalid Project", func(t *testing.T) {
		parts := []string{"pull", "UNKNOWN"}
		output := captureStdout(func() {
			success := HandlePull(parts, config, "key")
			if success {
				t.Error("Expected handlePull to return false for unknown project")
			}
		})

		if !strings.Contains(output, "not found in config.json") {
			t.Error("Expected error message in output")
		}
	})
}

// TestHandleFilter tests filtering logic using models.Issues
func TestHandleFilter(t *testing.T) {
	// Setup mock data in the internal package global state
	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: "https://test.atlassian.net", Email: "test@test.com"},
		},
	}

	parts := []string{"pull", "TEST"}
	// Note: internal.FetchIssues will likely be called here.
	_ = HandlePull(parts, config, "mock-api-key")

	// Note: We are using models.Issues (plural) as defined in your provided types
	internal.LastEntries = []models.Issues{
		{
			Key: "PROJ-1",
			Fields: models.Fields{
				Summary: "Test Issue",
				Assignee: struct {
					Name         string "json:\"displayName\""
					EmailAddress string "json:\"emailAddress\""
				}{
					Name:         "John Doe",
					EmailAddress: "test@test.com",
				},
				Priority: struct {
					Name string `json:"name"`
				}{Name: "High"},
				Status: struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Name      string `json:"name"`
						ColorName string `json:"colorName"`
					} `json:"statusCategory"`
				}{Name: "In Progress"},
			},
		},
		{
			Key: "PROJ-2",
			Fields: models.Fields{
				Summary: "Test Issue",
				IssueType: struct {
					Name string "json:\"name\""
				}{
					Name: "Epic",
				},
				Assignee: struct {
					Name         string "json:\"displayName\""
					EmailAddress string "json:\"emailAddress\""
				}{
					Name:         "John Doe",
					EmailAddress: "test@test.com",
				},
				Priority: struct {
					Name string `json:"name"`
				}{Name: "High"},
				Status: struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Name      string `json:"name"`
						ColorName string `json:"colorName"`
					} `json:"statusCategory"`
				}{Name: "In Progress"},
			},
		},
	}

	t.Run("Filter by Status Match", func(t *testing.T) {
		parts := []string{"filter", "status In Progress"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "PROJ-1") {
			t.Error("Expected PROJ-1 to be in filtered output")
		}
	})

	t.Run("Filter by Status No Match", func(t *testing.T) {
		parts := []string{"filter", "status Done"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "No matching issues found") {
			t.Error("Expected 'No matching issues' message")
		}
	})

	t.Run("Filter by Epic", func(t *testing.T) {
		parts := []string{"filter", "epics"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "PROJ-2") {
			t.Error("Expected PROJ-2 to be in filtered output")
		}
	})

	t.Run("Filter my issues", func(t *testing.T) {
		parts := []string{"filter", "myIssues"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "PROJ-1") {
			t.Error("Expected PROJ-1 to be in filtered output")
		}
	})

	internal.LastEntries = []models.Issues{
		{
			Key: "PROJ-1",
			Fields: models.Fields{
				Summary: "Test Issue",
				Priority: struct {
					Name string `json:"name"`
				}{Name: "High"},
				Status: struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Name      string `json:"name"`
						ColorName string `json:"colorName"`
					} `json:"statusCategory"`
				}{Name: "In Progress"},
			},
		},
	}

	t.Run("Filter by Epic negative", func(t *testing.T) {
		parts := []string{"filter", "epics"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "No Epics found in last pull.") {
			t.Error("Expected 'No Epics found in last pull.' in filtered output")
		}
	})

	t.Run("Filter my issues", func(t *testing.T) {
		parts := []string{"filter", "myIssues"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "Issue not found in last pull.") {
			t.Error("Expected 'Issue not found in last pull.' in filtered output")
		}
	})

	t.Run("Filter current sprint", func(t *testing.T) {
		parts := []string{"filter", "currentsprint"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "No issues found for this query.") {
			t.Error("Expected 'No issues found for this query.' in filtered output")
		}
	})

	t.Run("Filter backlog", func(t *testing.T) {
		parts := []string{"filter", "backlog"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "No issues found for this query.") {
			t.Error("Expected 'No issues found for this query.' in filtered output")
		}
	})

	t.Run("Filter error", func(t *testing.T) {
		parts := []string{"filter"}
		output := captureStdout(func() {
			HandleFilter(parts)
		})

		if !strings.Contains(output, "filter ---status In Progress") {
			t.Error("Expected 'filter ---status In Progress' in filtered output")
		}
	})
}

// TestHandleDetails tests showing issue information from cache
func TestHandleDetails(t *testing.T) {
	// Populate the Cache and LastEntries
	internal.EntriesCache = make(map[string]models.Issues)

	issue := models.Issues{
		Key: "PROJ-1",
		Fields: models.Fields{
			Summary: "Details Summary",
			Status: struct {
				Name           string `json:"name"`
				StatusCategory struct {
					Name      string `json:"name"`
					ColorName string `json:"colorName"`
				} `json:"statusCategory"`
			}{
				StatusCategory: struct {
					Name      string `json:"name"`
					ColorName string `json:"colorName"`
				}{Name: "In Progress"},
			},
		},
	}

	internal.EntriesCache["PROJ-1"] = issue
	internal.LastEntries = []models.Issues{issue}

	t.Run("Found in Cache", func(t *testing.T) {
		parts := []string{"details", "PROJ-1"}
		output := captureStdout(func() {
			// This will likely trigger internal.FetchComments(key)
			HandleDetails(parts)
		})

		if !strings.Contains(output, "Details Summary") {
			t.Error("Expected issue summary in output")
		}
	})

	t.Run("Found in Cache", func(t *testing.T) {
		parts := []string{"details"}
		output := captureStdout(func() {
			// This will likely trigger internal.FetchComments(key)
			HandleDetails(parts)
		})

		if !strings.Contains(output, "details ---{{Key}}") {
			t.Error("Expected 'details ---{{Key}}' in output")
		}
	})
}

// TestHandleStatus_Cancel tests the user input interruption logic
func TestHandleStatus_Cancel(t *testing.T) {
	// Mock Stdin to simulate user typing "c" then Enter
	input := "c\n"
	r, w, _ := os.Pipe()
	_, _ = w.Write([]byte(input))
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// We assume handleStatus will fail or return false because we aren't mocking the API response
	// for GetAvailableTransitions, but this verifies the 'c' input logic doesn't crash.
	parts := []string{"status", "PROJ-1"}
	success := HandleStatus(parts)

	if success {
		t.Error("Expected handleStatus to return false when cancelled with 'c'")
	}
}

// TestHandleSearch tests the text search logic
func TestHandleSearch(t *testing.T) {
	server := mockServerSetup()
	defer server.Close()

	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"PROJ-1": {URL: "https://test.atlassian.net", Email: "test@test.com"},
		},
	}

	parts := []string{"pull", "PROJ-1"}
	// Note: internal.FetchIssues will likely be called here.
	_ = HandlePull(parts, config, "mock-api-key")

	t.Run("Search keyword", func(t *testing.T) {
		parts := []string{"search", "PROJ-1"}
		// Note: internal.FetchIssues will likely be called here.
		success := HandleSearch(parts)

		if !success {
			t.Error("Expected handle search to return result")
		}

		if internal.CurrentInstance.Name != "PROJ-1" {
			t.Errorf("Expected instance name PROJ-1, got %s", internal.CurrentInstance.Name)
		}
	})

	t.Run("Invalid Search", func(t *testing.T) {
		parts := []string{"search", "UNKNOWN"}
		output := captureStdout(func() {
			success := HandleSearch(parts)
			if !success {
				t.Error("Expected handle search to return false for unknown project")
			}
		})

		if !strings.Contains(output, "No issues found for this query") {
			t.Error("Expected error message in output")
		}
	})

	t.Run("Invalid Search", func(t *testing.T) {
		parts := []string{"search"}
		output := captureStdout(func() {
			success := HandleSearch(parts)
			if success {
				t.Error("Expected handle search to return false for unknown project")
			}
		})

		if !strings.Contains(output, "search ---\"your keyword or phrase\"") {
			t.Error("Expected error message in output")
		}
	})
}

// TestHandleStatus tests the status update logic
func TestHandleStatus(t *testing.T) {
	server := mockServerSetup()
	defer server.Close()

	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: server.URL, Email: "test@test.com"},
		},
	}

	HandlePull([]string{"pull", "TEST"}, config, "mock-key")

	t.Run("Success update", func(t *testing.T) {
		// Mock user typing "1" then Enter
		restoreStdin := mockStdin("1\n")
		defer restoreStdin()

		parts := []string{"status", "PROJ-1"}
		output := captureStdout(func() {
			success := HandleStatus(parts)
			if !success {
				t.Error("HandleStatus failed")
			}
		})

		if !strings.Contains(output, "Status changed to Done") {
			t.Errorf("Expected success message, got: %s", output)
		}
	})

	t.Run("Invalid status update", func(t *testing.T) {
		parts := []string{"status"}
		output := captureStdout(func() {
			success := HandleStatus(parts)
			if success {
				t.Error("Expected handle search to return false for missing parameter")
			}
		})

		if !strings.Contains(output, "status ---{{KEY}}") {
			t.Error("Expected 'status ---{{KEY}}' in output")
		}
	})
}

func TestHandleAssign(t *testing.T) {
	server := mockServerSetup()
	defer server.Close()

	config := models.Config{Projects: map[string]models.ProjectConfig{"TEST": {URL: server.URL}}}
	HandlePull([]string{"pull", "TEST"}, config, "key")

	t.Run("Assign to user", func(t *testing.T) {
		// Mock user typing "1" to select the first user in the search results
		restore := mockStdin("1\n")
		defer restore()

		success := HandleAssign([]string{"assign", "PROJ-1", "ali"})
		if !success {
			t.Error("Expected HandleAssign to succeed")
		}
	})

	t.Run("Handle assign user negative", func(t *testing.T) {
		output := captureStdout(func() {
			success := HandleAssign([]string{"assign"})
			if success {
				t.Error("Expected handle add comment to return false for missing parameter")
			}
		})

		if !strings.Contains(output, "assign ---{{KEY}}") {
			t.Error("Expected 'assign ---{{KEY}}' in output")
		}
	})
}

func TestHandleCreateIssue(t *testing.T) {
	server := mockServerSetup()
	defer server.Close()

	// stores the original version of the function like a bookmark
	oldForm := RunIssueForm
	oldEditor := RunDescriptionEditor

	// reset changes at the end of the test because
	// in Go is that tests in the same package share the same memory.
	defer func() {
		RunIssueForm = oldForm
		RunDescriptionEditor = oldEditor
	}()

	// assign the version of function that test should execute
	RunIssueForm = func() (string, string, string, string, error) {
		// Simulate filling out the form
		return "Test Summary", "Task", "5", "", nil
	}

	// assign the version of function that test should execute
	RunDescriptionEditor = func() (string, bool) {
		// Simulate typing in the editor
		return "Test Description", false
	}

	// Setup internal state
	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: server.URL, Email: "test@test.com"},
		},
	}
	HandlePull([]string{"pull", "TEST"}, config, "mock-api-key")

	t.Run("Handle add create issue", func(t *testing.T) {
		// Act
		success := HandleCreateIssue()

		// Assert
		if !success {
			t.Error("Expected HandleCreateIssue to succeed")
		}
	})

	RunIssueForm = func() (string, string, string, string, error) {
		// Simulate filling out the form
		return "Test Summary", "Task", "5", "", fmt.Errorf("test error")
	}

	t.Run("Handle issue create negative for form abortion", func(t *testing.T) {
		output := captureStdout(func() {
			success := HandleCreateIssue()
			if success {
				t.Error("Expected HandleCreateIssue to return false")
			}
		})

		if !strings.Contains(output, "Cancelled.") {
			t.Error("Expected 'Cancelled.' in output")
		}
	})

	// reset this to the original test form for passing the validation
	RunIssueForm = func() (string, string, string, string, error) {
		// Simulate filling out the form
		return "Test Summary", "Task", "5", "", nil
	}

	RunDescriptionEditor = func() (string, bool) {
		// Simulate typing in the editor
		return "", true
	}

	t.Run("Handle issue create negative for editor closure", func(t *testing.T) {
		output := captureStdout(func() {
			success := HandleCreateIssue()
			if success {
				t.Error("Expected HandleCreateIssue to return false")
			}
		})

		if !strings.Contains(output, "Creation cancelled.") {
			t.Errorf("Expected 'Creation cancelled.' in output but it is %s", output)
		}
	})
}

func TestHandleAddComment(t *testing.T) {
	server := mockServerSetup()
	defer server.Close()

	// stores the original version of the function like a bookmark
	oldEditor := RunCommendEditor

	// reset changes at the end of the test because
	// in Go is that tests in the same package share the same memory.
	defer func() {
		RunCommendEditor = oldEditor
	}()

	// assign the version of function that test should execute
	RunCommendEditor = func() string {
		// Simulate typing in the editor
		return "Test comment"
	}

	// Setup internal state
	config := models.Config{
		Projects: map[string]models.ProjectConfig{
			"TEST": {URL: server.URL, Email: "test@test.com"},
		},
	}
	HandlePull([]string{"pull", "TEST"}, config, "mock-api-key")

	t.Run("Handle add comment", func(t *testing.T) {
		// Act
		success := HandleAddComment([]string{"addComment", "Test comment"})

		// Assert
		if !success {
			t.Error("Expected HandleCreateIssue to succeed")
		}
	})

	t.Run("Handle add negative", func(t *testing.T) {
		output := captureStdout(func() {
			success := HandleAddComment([]string{"addComment"})
			if success {
				t.Error("Expected handle add comment to return false for missing parameter")
			}
		})

		if !strings.Contains(output, "addComment {{Key}}") {
			t.Error("Expected 'addComment {{Key}}' in output")
		}
	})

	// assign the version of function that test should execute
	RunCommendEditor = func() string {
		// Simulate typing in the editor
		return ""
	}

	t.Run("Handle add negative for empty text", func(t *testing.T) {
		success := HandleAddComment([]string{"addComment", "Test comment"})
		if success {
			t.Error("Expected handle add comment to return false for missing parameter")
		}
	})
}
