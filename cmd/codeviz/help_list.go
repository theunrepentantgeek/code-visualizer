package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// nameDescription is a single entry in a help listing: a short identifier
// paired with a longer human-readable description.
type nameDescription struct {
	Name        string
	Description string
}

// renderNameDescriptionList renders entries in the same aligned, border-free
// style used by `help metrics`: an underlined section header followed by a
// name column aligned to the widest name, with descriptions wrapped to the
// console width.
func renderNameDescriptionList(title string, entries []nameDescription, width int) string {
	content := &strings.Builder{}

	if title != "" {
		writeSectionHeader(content, title)
	}

	nameWidth := 0
	for _, entry := range entries {
		nameWidth = max(nameWidth, utf8.RuneCountInString(entry.Name))
	}

	for _, entry := range entries {
		writeNameDescriptionBlock(content, nameWidth, entry, width)
	}

	return content.String()
}

func writeNameDescriptionBlock(content *strings.Builder, nameWidth int, entry nameDescription, width int) {
	prefix := fmt.Sprintf("  %-*s  ", nameWidth, entry.Name)

	if utf8.RuneCountInString(prefix) >= width {
		writeWrappedText(content, "  ", entry.Name, width)
		writeWrappedText(content, "    ", entry.Description, width)
		content.WriteString("\n")

		return
	}

	writeWrappedText(content, prefix, entry.Description, width)
	content.WriteString("\n")
}
