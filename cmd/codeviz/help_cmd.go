package main

import (
	"fmt"
	"strings"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
)

// HelpCmd groups the help subcommands.
type HelpCmd struct {
	Metrics  HelpMetricsCmd  `cmd:"" help:"List all available metrics."`
	Palettes HelpPalettesCmd `cmd:"" help:"List all available colour palettes."`
}

// HelpMetricsCmd prints a table of all registered metrics.
type HelpMetricsCmd struct{}

// gitMetricNames is the set of metrics that require a git repository.
var gitMetricNames = map[metric.Name]bool{
	"file-age": true, "file-freshness": true, "author-count": true,
}

func (HelpMetricsCmd) Run(_ *Flags) error {
	providers := provider.All()

	const (
		nameHeader = "METRIC"
		kindHeader = "KIND"
		descHeader = "DESCRIPTION"
		gitNote    = "† requires a git repository"
	)

	nameWidth := len(nameHeader)
	kindWidth := len(kindHeader)

	for _, p := range providers {
		if n := len(p.Name()); n > nameWidth {
			nameWidth = n
		}

		k := kindLabel(p.Kind())
		if n := len(k); n > kindWidth {
			kindWidth = n
		}
	}

	fmt.Printf("%-*s  %-*s  %s\n", nameWidth, nameHeader, kindWidth, kindHeader, descHeader)
	fmt.Printf("%s  %s  %s\n", strings.Repeat("-", nameWidth), strings.Repeat("-", kindWidth), strings.Repeat("-", len(descHeader)))

	hasGit := false

	for _, p := range providers {
		k := kindLabel(p.Kind())
		desc := p.Description()
		isGit := gitMetricNames[p.Name()]

		if isGit {
			hasGit = true
			desc += " †"
		}

		fmt.Printf("%-*s  %-*s  %s\n", nameWidth, p.Name(), kindWidth, k, desc)
	}

	if hasGit {
		fmt.Printf("\n%s\n", gitNote)
	}

	return nil
}

// HelpPalettesCmd prints a table of all available colour palettes.
type HelpPalettesCmd struct{}

const palettesDocURL = "https://github.com/theunrepentantgeek/code-visualizer/blob/main/docs/palettes.md"

func (HelpPalettesCmd) Run(_ *Flags) error {
	infos := palette.Infos()

	const (
		nameHeader = "PALETTE"
		descHeader = "DESCRIPTION"
	)

	nameWidth := len(nameHeader)

	for _, info := range infos {
		if n := len(info.Name); n > nameWidth {
			nameWidth = n
		}
	}

	fmt.Printf("%-*s  %s\n", nameWidth, nameHeader, descHeader)
	fmt.Printf("%s  %s\n", strings.Repeat("-", nameWidth), strings.Repeat("-", len(descHeader)))

	for _, info := range infos {
		fmt.Printf("%-*s  %s\n", nameWidth, info.Name, info.Description)
	}

	fmt.Printf("\nFor colour swatches, see: %s\n", palettesDocURL)

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
