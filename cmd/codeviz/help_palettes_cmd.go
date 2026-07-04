package main

import (
	"fmt"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// HelpPalettesCmd prints a list of all available colour palettes.
type HelpPalettesCmd struct{}

const palettesDocURL = "https://github.com/theunrepentantgeek/code-visualizer/blob/main/docs/palettes.md"

//nolint:unparam // nil error required to satisfy the interface for Kong
func (HelpPalettesCmd) Run(_ *Flags) error {
	infos := palette.Infos()

	entries := make([]nameDescription, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, nameDescription{
			Name:        string(info.Name),
			Description: info.Description,
		})
	}

	fmt.Print(renderNameDescriptionList("Palettes", entries, consoleWidth()))

	fmt.Printf("For colour swatches, see: %s\n", palettesDocURL)

	return nil
}
