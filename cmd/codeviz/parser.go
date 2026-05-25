package main

import (
	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func newParser(cli *CLI, options ...kong.Option) (*kong.Kong, error) {
	baseOptions := []kong.Option{
		kong.Name("codeviz"),
		kong.Description("Generate visualizations of file trees."),
		kong.NamedMapper(filter.RuleMapperName, filter.RuleMapper{}),
	}

	baseOptions = append(baseOptions, options...)

	return kong.New(cli, baseOptions...)
}
