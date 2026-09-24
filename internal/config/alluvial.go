package config

import "slices"

// Alluvial holds persistent configuration for alluvial visualizations.
type Alluvial struct {
	References []string    `yaml:"references,omitempty" json:"references,omitempty"`
	Metric     *string     `yaml:"metric,omitempty"     json:"metric,omitempty"`
	Fill       *MetricSpec `yaml:"fill,omitempty"    json:"fill,omitempty"`
	Expand     []string    `yaml:"expand,omitempty"     json:"expand,omitempty"`
}

// OverrideReferences replaces configured references when CLI references are supplied.
func (a *Alluvial) OverrideReferences(v []string) {
	overrideStrings(&a.References, v)
}

// OverrideMetric sets Metric to v if v is non-empty.
func (a *Alluvial) OverrideMetric(v string) { overrideString(&a.Metric, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (a *Alluvial) OverrideFill(v MetricSpec) { overrideMetricSpec(&a.Fill, v) }

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
