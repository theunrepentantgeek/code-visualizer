package filter

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

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
	index   int
}

var ruleCounter atomic.Int64

// IsIncluded evaluates relativePath against rules in order.
// The first matching rule wins. Returns true if the entry should be included.
// Default (no match) is include.
func IsIncluded(relativePath string, rules []Rule) bool {
	for _, r := range rules {
		matched, err := MatchPattern(r.Pattern, relativePath)
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

// MatchPattern tests whether relativePath matches a glob pattern using
// gitignore-like semantics: patterns without a leading "/" or "**/" prefix
// are also tried at any depth (i.e. with an implicit "**/" prefix).
func MatchPattern(pattern, relativePath string) (bool, error) {
	matched, err := doublestar.Match(pattern, relativePath)
	if err != nil {
		return false, eris.Wrap(err, "Failed to match pattern")
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
		return false, eris.Wrap(err, "failed to match pattern")
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

	return NewRule(pattern, mode)
}

// NewRule validates a glob pattern and constructs a Rule with the given mode.
func NewRule(pattern string, mode Mode) (Rule, error) {
	if pattern == "" {
		return Rule{}, eris.New("empty filter pattern after prefix")
	}

	switch mode {
	case Include, Exclude:
	default:
		return Rule{}, eris.Errorf("unknown filter mode: %d", mode)
	}

	// Validate the glob pattern
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return Rule{}, eris.Wrapf(err, "invalid glob pattern %q", pattern)
	}

	return Rule{
		Pattern: pattern,
		Mode:    mode,
		index:   int(ruleCounter.Add(1)),
	}, nil
}

// CompareByIndex compares two rules by their internal construction index.
// For use with slices.SortFunc to recover original command-line order.
func CompareByIndex(a, b Rule) int {
	return a.index - b.index
}

// Merge combines include and exclude rule slices, sorting by construction
// order so the result matches original command-line flag order.
func Merge(include, exclude []Rule) []Rule {
	if len(include) == 0 && len(exclude) == 0 {
		return []Rule{}
	}

	result := make([]Rule, 0, len(include)+len(exclude))
	result = append(result, include...)
	result = append(result, exclude...)
	slices.SortFunc(result, CompareByIndex)

	return result
}
