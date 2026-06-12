package main

import "github.com/alecthomas/kong"

type HelpCmd struct {
	Default  HelpDefaultCmd  `cmd:"" hidden:"" default:"1"`
	Metrics  HelpMetricsCmd  `cmd:"" help:"List all available metrics."`
	Palettes HelpPalettesCmd `cmd:"" help:"List all available colour palettes."`
}

type HelpDefaultCmd struct{}

func (HelpDefaultCmd) Run(realCtx *kong.Context) error {
	ctx, err := kong.Trace(realCtx.Kong, nil) // nil path => application root
	if err != nil {
		//nolint:wrapcheck // Returning the Kong usage error directly is fine here
		return err
	}

	if ctx.Error != nil {
		return ctx.Error
	}

	//nolint:wrapcheck // Returning the Kong usage error directly is fine here
	return ctx.PrintUsage(false)
}
