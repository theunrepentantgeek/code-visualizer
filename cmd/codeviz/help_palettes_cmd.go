package main

import (
	"fmt"
	"strings"

	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/table"
)

// HelpPalettesCmd prints a table of all available colour palettes.
type HelpPalettesCmd struct{}

const palettesDocURL = "https://github.com/theunrepentantgeek/code-visualizer/blob/main/docs/palettes.md"

//nolint:unparam // nil error required to satisfy the interface for Kong
func (HelpPalettesCmd) Run(_ *Flags) error {
	infos := palette.Infos()

	tbl := table.New("Palette", "Description")

	for _, info := range infos {
		tbl.AddRow(string(info.Name), info.Description)
	}

	content := &strings.Builder{}

	tbl.WriteTo(content)

	fmt.Print(content.String())

	fmt.Printf("\nFor colour swatches, see: %s\n", palettesDocURL)

	return nil
}
