package models

import (
	"fmt"
	"log_tracker/internal/style"
	"strings"
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
