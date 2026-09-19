package config

// Alluvial holds persistent configuration for alluvial visualizations.
type Alluvial struct {
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
	Metric     *string  `yaml:"metric,omitempty"     json:"metric,omitempty"`
	Expand     []string `yaml:"expand,omitempty"     json:"expand,omitempty"`
}

// OverrideReferences replaces configured references when CLI references are supplied.
func (a *Alluvial) OverrideReferences(v []string) {
	if len(v) > 0 {
		a.References = append([]string(nil), v...)
	}
}

// OverrideMetric sets Metric to v if v is non-empty.
func (a *Alluvial) OverrideMetric(v string) { overrideString(&a.Metric, v) }

// OverrideExpand replaces configured expansions when CLI expansions are supplied.
func (a *Alluvial) OverrideExpand(v []string) {
	if len(v) > 0 {
		a.Expand = append([]string(nil), v...)
	}
}
