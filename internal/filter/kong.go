package filter

import (
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"
)

const RuleMapperName = "filterrule"

// RuleMapper decodes --include/--exclude flags into filter rules.
type RuleMapper struct{}

func (RuleMapper) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	var pattern string
	if err := ctx.Scan.PopValueInto("pattern", &pattern); err != nil {
		return err
	}

	mode := Include
	if ctx.Value.Name == "exclude" {
		mode = Exclude
	}

	rule, err := NewRule(pattern, mode)
	if err != nil {
		return eris.Wrapf(err, "invalid %s %q", ctx.Value.Name, pattern)
	}

	target.Set(reflect.Append(target, reflect.ValueOf(rule)))

	return nil
}

// MergeFlagRules merges parsed include/exclude rules in the order their flags
// appeared on the command line.
func MergeFlagRules(kctx *kong.Context, includes []Rule, excludes []Rule) ([]Rule, error) {
	if kctx == nil {
		rules := make([]Rule, 0, len(includes)+len(excludes))
		rules = append(rules, includes...)
		rules = append(rules, excludes...)

		return rules, nil
	}

	rules := make([]Rule, 0, len(includes)+len(excludes))
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

			rules = append(rules, includes[includeIndex])
			includeIndex++
		case "exclude":
			if excludeIndex >= len(excludes) {
				return nil, eris.New("failed to reconcile exclude flags")
			}

			rules = append(rules, excludes[excludeIndex])
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
