# OdinCLI
Productivity tool connecting enterprise tools through CLI. 
Focus on your code, not your browser tabs!

## Currently Supported Toolstack
### Jira Cloud
Odin listens to your command and brings you the knowledge from Jira Cloud. 

Security-focused design - none of the pulled information is stored in the device but only cached during the session, and once the session has ended - they will be cleared up. For app to work, **Authentication Token** must be created from Jira and a local .env file must be configured with it.

Pull command defines the context(project), and for context change a new pull with the desired project key must be done. For refreshing the context, pull with same project key must be done.

### Command Syntax
Odin uses a unique `---` delimiter to separate the command from its arguments.

| Command | Action |
| :--- | :--- |
| `pull ---{{KEY}}` | Fetches open tickets for a project (e.g., `pull ---ABC`). |
| `details ---{{KEY}}` | Displays description, status, priority, and recent comments. |
| `details ---epic {{KEY}}` | Displays all child issues belonging to a specific epic. |
| `search ---"{{phrase}}"` | Performs a text search across summaries, descriptions, and comments. |
| `addComment ---{{KEY}} {{text}}` | Adds a new comment to a specific issue. |
| `status ---{{KEY}}` | Opens an interactive menu to transition the issue status. |
| `assign ---{{KEY}}` | Opens an interactive user search to change the assignee. |
| `filter ---status {{VAL}}` | Filters local list by status (e.g., `filter ---status In Progress`). |
| `filter ---prio {{VAL}}` | Filters local list by priority (e.g., `filter ---prio High`). |
| `filter ---currentSprint` | **Updates Context:** Fetches active sprint tickets. |
| `filter ---backlog` | **Updates Context:** Fetches the project backlog. |
| `filter ---myIssues` | Quick-filter to show only issues assigned to you. |
| `filter ---epics` | Quick-filter to show only Epics. |
| `help` | Lists all available commands. |
| `exit` | Safely closes Odin and clears the session. |

---

## Setup & Installation

1. **Prerequisites**: [Go](https://go.dev/dl/) installed on your machine.
2. **Clone the repo**: `git clone https://github.com/yourusername/odincli.git`
3. **Configure Environment**: Create a `.env` file in the root directory:
   ```env
   JIRA_TOKEN=your_generated_api_token
   ```
   Odin supports multiple instances connected to the same account.
4. **Config JSON**: Create a `config.json` file in the root directory:
    ```config
    {
        "projects": {
            "projectName1": { "url": "https://projectOne.atlassian.net", "email": "xxx@yyy.com" },
            "projectName2": { "url": "https://projectTwo.atlassian.net", "email": "xxx@yyy.com" }
        }
    }
    ```

## Build & Run
```
go build -o odin
./odin
```

### TODO
- [X] Extending filter to allow also use priority
- [X] Pull complete list of open issues with pagination
- [X] Adding change status command
- [X] Adding assign command both for current user and other potential users
- [X] Adding search epics and display all the child issues under it
- [X] Extend filter to see current sprint and backlog
- [X] Update search command for text search
- [ ] Adding logTime command
- [ ] Adding create command
- [ ] Adding changeEstimation command
- [ ] Adding ability of mentioning people in comments

## Contributing
This is an open-source project! If you have ideas for new enterprise integrations (Slack, GitHub, ADO), feel free to open an issue or a PR.

## License
MIT License - see LICENSE for details.
