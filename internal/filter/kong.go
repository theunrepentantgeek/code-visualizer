package filter

import (
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"
)

const RuleMapperName = "filterrule"

// RuleMapper decodes --include/--exclude flags into filter rules.
// Mode is inferred from the flag name; construction order is captured
// by the index assigned in NewRule().
type RuleMapper struct{}

func (RuleMapper) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	var pattern string
	if err := ctx.Scan.PopValueInto("pattern", &pattern); err != nil {
		return eris.Wrapf(err, "failed to read filter pattern for %q", ctx.Value.Name)
	}

	var mode Mode

	switch ctx.Value.Name {
	case "include":
		mode = Include
	case "exclude":
		mode = Exclude
	default:
		return eris.Errorf("unexpected filter flag name %q", ctx.Value.Name)
	}

	rule, err := NewRule(pattern, mode)
	if err != nil {
		return eris.Wrapf(err, "invalid %s %q", ctx.Value.Name, pattern)
	}

	target.Set(reflect.Append(target, reflect.ValueOf(rule)))

	return nil
}
