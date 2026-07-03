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

func TestIsIncluded_SlashlessPatternMatchesNestedPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: ".*", Mode: Exclude}}

	g.Expect(IsIncluded("docs/.cache", rules)).To(BeFalse())
	g.Expect(IsIncluded("docs/notes.md", rules)).To(BeTrue())
}

func TestIsIncluded_SuperpowersPatternMatchesNestedDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "superpowers/**", Mode: Exclude}}

	g.Expect(IsIncluded("docs/superpowers", rules)).To(BeFalse())
	g.Expect(IsIncluded("docs/superpowers/specs/design.md", rules)).To(BeFalse())
	g.Expect(IsIncluded("docs/specs/design.md", rules)).To(BeTrue())
}

func TestIsIncluded_InvalidPattern_TreatedAsNoMatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "[invalid", Mode: Exclude}}

	// Invalid patterns never match, so default (include) applies
	g.Expect(IsIncluded("anything", rules)).To(BeTrue())
}

func TestModeMarshaling_Include(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text, err := Include.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(text)).To(Equal("include"))

	var m Mode
	g.Expect(m.UnmarshalText([]byte("include"))).To(Succeed())
	g.Expect(m).To(Equal(Include))
}

func TestModeMarshaling_Exclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text, err := Exclude.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(text)).To(Equal("exclude"))

	var m Mode
	g.Expect(m.UnmarshalText([]byte("exclude"))).To(Succeed())
	g.Expect(m).To(Equal(Exclude))
}

func TestModeUnmarshaling_Invalid(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var m Mode
	g.Expect(m.UnmarshalText([]byte("bogus"))).To(HaveOccurred())
}

func TestParseFilterFlag_ExcludeWithBang(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := ParseFilterFlag("!.*")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rule.Pattern).To(Equal(".*"))
	g.Expect(rule.Mode).To(Equal(Exclude))
}

func TestParseFilterFlag_IncludeWithoutPrefix(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := ParseFilterFlag("*.go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rule.Pattern).To(Equal("*.go"))
	g.Expect(rule.Mode).To(Equal(Include))
}

func TestParseFilterFlag_DoublestarPattern(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := ParseFilterFlag("!**/*.log")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rule.Pattern).To(Equal("**/*.log"))
	g.Expect(rule.Mode).To(Equal(Exclude))
}

func TestParseFilterFlag_InvalidGlob(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseFilterFlag("![invalid")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid glob pattern"))
}

func TestParseFilterFlag_EmptyString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseFilterFlag("")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty filter"))
}

func TestParseFilterFlag_BangOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseFilterFlag("!")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty filter"))
}

func TestCompareByIndex_ReturnsNegativeForEarlierRule(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a, err := NewRule("*.go", Include)
	g.Expect(err).NotTo(HaveOccurred())

	b, err := NewRule("*.log", Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(CompareByIndex(a, b)).To(BeNumerically("<", 0))
	g.Expect(CompareByIndex(b, a)).To(BeNumerically(">", 0))
	g.Expect(CompareByIndex(a, a)).To(Equal(0))
}

func TestMerge_PreservesConstructionOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Create rules in a specific interleaved order
	excl1, err := NewRule(".*", Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	incl1, err := NewRule(".github/**", Include)
	g.Expect(err).NotTo(HaveOccurred())

	excl2, err := NewRule("**/*.log", Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	include := []Rule{incl1}
	exclude := []Rule{excl1, excl2}

	merged := Merge(include, exclude)

	g.Expect(merged).To(HaveLen(3))
	g.Expect(merged[0].Pattern).To(Equal(".*"))
	g.Expect(merged[0].Mode).To(Equal(Exclude))
	g.Expect(merged[1].Pattern).To(Equal(".github/**"))
	g.Expect(merged[1].Mode).To(Equal(Include))
	g.Expect(merged[2].Pattern).To(Equal("**/*.log"))
	g.Expect(merged[2].Mode).To(Equal(Exclude))
}

func TestMerge_EmptySlices_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Merge([]Rule{}, []Rule{})).To(BeEmpty())
}

func TestValidatePattern_ValidPatterns(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, pattern := range []string{"*", "**", "*.go", "*_test.go", "testdata/**", "**/*.go"} {
		g.Expect(ValidatePattern(pattern)).To(Succeed(), "expected pattern %q to be valid", pattern)
	}
}

func TestValidatePattern_EmptyPattern_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(ValidatePattern("")).To(MatchError(ContainSubstring("empty glob pattern")))
}
