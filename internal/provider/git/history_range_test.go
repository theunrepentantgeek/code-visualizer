package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

type tagRangeFixture struct {
	dir      string
	initial  string
	main     string
	feature  string
	merged   string
	detached string
}

func setupTagRangeRepo(t *testing.T) tagRangeFixture {
	t.Helper()
	g := NewGomegaWithT(t)
	dir := t.TempDir()

	run := func(args ...string) string {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // fixed test commands
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}

		return strings.TrimSpace(string(out))
	}

	writeCommit := func(name, contents, message, date string) string {
		t.Helper()
		g.Expect(os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600)).To(Succeed())
		run("git", "add", name)
		run("git", "commit", "-m", message, "--date="+date)

		return run("git", "rev-parse", "HEAD")
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	initial := writeCommit("shared.go", "package shared\n", "A", "2025-01-01T00:00:00Z")

	run("git", "tag", "v1.0")

	mainCommit := writeCommit("main.go", "package main\n", "B", "2025-02-01T00:00:00Z")

	run("git", "switch", "-c", "feature")

	feature := writeCommit("feature.go", "package feature\n", "C", "2025-03-01T00:00:00Z")
	writeCommit("feature.go", "package feature\n// D\n", "D", "2025-04-01T00:00:00Z")

	run("git", "switch", "main")
	run("git", "merge", "--no-ff", "feature", "-m", "M")
	merged := run("git", "rev-parse", "HEAD")
	run("git", "tag", "-a", "v2.0", "-m", "release v2")
	blob := run("git", "hash-object", "-w", "shared.go")
	run("git", "tag", "-a", "blob-tag", blob, "-m", "non-commit tag")

	run("git", "switch", "--orphan", "detached-line")

	detached := writeCommit("detached.go", "package detached\n", "U", "2025-05-01T00:00:00Z")

	run("git", "tag", "detached")
	run("git", "switch", "main")

	return tagRangeFixture{
		dir:      dir,
		initial:  initial,
		main:     mainCommit,
		feature:  feature,
		merged:   merged,
		detached: detached,
	}
}

func TestHistoryRange_FromRevisionIsExclusiveAndUntilRevisionIsInclusive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	commits, err := s.commitIterator(HistoryRange{From: "v1.0", Until: "v2.0"})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string

	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())

		hashes = append(hashes, commit.Hash.String())
	}

	g.Expect(hashes).To(ContainElements(fixture.main, fixture.feature, fixture.merged))
	g.Expect(hashes).NotTo(ContainElement(fixture.initial))
}

func TestHistoryRange_UntilRevisionCanBeOutsideHeadAncestry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	commits, err := s.commitIterator(HistoryRange{Until: "detached"})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string

	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())

		hashes = append(hashes, commit.Hash.String())
	}

	g.Expect(hashes).To(Equal([]string{fixture.detached}))
}

func TestHistoryRange_RejectsFromRevisionOutsideTipAncestry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	_, err = s.commitIterator(HistoryRange{From: "detached", Until: "v2.0"})
	g.Expect(err).To(MatchError(ContainSubstring(`history reference "detached" is not an ancestor of "v2.0"`)))
}

func TestHistoryRange_ReportsInvalidTags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	_, err = s.commitIterator(HistoryRange{Until: "tag:missing"})
	g.Expect(err).To(MatchError(ContainSubstring(`tag "missing" not found`)))

	_, err = s.commitIterator(HistoryRange{Until: "tag:blob-tag"})
	g.Expect(err).To(MatchError(ContainSubstring(`tag "blob-tag" does not reference a commit`)))
}

func TestHistoryRange_MixesRevisionAndDateBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	commits, err := s.commitIterator(HistoryRange{
		From:  "tag:v1.0",
		Until: "date:2025-03-01T23:59:59Z",
	})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string

	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())

		hashes = append(hashes, commit.Hash.String())
	}

	g.Expect(hashes).To(ContainElements(fixture.main, fixture.feature))
	g.Expect(hashes).NotTo(ContainElement(fixture.initial))
}

func TestHistoryRange_RejectsReversedDateRange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	_, err = s.commitIterator(HistoryRange{From: "2025-03-01", Until: "2025-01-01"})
	g.Expect(err).To(MatchError(ContainSubstring("--from must be before or equal to --until")))
}
