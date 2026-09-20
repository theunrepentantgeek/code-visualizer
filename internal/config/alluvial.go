package config

import "slices"

// Alluvial holds persistent configuration for alluvial visualizations.
type Alluvial struct {
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
	Metric     *string  `yaml:"metric,omitempty"     json:"metric,omitempty"`
	Expand     []string `yaml:"expand,omitempty"     json:"expand,omitempty"`
}

// OverrideReferences replaces configured references when CLI references are supplied.
func (a *Alluvial) OverrideReferences(v []string) {
	overrideStrings(&a.References, v)
}

// OverrideMetric sets Metric to v if v is non-empty.
func (a *Alluvial) OverrideMetric(v string) { overrideString(&a.Metric, v) }

// OverrideExpand replaces configured expansions when CLI expansions are supplied.
func (a *Alluvial) OverrideExpand(v []string) {
	overrideStrings(&a.Expand, v)
}

func overrideStrings(target *[]string, values []string) {
	if len(values) == 0 {
		return
	}

	*target = slices.Clone(values)
}
