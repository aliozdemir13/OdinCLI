package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aliozdemir13/odincli/internal/style"
)

// Properly parse the ADF format to markdown for better readability
func ParseADF(desc JiraDescription) string {
	var builder strings.Builder
	for _, node := range desc.Content {
		builder.WriteString(walkNodes(node, 0))
	}
	return builder.String()
}

// Recursively identify the node types and depths for rendering
func walkNodes(node DescriptionNode, depth int) string {
	var b strings.Builder

	switch node.Type {
	case "text":
		return applyMarks(node.Text, node.Marks) // Terminal does not parse the markdown currently completely, still it gives the visible distinction

	case "paragraph":
		for _, child := range node.Content {
			b.WriteString(walkNodes(child, depth))
		}
		b.WriteString("\n")

	case "heading":
		level, _ := node.Attrs["level"].(float64)
		b.WriteString(strings.Repeat("#", int(level)) + " ")
		for _, child := range node.Content {
			b.WriteString(walkNodes(child, depth))
		}
		b.WriteString("\n\n")

	case "taskList":
		for _, child := range node.Content {
			b.WriteString(walkNodes(child, depth))
		}

	case "taskItem":
		state, _ := node.Attrs["state"].(string)
		for _, child := range node.Content {
			if state == "DONE" {
				b.WriteString(style.StyleGreen("[x]") + " " + style.StyleGreen(strings.TrimSpace(walkNodes(child, depth))))
			} else {
				b.WriteString("[ ]" + " " + strings.TrimSpace(walkNodes(child, depth)))
			}
		}
		b.WriteString("\n")

	case "bulletList":
		for _, child := range node.Content {
			b.WriteString(walkNodes(child, depth))
		}
		b.WriteString("\n")

	case "listItem":
		indent := strings.Repeat("  ", depth)
		b.WriteString(indent + "• ")
		for _, child := range node.Content {
			// List items often contain paragraphs; we want to keep them on one line usually
			b.WriteString(strings.TrimSpace(walkNodes(child, depth+1)))
		}
		b.WriteString("\n")

	case "codeBlock":
		b.WriteString("```\n")
		for _, child := range node.Content {
			b.WriteString(child.Text)
		}
		b.WriteString("\n```\n\n")

	case "panel":
		panelType := node.Attrs["panelType"].(string)
		content := ""
		for _, child := range node.Content {
			content += walkNodes(child, depth)
		}
		b.WriteString(stylePanel(panelType, content))

	case "table":
		b.WriteString(parseTable(node))

	case "rule":
		b.WriteString("---\n")

	case "mention":
		if text, ok := node.Attrs["text"].(string); ok {
			b.WriteString(style.StyleBlue(text))
		}

	case "inlineCard", "blockCard":
		// Handles the "Smart Links" in Jira
		if url, ok := node.Attrs["url"].(string); ok {
			// Render it blue and underlined (if your style package supports it)
			b.WriteString(style.StyleCyan(url))
		}

	default:
		// Fallback for unknown nodes: just process children
		for _, child := range node.Content {
			b.WriteString(walkNodes(child, depth))
		}
	}

	return b.String()
}

func applyMarks(text string, marks []ADFMark) string {
	result := text
	for _, m := range marks {
		switch m.Type {
		case "strong":
			result = fmt.Sprintf("**%s**", result) // Markdown Bold
		case "em":
			result = fmt.Sprintf("*%s*", result) // Markdown Italic
		case "code":
			result = fmt.Sprintf("`%s`", result) // Inline code
		case "textColor":
			if color, ok := m.Attrs["color"].(string); ok {
				result = style.CustomColor(result, color) // Implement color mapping
			}
		case "link":
			if href, ok := m.Attrs["href"].(string); ok {
				result = fmt.Sprintf("[%s](%s)", result, href)
			}
		}
	}
	return result
}

func stylePanel(pType string, content string) string {
	// Trim trailing newlines from content to avoid huge gaps
	content = strings.TrimSpace(content)

	switch pType {
	case "info":
		return style.StyleBlue("ℹ️ INFO:\n"+content) + "\n\n"
	case "warning":
		return style.StyleYellow("⚠️ WARNING:\n"+content) + "\n\n"
	case "error":
		return style.StyleRed("🚫 ERROR:\n"+content) + "\n\n"
	case "success":
		return style.StyleGreen("✅ SUCCESS:\n"+content) + "\n\n"
	case "note":
		return style.StyleGray("📝 NOTE:\n"+content) + "\n\n"
	default:
		return content + "\n\n"
	}
}

func parseTable(node DescriptionNode) string {
	if len(node.Content) == 0 {
		return ""
	}

	// Parse all cells into a 2D slice of strings first
	var tableData [][]string
	for _, row := range node.Content {
		var rowData []string
		for _, cell := range row.Content {
			var cellParts []string
			for _, cellContent := range cell.Content {
				// Important: walkNodes here will apply bold/color marks
				txt := strings.TrimSpace(walkNodes(cellContent, 0))
				if txt != "" {
					cellParts = append(cellParts, txt)
				}
			}
			rowData = append(rowData, strings.Join(cellParts, " "))
		}
		tableData = append(tableData, rowData)
	}

	// Calculate the maximum width for each column
	// Note: use a helper 'visualLength' because ANSI color codes shouldn't count towards width
	colWidths := make([]int, len(tableData[0]))
	for _, row := range tableData {
		for i, cell := range row {
			if i < len(colWidths) {
				l := visualLength(cell)
				if l > colWidths[i] {
					colWidths[i] = l
				}
			}
		}
	}

	// Build the formatted string
	var b strings.Builder
	b.WriteString("\n")
	for i, row := range tableData {
		// Print Row
		b.WriteString("|")
		for colIdx, cell := range row {
			padding := colWidths[colIdx] - visualLength(cell)
			b.WriteString(" " + cell + strings.Repeat(" ", padding) + " |")
		}
		b.WriteString("\n")

		// Print Separator after header
		if i == 0 {
			b.WriteString("|")
			for _, width := range colWidths {
				b.WriteString(strings.Repeat("-", width+2) + "|")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// visualLength calculates length of string ignoring ANSI escape sequences
func visualLength(s string) int {
	// Simple regex to match ANSI escape codes
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	plain := re.ReplaceAllString(s, "")
	return len(plain)
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
				currentList = &DescriptionNode{Type: "taskList",
					Attrs: map[string]any{
						"localId": fmt.Sprintf("list-%d", time.Now().UnixNano()),
					}}
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
				Content: parseInlineMention(text),
			},
		},
	}
}

func isTaskItem(line string) bool {
	l := strings.ToLower(line)
	return strings.HasPrefix(l, "- [ ] ") || strings.HasPrefix(l, "- [x] ")
}

func parseTaskItem(line string) DescriptionNode {
	state := "TODO"
	if strings.HasPrefix(strings.ToLower(line), "- [x] ") {
		state = "DONE"
	}

	return DescriptionNode{
		Type: "taskItem",
		Attrs: map[string]any{
			"state": state,
			// Jira localIds must be unique within the document
			"localId": fmt.Sprintf("task-%d", time.Now().UnixNano()),
		},
		Content: parseInlineMention(strings.TrimSpace(line[6:])),
	}
}

func parseInlineMention(input string) []DescriptionNode {
	var nodes []DescriptionNode

	// Regular expression to find [[accountId|DisplayName]]
	mentionRegex := regexp.MustCompile(`\[\[(.*?)\|(.*?)\]\]`)

	lastIndex := 0
	// To cover the mention not only on the beginning of the text but in pretty much any position
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
