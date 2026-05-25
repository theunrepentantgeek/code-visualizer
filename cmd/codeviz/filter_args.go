package main

import (
	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func buildOrderedFilters(kctx *kong.Context, includes []string, excludes []string) ([]filter.Rule, error) {
	if kctx == nil {
		return buildFiltersWithoutContext(includes, excludes)
	}

	rules := make([]filter.Rule, 0, len(includes)+len(excludes))
	includeIndex := 0
	excludeIndex := 0

	for _, path := range kctx.Path {
		if path.Flag == nil || path.Resolved {
			continue
		}

		switch path.Flag.Name {
		case "include":
			if includeIndex >= len(includes) {
				return nil, eris.New("failed to reconcile include flags")
			}

			rule, err := filter.NewRule(includes[includeIndex], filter.Include)
			if err != nil {
				return nil, eris.Wrapf(err, "invalid include %q", includes[includeIndex])
			}

			rules = append(rules, rule)
			includeIndex++
		case "exclude":
			if excludeIndex >= len(excludes) {
				return nil, eris.New("failed to reconcile exclude flags")
			}

			rule, err := filter.NewRule(excludes[excludeIndex], filter.Exclude)
			if err != nil {
				return nil, eris.Wrapf(err, "invalid exclude %q", excludes[excludeIndex])
			}

			rules = append(rules, rule)
			excludeIndex++
		}
	}

	if includeIndex != len(includes) {
		return nil, eris.New("failed to reconcile include flags")
	}

	if excludeIndex != len(excludes) {
		return nil, eris.New("failed to reconcile exclude flags")
	}

	return rules, nil
}

func buildFiltersWithoutContext(includes []string, excludes []string) ([]filter.Rule, error) {
	rules := make([]filter.Rule, 0, len(includes)+len(excludes))

	for _, include := range includes {
		rule, err := filter.NewRule(include, filter.Include)
		if err != nil {
			return nil, eris.Wrapf(err, "invalid include %q", include)
		}

		rules = append(rules, rule)
	}

	for _, exclude := range excludes {
		rule, err := filter.NewRule(exclude, filter.Exclude)
		if err != nil {
			return nil, eris.Wrapf(err, "invalid exclude %q", exclude)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}
