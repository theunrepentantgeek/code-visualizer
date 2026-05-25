package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
	"github.com/theunrepentantgeek/code-visualizer/internal/table"
)

// HelpMetricsCmd prints a table of all registered metrics.
type HelpMetricsCmd struct{}

//nolint:unparam // nil error required to satisfy the interface for Kong
func (HelpMetricsCmd) Run(_ *Flags) error {
	descriptors := provider.AllDescriptors()
	groups := map[string][]provider.MetricDescriptor{
		"Filesystem metrics": nil,
		"Git metrics":        nil,
		"Go metrics":         nil,
		"Other metrics":      nil,
	}

	hasGit := false

	for _, d := range descriptors {
		if git.IsGitMetric(d.Name) {
			hasGit = true
		}

		groups[providerGroupLabel(d.Name)] = append(groups[providerGroupLabel(d.Name)], d)
	}

	content := &strings.Builder{}
	for _, label := range []string{
		"Filesystem metrics",
		"Git metrics",
		"Go metrics",
		"Other metrics",
	} {
		group := groups[label]
		if len(group) == 0 {
			continue
		}

		if content.Len() > 0 {
			content.WriteString("\n")
		}

		content.WriteString(label)
		content.WriteString("\n\n")

		tbl := table.New("Metric", "Kind", "Default Palette", "Description")
		tbl.SetMaxWidth(consoleWidth())

		for _, d := range group {
			desc := d.Description
			if git.IsGitMetric(d.Name) {
				desc += " †"
			}

			tbl.AddRow(string(d.Name), kindLabel(d.Kind), string(d.DefaultPalette), desc)
		}

		tbl.WriteTo(content)
	}

	fmt.Print(content.String())

	if hasGit {
		fmt.Printf("\n%s\n", "† requires a git repository")
	}

	return nil
}

func providerGroupLabel(name metric.Name) string {
	switch {
	case git.IsGitMetric(name):
		return "Git metrics"
	case golang.IsGoMetric(name):
		return "Go metrics"
	case name == filesystem.FileSize || name == filesystem.FileLines || name == filesystem.FileType:
		return "Filesystem metrics"
	default:
		return "Other metrics"
	}
}

// consoleWidth returns the width of the terminal, falling back to 120.
func consoleWidth() int {
	const defaultWidth = 120

	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w, err := strconv.Atoi(cols); err == nil && w > 0 {
			return w
		}
	}

	return defaultWidth
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
