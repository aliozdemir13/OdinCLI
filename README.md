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
- addComment ---{{KEY}}	comment to add -> Prompts to add a new comment to a specific issue.
- filter ---status {{STATUS}}	Filters the current local list by status (e.g., filter ---status In Progress).
- filter ---prio {{PRIORITY}}	Filters the current local list by priority (e.g., filter ---prio High).
- status ---{{KEY}}	Filters the current local list by priority (e.g., filter ---prio High).
- search ---{{KEY}} -> Performs a deep search for a specific ticket key.
- myIssues -> Quick-filter to show only issues assigned to you.
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
- [ ] Adding search epics and display all the child issues under it
- [ ] Extend filter to see current sprint and backlog
- [ ] Adding ability of mentioning people in comments
- [ ] Update search command for text search

## Contributing
This is an open-source project! If you have ideas for new enterprise integrations (Slack, GitHub, ADO), feel free to open an issue or a PR.

## License
MIT License - see LICENSE for details.
