package filter

import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rotisserie/eris"
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
		matched, err := matchPattern(r.Pattern, relativePath)
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

func matchPattern(pattern, relativePath string) (bool, error) {
	matched, err := doublestar.Match(pattern, relativePath)
	if err != nil {
		return false, eris.Wrapf(err, "Failed to match pattern %q against path %q", pattern, relativePath)
	}

	if matched {
		return matched, nil
	}

	if strings.HasPrefix(pattern, "/") || strings.HasPrefix(pattern, "**/") {
		return false, nil
	}

	// For unanchored patterns, also match at any depth (gitignore-like behavior).
	matched, err = doublestar.Match("**/"+pattern, relativePath)
	if err != nil {
		return false, eris.Wrapf(err, "Failed to match pattern %q against path %q", pattern, relativePath)
	}

	return matched, nil
}

// MarshalText implements encoding.TextMarshaler.
func (m Mode) MarshalText() ([]byte, error) {
	switch m {
	case Include:
		return []byte("include"), nil
	case Exclude:
		return []byte("exclude"), nil
	default:
		return nil, fmt.Errorf("unknown filter mode: %d", m)
	}
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *Mode) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "include":
		*m = Include
	case "exclude":
		*m = Exclude
	default:
		return fmt.Errorf("unknown filter mode: %q", string(text))
	}

	return nil
}

// ParseFilterFlag parses a CLI filter string into a Rule.
// A leading ! marks an exclusion; anything else is an inclusion.
func ParseFilterFlag(s string) (Rule, error) {
	if s == "" {
		return Rule{}, eris.New("empty filter string")
	}

	mode := Include
	pattern := s

	if strings.HasPrefix(s, "!") {
		mode = Exclude
		pattern = s[1:]
	}

	if pattern == "" {
		return Rule{}, eris.New("empty filter pattern after prefix")
	}

	// Validate the glob pattern
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return Rule{}, eris.Wrapf(err, "invalid glob pattern %q", pattern)
	}

	return Rule{Pattern: pattern, Mode: mode}, nil
}
