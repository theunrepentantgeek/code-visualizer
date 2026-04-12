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
