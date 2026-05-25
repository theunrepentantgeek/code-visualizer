package filter

import (
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"
)

const RuleMapperName = "filterrule"

var ruleSliceType = reflect.TypeOf([]Rule{})

type ruleBinding struct {
	filters *[]Rule
	mode    Mode
}

// RuleMapper decodes --include/--exclude flags into filter rules.
type RuleMapper struct {
	bindings map[uintptr]ruleBinding
}

func NewRuleMapper(root any) RuleMapper {
	mapper := RuleMapper{
		bindings: make(map[uintptr]ruleBinding),
	}

	mapper.bindValue(reflect.ValueOf(root))

	return mapper
}

func (m RuleMapper) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	var pattern string
	if err := ctx.Scan.PopValueInto("pattern", &pattern); err != nil {
		return err
	}

	binding, ok := m.bindings[ctx.Value.Target.Addr().Pointer()]
	if !ok {
		return eris.Errorf("filter rule mapper not bound for %q", ctx.Value.Name)
	}

	rule, err := NewRule(pattern, binding.mode)
	if err != nil {
		return eris.Wrapf(err, "invalid %s %q", ctx.Value.Name, pattern)
	}

	target.Set(reflect.Append(target, reflect.ValueOf(rule)))
	*binding.filters = append(*binding.filters, rule)

	return nil
}

func (m RuleMapper) bindValue(value reflect.Value) {
	if !value.IsValid() {
		return
	}

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return
	}

	m.bindStruct(value)

	for i := range value.NumField() {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Pointer, reflect.Struct:
			m.bindValue(field)
		}
	}
}

func (m RuleMapper) bindStruct(value reflect.Value) {
	include := value.FieldByName("Include")
	exclude := value.FieldByName("Exclude")
	filters := value.FieldByName("Filters")

	if !include.IsValid() || !exclude.IsValid() || !filters.IsValid() {
		return
	}

	if include.Type() != ruleSliceType || exclude.Type() != ruleSliceType || filters.Type() != ruleSliceType {
		return
	}

	filtersPtr, ok := filters.Addr().Interface().(*[]Rule)
	if !ok {
		return
	}

	m.bindings[include.Addr().Pointer()] = ruleBinding{
		filters: filtersPtr,
		mode:    Include,
	}
	m.bindings[exclude.Addr().Pointer()] = ruleBinding{
		filters: filtersPtr,
		mode:    Exclude,
	}
}
