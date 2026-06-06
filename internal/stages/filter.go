package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

// BuildFilterRulesHelper merges config-file filter rules with CLI filter
// flags. CLI filters must already have been syntax-validated by the
// command's Validate() method.
func BuildFilterRulesHelper(cfg *config.Config, cliFilters []filter.Rule) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(cliFilters))
	rules = append(rules, cfg.FileFilter...)
	rules = append(rules, cliFilters...)

	return rules
}

// BuildFilterRules populates c.FilterRules from c.RootConfig.FileFilter plus
// c.CLIFilters.
func BuildFilterRules(c *CommonState) error {
	c.FilterRules = BuildFilterRulesHelper(c.RootConfig, c.CLIFilters)

	return nil
}
