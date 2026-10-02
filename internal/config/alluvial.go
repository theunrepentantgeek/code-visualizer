package config

import "slices"

// ConstantBandsMode controls how alluvial bands with unchanged widths are displayed.
type ConstantBandsMode string

const (
	ConstantBandsHide  ConstantBandsMode = "hide"
	ConstantBandsMute  ConstantBandsMode = "mute"
	ConstantBandsMerge ConstantBandsMode = "merge"
)

// Alluvial holds persistent configuration for alluvial visualizations.
type Alluvial struct {
	References    []string           `yaml:"references,omitempty"    json:"references,omitempty"`
	Metric        *string            `yaml:"metric,omitempty"        json:"metric,omitempty"`
	Fill          *MetricSpec        `yaml:"fill,omitempty"          json:"fill,omitempty"`
	Expand        []string           `yaml:"expand,omitempty"        json:"expand,omitempty"`
	ConstantBands *ConstantBandsMode `yaml:"constantBands,omitempty" json:"constantBands,omitempty"`
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

// ConstantBandsMode returns the configured mode, or the zero value when omitted.
func (a *Alluvial) ConstantBandsMode() ConstantBandsMode {
	if a == nil || a.ConstantBands == nil {
		return ""
	}

	return *a.ConstantBands
}

// OverrideConstantBands sets ConstantBands when a CLI value is supplied.
func (a *Alluvial) OverrideConstantBands(v string) {
	if v == "" {
		return
	}

	mode := ConstantBandsMode(v)
	a.ConstantBands = &mode
}

func overrideStrings(target *[]string, values []string) {
	if len(values) == 0 {
		return
	}

	*target = slices.Clone(values)
}
