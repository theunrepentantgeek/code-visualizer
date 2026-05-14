package table

import (
	"fmt"
	"strings"
)

type ConsoleTable struct {
	content  [][]string
	widths   []int
	maxWidth int // 0 means unlimited
}

// New returns a new console table with the specified columns.
func New(columns ...string) *ConsoleTable {
	result := &ConsoleTable{}
	result.AddRow(columns...)

	return result
}

// SetMaxWidth sets a maximum total width for the table. When > 0, the last
// column is word-wrapped so the table fits within the given width.
func (t *ConsoleTable) SetMaxWidth(width int) {
	t.maxWidth = width
}

// AddRow adds an entire row to the table, tracking widths for final formatting.
func (t *ConsoleTable) AddRow(row ...string) {
	t.content = append(t.content, row)
	for i, r := range row {
		w := len(r)
		if i >= len(t.widths) {
			t.widths = append(t.widths, w)
		} else if w > t.widths[i] {
			t.widths[i] = w
		}
	}
}

// WriteTo renders the console table into the specified buffer.
func (t *ConsoleTable) WriteTo(buffer *strings.Builder) {
	// Calculate effective widths considering maxWidth constraint.
	widths := t.effectiveWidths()

	for i, r := range t.content {
		t.renderRow(r, widths, buffer)

		if i == 0 {
			t.renderRowDivider(widths, buffer)
		}
	}
}

// effectiveWidths returns the column widths to use for rendering.
// When maxWidth is set, the last column is shrunk to fit.
func (t *ConsoleTable) effectiveWidths() []int {
	widths := make([]int, len(t.widths))
	copy(widths, t.widths)

	if t.maxWidth <= 0 || len(widths) == 0 {
		return widths
	}

	// Total width = sum of (width + 3) for each column + 1 (leading '|')
	// Each column renders as " %*s |" which is width+3 chars after the leading '|'.
	fixedWidth := 1
	for _, w := range widths {
		fixedWidth += w + 3
	}

	if fixedWidth <= t.maxWidth {
		return widths
	}

	// Shrink the last column to make the table fit.
	lastIdx := len(widths) - 1
	overhead := fixedWidth - widths[lastIdx] - 3 // width without last column content
	available := t.maxWidth - overhead - 3        // available chars for last column content

	const minLastCol = 10
	if available < minLastCol {
		available = minLastCol
	}

	if available < widths[lastIdx] {
		widths[lastIdx] = available
	}

	return widths
}

// renderRow writes a single row into the buffer, word-wrapping the last column
// if its effective width is smaller than the stored width.
func (t *ConsoleTable) renderRow(row []string, widths []int, buffer *strings.Builder) {
	if len(row) == 0 {
		return
	}

	lastIdx := len(row) - 1
	lastEffective := widths[lastIdx]
	lastStored := t.widths[lastIdx]

	// If the last column doesn't need wrapping, render normally.
	if lastEffective >= lastStored {
		buffer.WriteRune('|')

		for i, c := range row {
			fmt.Fprintf(buffer, " %*s |", -widths[i], c)
		}

		buffer.WriteString("\n")

		return
	}

	// Word-wrap the last column.
	wrapped := wordWrap(row[lastIdx], lastEffective)

	// First line: all columns.
	buffer.WriteRune('|')

	for i, c := range row[:lastIdx] {
		fmt.Fprintf(buffer, " %*s |", -widths[i], c)
	}

	fmt.Fprintf(buffer, " %*s |\n", -lastEffective, wrapped[0])

	// Continuation lines: empty cells for leading columns, wrapped content for last.
	for _, cont := range wrapped[1:] {
		buffer.WriteRune('|')

		for i := range row[:lastIdx] {
			fmt.Fprintf(buffer, " %*s |", -widths[i], "")
		}

		fmt.Fprintf(buffer, " %*s |\n", -lastEffective, cont)
	}
}

// renderRowDivider writes a dividing line into the buffer.
func (t *ConsoleTable) renderRowDivider(widths []int, buffer *strings.Builder) {
	buffer.WriteString("|")

	for _, w := range widths {
		for i := -2; i < w; i++ {
			buffer.WriteRune('-')
		}

		buffer.WriteRune('|')
	}

	buffer.WriteString("\n")
}

// wordWrap splits text into lines of at most maxWidth runes, breaking on word
// boundaries where possible.
func wordWrap(text string, maxWidth int) []string {
	if maxWidth <= 0 || len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	current := ""

	for _, word := range words {
		if current == "" {
			current = word
			continue
		}

		candidate := current + " " + word
		if len(candidate) <= maxWidth {
			current = candidate
		} else {
			lines = append(lines, current)
			current = word
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}
