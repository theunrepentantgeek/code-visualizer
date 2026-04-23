package table

import (
	"fmt"
	"strings"
)

type ConsoleTable struct {
	content [][]string
	widths  []int
}

// New returns a new console table with the specified columns.
func New(columns ...string) *ConsoleTable {
	result := &ConsoleTable{}
	result.AddRow(columns...)

	return result
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
	for i, r := range t.content {
		t.renderRow(r, buffer)

		if i == 0 {
			t.renderRowDivider(buffer)
		}
	}
}

// renderRow writes a single row into the buffer.
func (t *ConsoleTable) renderRow(row []string, buffer *strings.Builder) {
	buffer.WriteRune('|')

	for i, c := range row {
		fmt.Fprintf(buffer, " %*s |", -t.widths[i], c)
	}

	buffer.WriteString("\n")
}

// renderRowDivider writes a dividing line into the buffer.
func (t *ConsoleTable) renderRowDivider(buffer *strings.Builder) {
	buffer.WriteString("|")

	for _, w := range t.widths {
		for i := -2; i < w; i++ {
			buffer.WriteRune('-')
		}

		buffer.WriteRune('|')
	}

	buffer.WriteString("\n")
}
