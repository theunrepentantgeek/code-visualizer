package filter

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestIsIncluded_NoRules_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsIncluded("anything.go", nil)).To(BeTrue())
}

func TestIsIncluded_SingleExclude_MatchesEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: ".*", Mode: Exclude}}

	g.Expect(IsIncluded(".git", rules)).To(BeFalse())
	g.Expect(IsIncluded(".gitignore", rules)).To(BeFalse())
}

func TestIsIncluded_SingleExclude_NoMatch_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: ".*", Mode: Exclude}}

	g.Expect(IsIncluded("main.go", rules)).To(BeTrue())
	g.Expect(IsIncluded("src/main.go", rules)).To(BeTrue())
}

func TestIsIncluded_SingleInclude_MatchesEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "*.go", Mode: Include}}

	g.Expect(IsIncluded("main.go", rules)).To(BeTrue())
}

func TestIsIncluded_FirstMatchWins_IncludeBeforeExclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{
		{Pattern: ".github", Mode: Include},
		{Pattern: ".github/**", Mode: Include},
		{Pattern: ".*", Mode: Exclude},
	}

	g.Expect(IsIncluded(".github", rules)).To(BeTrue())
	g.Expect(IsIncluded(".github/workflows/ci.yml", rules)).To(BeTrue())
	g.Expect(IsIncluded(".git", rules)).To(BeFalse())
}

func TestIsIncluded_FirstMatchWins_ExcludeBeforeInclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{
		{Pattern: ".*", Mode: Exclude},
		{Pattern: ".github/**", Mode: Include},
	}

	// .github matches .* first → excluded
	g.Expect(IsIncluded(".github", rules)).To(BeFalse())
}

func TestIsIncluded_DoublestarPattern(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "**/*.log", Mode: Exclude}}

	g.Expect(IsIncluded("src/debug.log", rules)).To(BeFalse())
	g.Expect(IsIncluded("src/main.go", rules)).To(BeTrue())
	g.Expect(IsIncluded("debug.log", rules)).To(BeFalse())
}

func TestIsIncluded_InvalidPattern_TreatedAsNoMatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "[invalid", Mode: Exclude}}

	// Invalid patterns never match, so default (include) applies
	g.Expect(IsIncluded("anything", rules)).To(BeTrue())
}
