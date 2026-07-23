package scan

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func TestFilterPolicy_Includes_NoRules_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := "/src"
	policy := newFilterPolicy(root, nil)

	included, _, err := policy.includes(filepath.Join(root, "main.go"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(included).To(BeTrue())
}

func TestFilterPolicy_Includes_ExcludeRule_MatchingPath_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := "/src"
	rules := []filter.Rule{
		{Pattern: "**/*.go", Mode: filter.Exclude},
	}
	policy := newFilterPolicy(root, rules)

	included, _, err := policy.includes(filepath.Join(root, "main.go"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(included).To(BeFalse())
}

func TestFilterPolicy_Includes_ExcludeRule_NonMatchingPath_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := "/src"
	rules := []filter.Rule{
		{Pattern: "**/*.go", Mode: filter.Exclude},
	}
	policy := newFilterPolicy(root, rules)

	included, _, err := policy.includes(filepath.Join(root, "README.md"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(included).To(BeTrue())
}

func TestFilterPolicy_Includes_ReturnsRelativePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := "/src"
	policy := newFilterPolicy(root, nil)

	_, rel, err := policy.includes(filepath.Join(root, "pkg", "util.go"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rel).To(Equal(filepath.Join("pkg", "util.go")))
}

func TestFilterPolicy_Includes_NestedPath_MatchedByExcludeRule(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := "/src"
	rules := []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
	}
	policy := newFilterPolicy(root, rules)

	// Dotfile in a subdirectory should match the unanchored exclude rule.
	included, _, err := policy.includes(filepath.Join(root, "sub", ".hidden"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(included).To(BeFalse())
}

func TestFilterPolicy_Includes_IncludeOverridesExclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := "/src"
	rules := []filter.Rule{
		{Pattern: ".config", Mode: filter.Include},
		{Pattern: ".*", Mode: filter.Exclude},
	}
	policy := newFilterPolicy(root, rules)

	// .config is explicitly included before the broad dotfile exclude.
	included, _, err := policy.includes(filepath.Join(root, ".config"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(included).To(BeTrue())
}
