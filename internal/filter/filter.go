package filter

import (
	"github.com/bmatcuk/doublestar/v4"
)

// Mode indicates whether a rule includes or excludes matching paths.
type Mode int

const (
	// Include means matching paths are included.
	Include Mode = iota
	// Exclude means matching paths are excluded.
	Exclude
)

// Rule pairs a glob pattern with an include/exclude mode.
type Rule struct {
	Pattern string `yaml:"pattern" json:"pattern"`
	Mode    Mode   `yaml:"mode"   json:"mode"`
}

// IsIncluded evaluates relativePath against rules in order.
// The first matching rule wins. Returns true if the entry should be included.
// Default (no match) is include.
func IsIncluded(relativePath string, rules []Rule) bool {
	for _, r := range rules {
		matched, err := doublestar.Match(r.Pattern, relativePath)
		if err != nil {
			// Invalid pattern → skip this rule
			continue
		}

		if matched {
			return r.Mode == Include
		}
	}

	return true
}
