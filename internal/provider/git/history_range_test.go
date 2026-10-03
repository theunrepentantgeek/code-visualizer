package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestHistoryRangeWithTimeNow_PreservesExistingTime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	referenceTime := time.Date(2026, 10, 3, 3, 20, 0, 0, time.UTC)

	historyRange := (HistoryRange{
		From:    "ago(14d)",
		Until:   "HEAD",
		TimeNow: referenceTime,
	}).withTimeNow()

	g.Expect(historyRange.TimeNow).To(Equal(referenceTime))
}

func TestHistoryRangeWithTimeNow_CapturesCurrentTimeOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	before := time.Now()

	historyRange := (HistoryRange{From: "ago(14d)", Until: "HEAD"}).withTimeNow()

	after := time.Now()
	g.Expect(historyRange.TimeNow).To(BeTemporally(">=", before))
	g.Expect(historyRange.TimeNow).To(BeTemporally("<=", after))
}

func TestResolveHistoryRange_StopsForCancelledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fixture := setupTagRangeRepo(t)
	service, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	if service == nil {
		panic("getService returned nil without an error")
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = service.resolveHistoryRange(ctx, HistoryRange{From: "tag:v1.0"})

	g.Expect(errors.Is(err, context.Canceled)).To(BeTrue())
}

type tagRangeFixture struct {
	dir        string
	initial    string
	main       string
	feature    string
	featureTip string
	merged     string
	detached   string
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
	featureTip := writeCommit("feature.go", "package feature\n// D\n", "D", "2025-04-01T00:00:00Z")

	run("git", "switch", "main")
	run("git", "merge", "--no-ff", "feature", "-m", "M")
	merged := run("git", "rev-parse", "HEAD")
	run("git", "tag", "-a", "v2.0", "-m", "release v2")
	run("git", "tag", "previous-main", mainCommit)
	run("git", "tag", "-a", "previous-feature", featureTip, "-m", "feature checkpoint")
	run("git", "tag", "latest-lightweight", merged)
	blob := run("git", "hash-object", "-w", "shared.go")
	run("git", "tag", "-a", "blob-tag", blob, "-m", "non-commit tag")

	run("git", "switch", "--orphan", "detached-line")

	detached := writeCommit("detached.go", "package detached\n", "U", "2025-05-01T00:00:00Z")

	run("git", "tag", "detached")
	run("git", "switch", "main")

	return tagRangeFixture{
		dir:        dir,
		initial:    initial,
		main:       mainCommit,
		feature:    feature,
		featureTip: featureTip,
		merged:     merged,
		detached:   detached,
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

	commits, err := s.commitIterator(t.Context(), HistoryRange{From: "v1.0", Until: "v2.0"})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string

	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())

		hashes = append(hashes, commit.Hash.String())
	}

	g.Expect(hashes).To(ContainElements(fixture.main, fixture.feature, fixture.merged))
	g.Expect(hashes).NotTo(ContainElement(fixture.initial))
}

func TestHistoryRange_AcceptsPredecessorBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)

	commits, err := s.commitIterator(context.Background(), HistoryRange{
		From:  "^HEAD",
		Until: "HEAD",
	})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string
	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())
		hashes = append(hashes, commit.Hash.String())
	}

	g.Expect(hashes).To(ContainElements(fixture.feature, fixture.featureTip, fixture.merged))
	g.Expect(hashes).NotTo(ContainElements(fixture.initial, fixture.main))
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

	commits, err := s.commitIterator(t.Context(), HistoryRange{Until: "detached"})
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

	_, err = s.commitIterator(t.Context(), HistoryRange{From: "detached", Until: "v2.0"})
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

	_, err = s.commitIterator(t.Context(), HistoryRange{Until: "tag:missing"})
	g.Expect(err).To(MatchError(ContainSubstring(`tag "missing" not found`)))

	_, err = s.commitIterator(t.Context(), HistoryRange{Until: "tag:blob-tag"})
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

	commits, err := s.commitIterator(t.Context(), HistoryRange{
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

func TestHistoryRange_RelativeBoundsShareTimeNow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)
	timeNow := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)

	resolved, err := s.resolveHistoryRange(context.Background(), HistoryRange{
		From:    "ago(14d)",
		Until:   "ago(7d)",
		TimeNow: timeNow,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.from).To(Equal(timeNow.Add(-14 * 24 * time.Hour)))
	g.Expect(resolved.until).To(Equal(timeNow.Add(-7 * 24 * time.Hour)))
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

	_, err = s.commitIterator(t.Context(), HistoryRange{From: "2025-03-01", Until: "2025-01-01"})
	g.Expect(err).To(MatchError(ContainSubstring("--from must be before or equal to --until")))
}
