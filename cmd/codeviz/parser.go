package main

import "github.com/alecthomas/kong"

func filterMapperOption() kong.Option {
	return kong.NamedMapper(ruleMapperName, ruleMapper{})
}
