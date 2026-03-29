# OdinCLI
Productivity tool connecting enterprise tools through CLI. 
Focus on your code, not your browser tabs!

## Currently Supported Toolstack
### Jira Cloud
Odin listens to your command and brings you the knowledge from Jira Cloud. 

Security-focused design - none of the pulled information stored in the device but only cached during the session, and once the session has ended - they will be cleared up. For app to work, **Authentication Token** must be created from Jira and a local .env file must be configured with it.

Pull command defines the context(project), and for context change a new pull with the desired project key must be done. For refreshing the context, pull with same project key must be done.

Command	Action:
- pull ---{{KEY}} -> Fetches open tickets for a specific project (e.g., pull ABC).
- details ---{{KEY}} -> Displays description, status, priority, and recent comments.
- details ---epic {{KEY}} -> Displays issues under the specified epic.
- addComment ---{{KEY}}	comment to add -> Prompts to add a new comment to a specific issue.
- filter ---status {{STATUS}} -> Filters the current local list by status (e.g., filter ---status In Progress).
- filter ---prio {{PRIORITY}} -> Filters the current local list by priority (e.g., filter ---prio High).
- filter ---currentSprint (case insensitive) -> Fetches the current sprint tickets and change context as current sprint. Any filter to apply after this, will be done on current sprint tickets.
- filter ---backlog (case insensitive) -> Fetches the backlog tickets and change context as current sprint. Any filter to apply after this, will be done on backlog tickets.
- filter ---myIssues (case insensitive) -> Quick-filter to show only issues assigned to you. This filter always work on the current context (sprint, backlog or all issues)
- filter ---epics (case insensitive) -> Quick-filter to show only the epics. This filter always work on the current context (sprint, backlog or all issues)
- status ---{{KEY}}	Allo updating status of the issue. Available issues getting displayed as menu options to select for this command.
- search ---"your keyword or phrase" -> Performs a text search based on given phrase or keyword.
- assign ---{{KEY}} -> change assignment of the issue.
- help -> Lists all commands.
- exit -> Safely closes the application and clears the session.

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
- [ ] Adding logTime command
- [ ] Adding create command
- [ ] Adding changeEstimation command
- [X] Adding search epics and display all the child issues under it
- [X] Extend filter to see current sprint and backlog
- [ ] Adding ability of mentioning people in comments
- [X] Update search command for text search

## Contributing
This is an open-source project! If you have ideas for new enterprise integrations (Slack, GitHub, ADO), feel free to open an issue or a PR.

## License
MIT License - see LICENSE for details.
