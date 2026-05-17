package git

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestBulkCommitHistory_ReturnsCommitsForTrackedFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{
		"old.go":    true,
		"shared.go": true,
		"new.go":    true,
	}

	commits, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).NotTo(BeEmpty())

	for _, c := range commits {
		g.Expect(c.Hash).NotTo(BeEmpty())
		g.Expect(c.Author.Name).NotTo(BeEmpty())
		g.Expect(c.Author.When.IsZero()).To(BeFalse())
		g.Expect(c.ChangedPaths).NotTo(BeEmpty())
	}
}

func TestBulkCommitHistory_CapturesAuthorIdentity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"shared.go": true}

	commits, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	authors := map[string]bool{}
	for _, c := range commits {
		authors[c.Author.Name] = true
	}

	g.Expect(authors).To(HaveKey("Alice"))
	g.Expect(authors).To(HaveKey("Bob"))
}

func TestBulkCommitHistory_SkipsCommitsNotTouchingTracked(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"old.go": true}

	commits, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	for _, c := range commits {
		g.Expect(c.ChangedPaths).To(ContainElement("old.go"))
	}
}

func TestBulkCommitHistory_InvokesProgressCallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"old.go": true, "shared.go": true, "new.go": true}

	count := 0

	_, err := BulkCommitHistory(dir, tracked, func() { count++ })
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(count).To(BeNumerically(">=", 1))
}
