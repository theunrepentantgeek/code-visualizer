package main

import (
	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func filterMapperOption() kong.Option {
	return kong.NamedMapper(filter.RuleMapperName, filter.RuleMapper{})
}
