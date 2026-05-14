package main

import (
	"fmt"
	"strings"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/table"
)

// HelpMetricsCmd prints a table of all registered metrics.
type HelpMetricsCmd struct{}

//nolint:unparam // nil error required to satisfy the interface for Kong
func (HelpMetricsCmd) Run(_ *Flags) error {
	descriptors := provider.AllDescriptors()

	tbl := table.New("Metric", "Kind", "Default Palette", "Description")

	hasGit := false

	for _, d := range descriptors {
		k := kindLabel(d.Kind)
		desc := d.Description

		if git.IsGitMetric(d.Name) {
			hasGit = true
			desc += " †"
		}

		tbl.AddRow(string(d.Name), k, string(d.DefaultPalette), desc)
	}

	content := &strings.Builder{}

	tbl.WriteTo(content)

	fmt.Print(content.String())

	if hasGit {
		fmt.Printf("\n%s\n", "† requires a git repository")
	}

	return nil
}

func kindLabel(k metric.Kind) string {
	switch k {
	case metric.Quantity:
		return "quantity"
	case metric.Measure:
		return "measure"
	case metric.Classification:
		return "category"
	default:
		return "unknown"
	}
}
