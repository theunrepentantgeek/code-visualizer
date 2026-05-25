package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
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

// BuildFilterRules is a pipeline.Stage that populates Common().FilterRules
// from Common().RootConfig.FileFilter plus Common().CLIFilters.
func BuildFilterRules[S VizState](s S) error {
	c := s.Common()
	c.FilterRules = BuildFilterRulesHelper(c.RootConfig, c.CLIFilters)

	return nil
}

var _ pipeline.Stage[VizState] = BuildFilterRules[VizState]
