package main

import (
	"fmt"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// HelpPalettesCmd prints a list of all available colour palettes.
type HelpPalettesCmd struct{}

const palettesDocURL = "https://github.com/theunrepentantgeek/code-visualizer/blob/main/docs/palettes.md"

func (HelpPalettesCmd) Run(flags *Flags) error {
	infos := palette.Infos()

	entries := make([]nameDescription, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, nameDescription{
			Name:        string(info.Name),
			Description: info.Description,
		})
	}

	content := renderNameDescriptionList("Palettes", entries, consoleWidth())
	if _, err := fmt.Fprint(flags.stdoutWriter(), content); err != nil {
		return eris.Wrap(err, "write palette help")
	}

	_, err := fmt.Fprintf(flags.stdoutWriter(), "For colour swatches, see: %s\n", palettesDocURL)

	return eris.Wrap(err, "write palette documentation link")
}
