package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aliozdemir13/odincli/internal/style"
)

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
			b.WriteString(style.StyleBlue(fmt.Sprintf("%v", val)))
		}
	}
	for _, child := range node.Content {
		walkNodes(child, b)
		if child.Type == "paragraph" || child.Type == "listItem" {
			b.WriteString("\n")
		}
	}
}

func MarkdownToADF(input string) JiraDescription {
	lines := strings.Split(input, "\n")
	var rootContent []DescriptionNode

	var currentList *DescriptionNode
	var currentListType string // "bulletList" or "taskList"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle Empty Lines (Close any open lists)
		if trimmed == "" {
			if currentList != nil {
				rootContent = append(rootContent, *currentList)
				currentList = nil
				currentListType = ""
			}
			continue
		}

		// Detect Line Type
		if strings.HasPrefix(trimmed, "#") {
			// --- HEADERS ---
			if currentList != nil {
				rootContent = append(rootContent, *currentList)
				currentList = nil
				currentListType = ""
			}
			rootContent = append(rootContent, parseHeading(trimmed))

		} else if isTaskItem(trimmed) {
			// --- TASK LISTS ---
			if currentListType != "taskList" {
				if currentList != nil {
					rootContent = append(rootContent, *currentList)
				}
				currentList = &DescriptionNode{Type: "taskList"}
				currentListType = "taskList"
			}
			currentList.Content = append(currentList.Content, parseTaskItem(trimmed))

		} else if isBulletItem(trimmed) {
			// --- BULLET LISTS ---
			if currentListType != "bulletList" {
				if currentList != nil {
					rootContent = append(rootContent, *currentList)
				}
				currentList = &DescriptionNode{Type: "bulletList", Content: []DescriptionNode{}}
				currentListType = "bulletList"
			}
			currentList.Content = append(currentList.Content, parseBulletItem(trimmed))

		} else {
			// --- STANDARD PARAGRAPH ---
			if currentList != nil {
				rootContent = append(rootContent, *currentList)
				currentList = nil
				currentListType = ""
			}

			rootContent = append(rootContent, DescriptionNode{
				Type:    "paragraph",
				Content: parseInlineMention(line),
			})
		}
	}

	// Catch the last list if the file doesn't end with an empty line
	if currentList != nil {
		rootContent = append(rootContent, *currentList)
	}

	return JiraDescription{
		Type:    "doc",
		Version: 1,
		Content: rootContent,
	}
}

func parseHeading(line string) DescriptionNode {
	level := 0
	for line[level] == '#' && level < 6 {
		level++
	}
	text := strings.TrimSpace(line[level:])
	return DescriptionNode{
		Type:    "heading",
		Attrs:   map[string]any{"level": level},
		Content: []DescriptionNode{{Type: "text", Text: text}},
	}
}

func isBulletItem(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func parseBulletItem(line string) DescriptionNode {
	text := strings.TrimSpace(line[2:])
	return DescriptionNode{
		Type: "listItem",
		Content: []DescriptionNode{
			{
				Type:    "paragraph",
				Content: []DescriptionNode{{Type: "text", Text: text}},
			},
		},
	}
}

func isTaskItem(line string) bool {
	l := strings.ToLower(line)
	return strings.HasPrefix(l, "- [ ] ") || strings.HasPrefix(l, "- [X] ")
}

func parseTaskItem(line string) DescriptionNode {
	state := "TODO"
	if strings.HasPrefix(strings.ToLower(line), "- [X] ") {
		state = "DONE"
	}

	return DescriptionNode{
		Type: "taskItem",
		Attrs: map[string]any{
			"state":   state,
			"localId": fmt.Sprintf("task-%d", time.Now().UnixNano()),
		},
		Content: []DescriptionNode{
			{
				Type:    "paragraph",
				Content: parseInlineMention(strings.TrimSpace(line[6:])),
			},
		},
	}
}

func parseInlineMention(input string) []DescriptionNode {
	var nodes []DescriptionNode

	// Regular expression to find [[accountId|DisplayName]]
	mentionRegex := regexp.MustCompile(`\[\[(.*?)\|(.*?)\]\]`)

	lastIndex := 0
	matches := mentionRegex.FindAllStringSubmatchIndex(input, -1)

	for _, m := range matches {
		// Append text before the mention
		if m[0] > lastIndex {
			nodes = append(nodes, DescriptionNode{
				Type: "text",
				Text: input[lastIndex:m[0]],
			})
		}

		// Append the mention node
		accountId := input[m[2]:m[3]]
		displayName := input[m[4]:m[5]]

		nodes = append(nodes, DescriptionNode{
			Type: "mention",
			Attrs: map[string]any{
				"id":          accountId,
				"text":        "@" + displayName,
				"accessLevel": "", // Jira often expects this field, even if empty
			},
		})

		lastIndex = m[1]
	}

	// Append remaining text after last mention
	if lastIndex < len(input) {
		nodes = append(nodes, DescriptionNode{
			Type: "text",
			Text: input[lastIndex:],
		})
	}

	// Fallback: if no mentions found, return as plain text
	if len(nodes) == 0 {
		return []DescriptionNode{{Type: "text", Text: input}}
	}

	return nodes
}
