package git

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseMailmap_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mm := parseMailmap(strings.NewReader(""))
	g.Expect(mm).To(BeEmpty())
}

func TestParseMailmap_CommentsAndBlanks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	input := "# This is a comment\n\n# Another comment\n"
	mm := parseMailmap(strings.NewReader(input))
	g.Expect(mm).To(BeEmpty())
}

func TestParseMailmap_ReEmailOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "<proper@email.com> <old@email.com>" — re-email, keep name.
	mm := parseMailmap(strings.NewReader("<proper@example.com> <old@example.com>"))
	email, name := mm.apply("old@example.com", "Old Name")

	g.Expect(email).To(Equal("proper@example.com"))
	g.Expect(name).To(Equal("Old Name")) // name unchanged
}

func TestParseMailmap_RenameAndReEmail(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "Proper Name <proper@email.com> <old@email.com>"
	mm := parseMailmap(strings.NewReader("Proper Name <proper@example.com> <old@example.com>"))
	email, name := mm.apply("old@example.com", "Old Name")

	g.Expect(email).To(Equal("proper@example.com"))
	g.Expect(name).To(Equal("Proper Name"))
}

func TestParseMailmap_RenameWithOldName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "Proper Name <proper@email.com> Old Name <old@email.com>"
	mm := parseMailmap(strings.NewReader("Proper Name <proper@example.com> Old Name <old@example.com>"))
	email, name := mm.apply("old@example.com", "Old Name")

	g.Expect(email).To(Equal("proper@example.com"))
	g.Expect(name).To(Equal("Proper Name"))
}

func TestParseMailmap_CaseInsensitiveEmailLookup(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mm := parseMailmap(strings.NewReader("Proper Name <proper@example.com> <OLD@EXAMPLE.COM>"))
	email, name := mm.apply("old@example.com", "Someone")

	g.Expect(email).To(Equal("proper@example.com"))
	g.Expect(name).To(Equal("Proper Name"))
}

func TestParseMailmap_UnknownEmailPassesThrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mm := parseMailmap(strings.NewReader("<proper@example.com> <old@example.com>"))
	email, name := mm.apply("other@example.com", "Other")

	g.Expect(email).To(Equal("other@example.com"))
	g.Expect(name).To(Equal("Other"))
}

func TestMailmap_ApplyOnEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var mm mailmap // nil map

	email, name := mm.apply("a@b.com", "Alice")

	g.Expect(email).To(Equal("a@b.com"))

	g.Expect(name).To(Equal("Alice"))
}

func TestParseMailmap_MultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	input := `
# remap two authors
Proper Alice <alice@new.com> <alice@old.com>
Proper Bob <bob@new.com> <bob@old.com>
`
	mm := parseMailmap(strings.NewReader(input))

	aEmail, aName := mm.apply("alice@old.com", "Alice")
	bEmail, bName := mm.apply("bob@old.com", "Bob")

	g.Expect(aEmail).To(Equal("alice@new.com"))
	g.Expect(aName).To(Equal("Proper Alice"))
	g.Expect(bEmail).To(Equal("bob@new.com"))
	g.Expect(bName).To(Equal("Proper Bob"))
}

func TestParseMailmap_InlineCommentStripped(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mm := parseMailmap(strings.NewReader("<p@example.com> <o@example.com> # inline comment"))
	email, _ := mm.apply("o@example.com", "")

	g.Expect(email).To(Equal("p@example.com"))
}
