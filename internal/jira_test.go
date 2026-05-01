package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aliozdemir13/odincli/internal/models"
)

// Helper to seed anonymous structs via JSON workaround
func seedIssue(jsonStr string) models.Issues {
	var issue models.Issues
	json.Unmarshal([]byte(jsonStr), &issue)
	return issue
}

func TestAllInternalFunctions(t *testing.T) {
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
	defer server.Close()

	// Initialize Instance
	CurrentInstance = models.JiraInstance{
		BaseURL: server.URL,
		Email:   "test@test.com",
		Token:   "token",
	}

	// Workaround for seeded cache
	EntriesCache = make(map[string]models.Issues)
	EntriesCache["PROJ-2"] = seedIssue(`{"fields": {"summary": "The Epic Name"}}`)

	// --- 1. Test Request Helpers & Error Branches ---
	t.Run("PerformRequest_Failures", func(t *testing.T) {
		// Test Status Code Error (404)
		req, _ := http.NewRequest("GET", server.URL+"/invalid-path", nil)
		err := performRequest(req, http.StatusOK, nil)
		if err == nil {
			t.Error("Expected error for 404 status code")
		}

		// Test Network Failure
		badReq, _ := http.NewRequest("GET", "http://localhost:12345", nil)
		err = performRequest(badReq, http.StatusOK, nil)
		if err == nil {
			t.Error("Expected error for network failure")
		}
	})

	// --- 2. Test Fetching Functions ---
	t.Run("FetchIssues_Logic", func(t *testing.T) {
		jqlCallCount = 0            // Reset for test
		FetchIssues("project=PROJ") // Hits Page 1, then Page 2 (isLast loop)

		FetchIssues("trigger-empty") // Hits "No issues found" branch
		FetchIssues("trigger-error") // Hits performRequest error branch

		// Hits "No instance" branch
		tmp := CurrentInstance.BaseURL
		CurrentInstance.BaseURL = ""
		FetchIssues("test")
		CurrentInstance.BaseURL = tmp
	})

	t.Run("FetchComments_Logic", func(t *testing.T) {
		FetchComments("PROJ-1")
	})

	t.Run("FetchEpicChildren_Logic", func(t *testing.T) {
		FetchEpicChildren("EPIC-1")

		// Trigger "No child issues found"
		old := CurrentInstance.BaseURL
		CurrentInstance.BaseURL = server.URL + "/search/jql?jql=trigger-empty"
		// Note: we just need the request to fail or return 0
		FetchEpicChildren("EPIC-1")
		CurrentInstance.BaseURL = server.URL + "/search/jql?jql=trigger-error"
		// Error case
		FetchEpicChildren("EPIC-12")
		CurrentInstance.BaseURL = old
	})

	// --- 3. Test API Actions ---
	t.Run("APIActions", func(t *testing.T) {
		AddCommentToJira("PROJ-1", "new comment")
		GetAvailableTransitions("PROJ-1")
		PerformTransition("PROJ-1", "1")
		SearchUsers("ali")
		AssignIssue("PROJ-1", "acc-1")
		CreateIssueInJira(models.CreateIssueRequest{}, "13") // With effort logic
	})

	// --- 4. Test Interactive Input (AssignInteractive) ---
	t.Run("AssignInteractive_Paths", func(t *testing.T) {
		// Scenario A: Confirm first match (Search "ali", then "y")
		input := "ali\ny\n"
		r, w, _ := os.Pipe()
		oldStdin := os.Stdin
		os.Stdin = r
		go func() {
			fmt.Fprint(w, input)
			w.Close()
		}()
		AssignInteractive("PROJ-1")
		os.Stdin = oldStdin

		// Scenario B: Select from list (Search "ali", then "n", then choice "2")
		inputMulti := "ali\nn\n2\n"
		r2, w2, _ := os.Pipe()
		os.Stdin = r2
		go func() {
			fmt.Fprint(w2, inputMulti)
			w2.Close()
		}()
		AssignInteractive("PROJ-1")
		os.Stdin = oldStdin

		// Scenario C: Empty input coverage
		r3, w3, _ := os.Pipe()
		os.Stdin = r3
		go func() {
			fmt.Fprint(w3, "\n")
			w3.Close()
		}()
		AssignInteractive("PROJ-1")
		os.Stdin = oldStdin

		// Scenario A: Confirm first match (Search "ali", then "y")
		inputError := "james\ny\n"
		r4, w4, _ := os.Pipe()
		os.Stdin = r4
		go func() {
			fmt.Fprint(w4, inputError)
			w4.Close()
		}()
		AssignInteractive("PROJ-1")
		os.Stdin = oldStdin
	})

	// --- 5. Test Assignment Error ---
	t.Run("Negative_APIActions", func(t *testing.T) {
		AssignIssue("negative", "acc-1")
		AddCommentToJira("negative", "new comment")
	})
}
