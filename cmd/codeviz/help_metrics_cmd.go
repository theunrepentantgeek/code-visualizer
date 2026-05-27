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

const (
	filesystemMetricsSection = "Filesystem metrics"
	gitMetricsSection        = "Git metrics"
	goMetricsSection         = "Go metrics"
	otherMetricsSection      = "Other metrics"
)

var providerSectionOrder = []string{
	filesystemMetricsSection,
	gitMetricsSection,
	goMetricsSection,
	otherMetricsSection,
}

//nolint:unparam // nil error required to satisfy the interface for Kong
func (HelpMetricsCmd) Run(_ *Flags) error {
	descriptors := provider.AllDescriptors()
	groups := buildProviderGroups(descriptors)
	fmt.Print(renderProviderGroups(groups))

	return nil
}

func buildProviderGroups(
	descriptors []provider.MetricDescriptor,
) map[string][]provider.MetricDescriptor {
	groups := map[string][]provider.MetricDescriptor{
		filesystemMetricsSection: nil,
		gitMetricsSection:        nil,
		goMetricsSection:         nil,
		otherMetricsSection:      nil,
	}

	for _, d := range descriptors {
		label := providerGroupLabel(d.Name)
		groups[label] = append(groups[label], d)
	}

	return groups
}

func renderProviderGroups(groups map[string][]provider.MetricDescriptor) string {
	content := &strings.Builder{}

	for _, label := range providerSectionOrder {
		group := groups[label]
		if len(group) == 0 {
			continue
		}

		if content.Len() > 0 {
			content.WriteString("\n")
		}

		content.WriteString(label)
		content.WriteString("\n\n")

		writeProviderGroupTable(content, group)
	}

	return content.String()
}

func writeProviderGroupTable(content *strings.Builder, group []provider.MetricDescriptor) {
	tbl := table.New("Metric", "Kind", "Default Palette", "Description")
	tbl.SetMaxWidth(consoleWidth())

	for _, d := range group {
		desc := d.Description
		tbl.AddRow(string(d.Name), kindLabel(d.Kind), string(d.DefaultPalette), desc)
	}

	tbl.WriteTo(content)
}

func providerGroupLabel(name metric.Name) string {
	switch {
	case git.IsGitMetric(name):
		return gitMetricsSection
	case golang.IsGoMetric(name):
		return goMetricsSection
	case filesystem.IsFilesystemMetric(name):
		return filesystemMetricsSection
	default:
		return otherMetricsSection
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
