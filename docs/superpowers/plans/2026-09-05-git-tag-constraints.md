# Git Tag Constraints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add graph-correct `--from-tag` and `--until-tag` constraints to every visualization and preset command.

**Architecture:** Model all Git history selection in a provider-level `HistoryRange`. Resolve tags to commits inside the Git provider, traverse from the upper tag or `HEAD`, exclude the lower tag and all its ancestors, then apply existing inclusive author-date bounds. Reuse the resulting iterator for commit totals, timeline history, Git metric prewarming, and authorship history.

**Tech Stack:** Go 1.26.1, Kong, go-git v5, Gomega, Task

---

## File Structure

- Create `internal/provider/git/history_range.go`: public range model, tag peeling, ancestry validation, and range-aware commit iteration.
- Create `internal/provider/git/history_range_test.go`: graph-range fixtures and focused resolution/selection tests.
- Modify `internal/provider/git/commit.go`: route totals, history collection, and prewarming through `HistoryRange`.
- Modify `internal/provider/git/commit_test.go`: verify totals, timeline commits, progress, and prewarm consistency for tag ranges.
- Modify `internal/provider/git/author_history.go`: route authorship aggregation through the shared range iterator.
- Modify `internal/provider/git/author_history_test.go`: verify tag-constrained authorship.
- Modify `internal/stages/common.go`: carry one `git.HistoryRange` through shared pipeline state.
- Modify `internal/stages/git_history.go`: use the unified range for totals and history/prewarm.
- Modify `internal/stages/author_history.go`: use the unified range for totals and authorship.
- Modify `internal/stages/git_history_test.go` and `internal/stages/author_history_test.go`: verify stage propagation.
- Rename `cmd/codeviz/date_range.go` to `cmd/codeviz/history_range.go`: parse dates and validate/construct the unified range.
- Rename `cmd/codeviz/date_range_test.go` to `cmd/codeviz/history_range_test.go`: cover date behavior and date/tag conflicts.
- Modify `cmd/codeviz/{tree,radial,donut,bubble}tree_cmd.go`, `spiral_cmd.go`, and `scatter_cmd.go`: expose and forward tag flags.
- Modify `cmd/codeviz/render_cmd.go`: expose tag flags and copy them into generated preset commands.
- Modify `cmd/codeviz/main.go`, `main_test.go`, and `render_cmd_test.go`: update cross-layer flags and CLI parsing coverage.
- Modify `docs/content/docs/usage.md`: document history-range semantics and examples.

### Task 1: Add graph-aware history range resolution

**Files:**
- Create: `internal/provider/git/history_range.go`
- Create: `internal/provider/git/history_range_test.go`
- Test helper: `internal/provider/git/metrics_test.go:23-88`

- [ ] **Step 1: Write a reusable graph fixture**

Add a package-local helper to `history_range_test.go` that creates this graph:

```text
A (tag: v1.0) -- B -------- M (tag: v2.0)
                \          /
                 C -- D ---

U (tag: detached)
```

Use explicit commit dates and return the hashes needed by assertions:

```go
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
	dir := t.TempDir()

	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // fixed test commands
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
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

	writeCommit := func(branchFile, contents, message, date string) string {
		t.Helper()
		g.Expect(os.WriteFile(filepath.Join(dir, branchFile), []byte(contents), 0o600)).To(Succeed())
		run("git", "add", branchFile)
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
	run("git", "rm", "-rf", ".")
	detached := writeCommit("detached.go", "package detached\n", "U", "2025-05-01T00:00:00Z")
	run("git", "tag", "detached")
	run("git", "switch", "main")

	return tagRangeFixture{
		dir: dir, initial: initial, main: mainCommit,
		feature: feature, merged: merged, detached: detached,
	}
}
```

Declare `g := NewGomegaWithT(t)` at the start of the helper and import `os`,
`os/exec`, `path/filepath`, and `strings`.

- [ ] **Step 2: Write failing range-selection tests**

Add focused tests before implementation:

```go
func TestHistoryRange_FromTagIsExclusiveAndUntilTagIsInclusive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())

	commits, err := s.commitIterator(HistoryRange{FromTag: "v1.0", UntilTag: "v2.0"})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string
	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())
		hashes = append(hashes, commit.Hash.String())
	}

	g.Expect(hashes).To(ContainElements(fixture.main, fixture.feature, fixture.merged))
	g.Expect(hashes).NotTo(ContainElement(fixture.initial))
}

func TestHistoryRange_UntilTagCanBeOutsideHeadAncestry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	commits, err := s.commitIterator(HistoryRange{UntilTag: "detached"})
	g.Expect(err).NotTo(HaveOccurred())

	var hashes []string
	for commit, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())
		hashes = append(hashes, commit.Hash.String())
	}
	g.Expect(hashes).To(Equal([]string{fixture.detached}))
}

func TestHistoryRange_RejectsFromTagOutsideTipAncestry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	_, err = s.commitIterator(HistoryRange{FromTag: "detached", UntilTag: "v2.0"})
	g.Expect(err).To(MatchError(ContainSubstring(`tag "detached" is not an ancestor of tag "v2.0"`)))
}

func TestHistoryRange_ReportsMissingAndNonCommitTags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = s.commitIterator(HistoryRange{UntilTag: "missing"})
	g.Expect(err).To(MatchError(ContainSubstring(`tag "missing" not found`)))

	_, err = s.commitIterator(HistoryRange{UntilTag: "blob-tag"})
	g.Expect(err).To(MatchError(ContainSubstring(`tag "blob-tag" does not reference a commit`)))
}
```

- [ ] **Step 3: Run the focused tests to verify failure**

Run:

```bash
go test ./internal/provider/git -run 'TestHistoryRange_' -count=1
```

Expected: compilation fails because `HistoryRange` and the range-aware
`commitIterator` do not exist.

- [ ] **Step 4: Implement the range model and resolver**

Create `history_range.go` with the following public model and focused private
helpers:

```go
package git

import (
	"errors"
	"iter"
	"strconv"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

type HistoryRange struct {
	From     time.Time
	Until    time.Time
	FromTag  string
	UntilTag string
}

type resolvedHistoryRange struct {
	tip      plumbing.Hash
	excluded map[plumbing.Hash]struct{}
	from     time.Time
	until    time.Time
}

func (s *repoService) resolveHistoryRange(r HistoryRange) (resolvedHistoryRange, error) {
	tip, tipLabel, err := s.resolveRangeTip(r.UntilTag)
	if err != nil {
		return resolvedHistoryRange{}, err
	}

	resolved := resolvedHistoryRange{tip: tip, from: r.From, until: r.Until}
	if r.FromTag == "" {
		return resolved, nil
	}

	from, err := s.resolveTagCommit(r.FromTag)
	if err != nil {
		return resolvedHistoryRange{}, err
	}

	excluded, err := s.reachableHashes(from)
	if err != nil {
		return resolvedHistoryRange{}, eris.Wrapf(err, "failed to inspect tag %q history", r.FromTag)
	}

	reachableFromTip, err := s.reachableHashes(tip)
	if err != nil {
		return resolvedHistoryRange{}, eris.Wrap(err, "failed to inspect effective tip history")
	}
	if _, ok := reachableFromTip[from]; !ok {
		return resolvedHistoryRange{}, eris.Errorf(
			"tag %q is not an ancestor of %s", r.FromTag, tipLabel,
		)
	}

	resolved.excluded = excluded
	return resolved, nil
}

func (s *repoService) resolveRangeTip(untilTag string) (plumbing.Hash, string, error) {
	if untilTag != "" {
		hash, err := s.resolveTagCommit(untilTag)
		return hash, "tag " + strconv.Quote(untilTag), err
	}
	head, err := s.repo.Head()
	if err != nil {
		return plumbing.ZeroHash, "", eris.Wrap(err, "failed to get HEAD")
	}
	return head.Hash(), "HEAD", nil
}

func (s *repoService) resolveTagCommit(name string) (plumbing.Hash, error) {
	ref, err := s.repo.Reference(plumbing.NewTagReferenceName(name), true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return plumbing.ZeroHash, eris.Errorf("tag %q not found", name)
		}
		return plumbing.ZeroHash, eris.Wrapf(err, "failed to resolve tag %q", name)
	}

	hash := ref.Hash()
	seen := map[plumbing.Hash]struct{}{}
	for {
		if _, err := s.repo.CommitObject(hash); err == nil {
			return hash, nil
		}
		if _, duplicate := seen[hash]; duplicate {
			return plumbing.ZeroHash, eris.Errorf("tag %q contains a tag cycle", name)
		}
		seen[hash] = struct{}{}

		tag, err := s.repo.TagObject(hash)
		if err != nil {
			return plumbing.ZeroHash, eris.Errorf("tag %q does not reference a commit", name)
		}
		hash = tag.Target
	}
}

func (s *repoService) reachableHashes(from plumbing.Hash) (map[plumbing.Hash]struct{}, error) {
	iter, err := s.repo.Log(&gogit.LogOptions{From: from})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	result := map[plumbing.Hash]struct{}{}
	err = iter.ForEach(func(commit *object.Commit) error {
		result[commit.Hash] = struct{}{}
		return nil
	})
	return result, err
}
```

Keep tag-resolution errors stable enough for the tests.

- [ ] **Step 5: Move commit iteration onto the resolved range**

Move `commitSequence`, `yieldCommits`, `nextCommit`, `filterCommitsInRange`, and
`commitInRange` from `commit.go` into `history_range.go`. Replace
`commitIterator(from, until)` with:

```go
func (s *repoService) commitIterator(r HistoryRange) (iter.Seq2[*object.Commit, error], error) {
	resolved, err := s.resolveHistoryRange(r)
	if err != nil {
		return nil, err
	}

	commitIter, err := s.repo.Log(&gogit.LogOptions{From: resolved.tip})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}

	return filterCommitsInRange(commitSequence(commitIter), resolved), nil
}

func filterCommitsInRange(
	commits iter.Seq2[*object.Commit, error],
	r resolvedHistoryRange,
) iter.Seq2[*object.Commit, error] {
	return func(yield func(*object.Commit, error) bool) {
		for commit, iterationErr := range commits {
			if iterationErr != nil {
				yield(nil, iterationErr)
				return
			}
			if _, excluded := r.excluded[commit.Hash]; excluded {
				continue
			}
			if commitInDateRange(commit, r.from, r.until) && !yield(commit, nil) {
				return
			}
		}
	}
}
```

Rename `commitInRange` to `commitInDateRange`. Keep date bounds inclusive.

- [ ] **Step 6: Run focused tests**

Run:

```bash
go test ./internal/provider/git -run 'TestHistoryRange_' -count=1
```

Expected: all `TestHistoryRange_...` tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/provider/git/history_range.go internal/provider/git/history_range_test.go
git commit -m "feat(git): resolve graph-aware history ranges" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Apply the range to totals, timeline history, and prewarming

**Files:**
- Modify: `internal/provider/git/commit.go:43-195,348-378`
- Modify: `internal/provider/git/commit_test.go:84-191`

- [ ] **Step 1: Write failing consistency tests**

Add tests that exercise the public range-aware operations:

```go
func TestHistoryRange_TotalHistoryAndPrewarmSelectSameCommits(t *testing.T) {
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	r := HistoryRange{FromTag: "v1.0", UntilTag: "v2.0"}
	tracked := map[string]bool{"main.go": true, "feature.go": true}

	total, err := CommitTotalInHistoryRange(fixture.dir, r)
	g.Expect(err).NotTo(HaveOccurred())

	resetService()
	commits, err := BulkCommitHistoryAndPrewarmInHistoryRange(
		fixture.dir, tracked, []metric.Name{CommitCount}, r, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(total).To(Equal(int64(4)))
	g.Expect(commits).To(HaveLen(4))

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s.cachedCommitData("main.go").count).To(Equal(int64(1)))
	g.Expect(s.cachedCommitData("feature.go").count).To(Equal(int64(2)))
}
```

Add a second test with `HistoryRange{FromTag: "v1.0", Until: <date after C
before D>}` to prove graph filtering happens before the inclusive author-date
filter.

- [ ] **Step 2: Run the tests to verify failure**

Run:

```bash
go test ./internal/provider/git -run 'TestHistoryRange_Total|TestHistoryRange_Mixed' -count=1
```

Expected: compilation fails because the `InHistoryRange` entry points do not
exist.

- [ ] **Step 3: Add range-aware provider entry points**

Keep all existing public functions for compatibility. Make them thin wrappers:

```go
func CommitTotal(repoPath string) (int64, error) {
	return CommitTotalInHistoryRange(repoPath, HistoryRange{})
}

func CommitTotalInRange(repoPath string, from, until time.Time) (int64, error) {
	return CommitTotalInHistoryRange(repoPath, HistoryRange{From: from, Until: until})
}

func CommitTotalInHistoryRange(repoPath string, r HistoryRange) (int64, error) {
	s, err := getService(repoPath)
	if err != nil {
		return 0, eris.Wrap(err, "failed to open git repository")
	}
	return s.commitTotalInHistoryRange(r)
}
```

Apply the same pattern to:

```go
func BulkCommitHistoryInHistoryRange(
	repoPath string,
	tracked map[string]bool,
	r HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error)

func BulkCommitHistoryAndPrewarmInHistoryRange(
	repoPath string,
	tracked map[string]bool,
	requested []metric.Name,
	r HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error)
```

Rename private methods to `commitTotalInHistoryRange`,
`bulkCommitHistoryAndPrewarmInHistoryRange`, and
`walkTrackedHistoryInHistoryRange`. Each receives one `HistoryRange` and calls
`s.commitIterator(r)`. Preserve locking, callback timing, atomic cache
publication, path normalization, and existing wrappers.

- [ ] **Step 4: Run provider tests**

Run:

```bash
go test ./internal/provider/git -run 'Test(HistoryRange_|CommitTotal|BulkCommitHistory)' -count=1
```

Expected: all selected tests pass, including existing unbounded and date-only
regressions.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git/commit.go internal/provider/git/commit_test.go
git commit -m "feat(git): apply history ranges to commits and metrics" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Apply the shared range to authorship and pipeline stages

**Files:**
- Modify: `internal/provider/git/author_history.go:71-160`
- Modify: `internal/provider/git/author_history_test.go`
- Modify: `internal/stages/common.go:17-30`
- Modify: `internal/stages/git_history.go:31-58`
- Modify: `internal/stages/author_history.go:15-35`
- Modify: `internal/stages/git_history_test.go`
- Modify: `internal/stages/author_history_test.go`

- [ ] **Step 1: Write a failing authorship range test**

Add:

```go
func TestBulkAuthorHistoryInHistoryRange_UsesTagSelection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	result, err := BulkAuthorHistoryInHistoryRange(
		fixture.dir,
		map[string]bool{"main.go": true, "feature.go": true},
		false,
		HistoryRange{FromTag: "v1.0", UntilTag: "v2.0"},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.ByFile).To(HaveKey("main.go"))
	g.Expect(result.ByFile).To(HaveKey("feature.go"))
	g.Expect(result.HeadDate.IsZero()).To(BeFalse())
}
```

- [ ] **Step 2: Run the test to verify failure**

Run:

```bash
go test ./internal/provider/git -run TestBulkAuthorHistoryInHistoryRange -count=1
```

Expected: compilation fails because the new function is undefined.

- [ ] **Step 3: Route authorship through the shared iterator**

Add `BulkAuthorHistoryInHistoryRange` and retain both existing wrappers:

```go
func BulkAuthorHistoryInRange(
	repoPath string,
	filePaths map[string]bool,
	honorMailmap bool,
	from, until time.Time,
	onCommitProcessed func(),
) (AuthorHistoryResult, error) {
	return BulkAuthorHistoryInHistoryRange(
		repoPath, filePaths, honorMailmap,
		HistoryRange{From: from, Until: until},
		onCommitProcessed,
	)
}
```

Inside the new function, acquire `repoMu`, call `s.commitIterator(r)`, and
replace `iter.ForEach` plus inline date checks with:

```go
for c, iterationErr := range commits {
	if iterationErr != nil {
		return AuthorHistoryResult{}, eris.Wrap(iterationErr, "failed to iterate commits")
	}

	when := c.Author.When
	// existing accumulation body remains unchanged
}
```

Do not open a second `repo.Log`; the shared iterator is the source of truth.

- [ ] **Step 4: Replace stage date fields with one range**

Change `stages.Flags`:

```go
type Flags struct {
	Quiet        bool
	Verbose      bool
	Debug        bool
	ExportConfig string
	ExportData   string
	Config       *config.Config
	HistoryRange git.HistoryRange
}
```

Update `LoadGitHistory`:

```go
r := c.Flags.HistoryRange
total, err := git.CommitTotalInHistoryRange(repoRoot, r)
// ...
commits, err := git.BulkCommitHistoryAndPrewarmInHistoryRange(
	repoRoot, tracked, c.Requested.BaseMetrics, r, onCommit,
)
```

Update `LoadAuthorHistory` identically, calling
`BulkAuthorHistoryInHistoryRange(repoRoot, tracked, false, r, onCommit)`.

- [ ] **Step 5: Add stage propagation assertions**

Extend the existing temporary repositories with `v1.0` on the first commit.
Set:

```go
state.Flags.HistoryRange = git.HistoryRange{FromTag: "v1.0"}
```

Assert `LoadGitHistory` excludes the first commit and prewarmed
`git.CommitCount` reflects only later commits. Add the equivalent
`LoadAuthorHistory` assertion that the first commit's author contribution is
absent.

- [ ] **Step 6: Run focused provider and stage tests**

Run:

```bash
go test ./internal/provider/git ./internal/stages \
  -run 'Test(BulkAuthorHistoryInHistoryRange|LoadGitHistory|LoadAuthorHistory)' \
  -count=1
```

Expected: all selected tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/provider/git/author_history.go \
  internal/provider/git/author_history_test.go \
  internal/stages/common.go internal/stages/git_history.go \
  internal/stages/author_history.go internal/stages/git_history_test.go \
  internal/stages/author_history_test.go
git commit -m "feat(stages): propagate unified git history ranges" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Expose tag constraints on every command

**Files:**
- Rename: `cmd/codeviz/date_range.go` to `cmd/codeviz/history_range.go`
- Rename: `cmd/codeviz/date_range_test.go` to `cmd/codeviz/history_range_test.go`
- Modify: `cmd/codeviz/main.go:39-69`
- Modify: `cmd/codeviz/main_test.go:90-111`
- Modify: `cmd/codeviz/treemap_cmd.go:14-49,87`
- Modify: `cmd/codeviz/radialtree_cmd.go:14-56,116`
- Modify: `cmd/codeviz/donuttree_cmd.go:14-48,87`
- Modify: `cmd/codeviz/bubbletree_cmd.go:14-51,90`
- Modify: `cmd/codeviz/spiral_cmd.go:15-52,138`
- Modify: `cmd/codeviz/scatter_cmd.go:16-56,178`
- Modify: `cmd/codeviz/render_cmd.go:22-34,98-121,197-269`
- Modify: `cmd/codeviz/render_cmd_test.go`

- [ ] **Step 1: Write failing validation tests**

Rename the date test file and add:

```go
func TestParseHistoryRange_AllowsMixedOppositeBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r, err := parseHistoryRange("2025-01-01", "", "", "v2.0")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(r.From).NotTo(BeZero())
	g.Expect(r.UntilTag).To(Equal("v2.0"))
}

func TestParseHistoryRange_RejectsSameSideDateAndTag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := parseHistoryRange("2025-01-01", "", "v1.0", "")
	g.Expect(err).To(MatchError("--from and --from-tag are mutually exclusive"))

	_, err = parseHistoryRange("", "2025-12-31", "", "v2.0")
	g.Expect(err).To(MatchError("--until and --until-tag are mutually exclusive"))
}
```

Extend `TestCLI_ParsesDateRangeFlags` to parse `--from-tag v1.0 --until-tag
v2.0` in a separate table row and assert the command fields. Add a render test
that verifies generated preset commands retain both tag fields.

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./cmd/codeviz -run 'Test(ParseHistoryRange|CLI_Parses.*Range|RenderCmd.*Tag)' -count=1
```

Expected: compilation or assertions fail because tag fields and
`parseHistoryRange` do not exist.

- [ ] **Step 3: Generalize range parsing**

Use `git mv` for the file rename. Preserve `parseDate` and `parseDateRange`, then
add:

```go
func parseHistoryRange(from, until, fromTag, untilTag string) (git.HistoryRange, error) {
	if from != "" && fromTag != "" {
		return git.HistoryRange{}, eris.New("--from and --from-tag are mutually exclusive")
	}
	if until != "" && untilTag != "" {
		return git.HistoryRange{}, eris.New("--until and --until-tag are mutually exclusive")
	}

	fromTime, untilTime, err := parseDateRange(from, until)
	if err != nil {
		return git.HistoryRange{}, err
	}
	return git.HistoryRange{
		From: fromTime, Until: untilTime,
		FromTag: fromTag, UntilTag: untilTag,
	}, nil
}

func stagesFlagsForCommand(
	flags *Flags,
	from, until, fromTag, untilTag string,
) (*stages.Flags, error) {
	parsedFlags := toStagesFlags(flags)
	r, err := parseHistoryRange(from, until, fromTag, untilTag)
	if err != nil {
		return nil, err
	}
	parsedFlags.HistoryRange = r
	return parsedFlags, nil
}
```

- [ ] **Step 4: Add and forward flags on direct commands**

In each of the six visualization command structs, add:

```go
FromTag  string `help:"Filter git activity strictly after this tag." name:"from-tag" optional:""`
UntilTag string `help:"Filter git activity through and including this tag." name:"until-tag" optional:""`
```

Change every `Validate` method to:

```go
_, err := parseHistoryRange(c.From, c.Until, c.FromTag, c.UntilTag)
return err
```

Change every `Run` call to:

```go
stagesFlags, err := stagesFlagsForCommand(
	flags, c.From, c.Until, c.FromTag, c.UntilTag,
)
```

- [ ] **Step 5: Add and forward flags on render presets**

Add the same fields to `RenderCmd`, validate through `parseHistoryRange`, and
copy both values in all five preset constructors:

```go
From:       r.From,
Until:      r.Until,
FromTag:    r.FromTag,
UntilTag:   r.UntilTag,
```

Do not add configuration fields.

- [ ] **Step 6: Simplify cross-layer flags**

Remove `From` and `Until` from `cmd/codeviz.Flags` and remove the `time` import
from `main.go`. `toStagesFlags` should copy only global output/config fields;
`stagesFlagsForCommand` assigns `HistoryRange`.

- [ ] **Step 7: Run all CLI tests**

Run:

```bash
go test ./cmd/codeviz -count=1
```

Expected: all `cmd/codeviz` tests pass.

- [ ] **Step 8: Commit**

```bash
git add cmd/codeviz
git commit -m "feat(cli): add git tag constraint flags" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Document tag ranges

**Files:**
- Modify: `docs/content/docs/usage.md:29-68`

- [ ] **Step 1: Add the history constraints reference**

Insert after the Commands section:

````markdown
## Git history constraints

Every visualization command and `render` accepts the following Git history
constraints:

| Flag | Includes |
| --- | --- |
| `--from YYYY-MM-DD` | Commits authored on or after the date |
| `--until YYYY-MM-DD` | Commits authored through the end of the date |
| `--from-tag TAG` | Commits strictly after the tag |
| `--until-tag TAG` | The tagged commit and its reachable history |

`--from` and `--from-tag` are mutually exclusive, as are `--until` and
`--until-tag`. Opposite date and tag bounds may be mixed:

```sh
codeviz tree-map . -o release.png -f commit-count \
  --from-tag v1.0 --until-tag v2.0
```

Tag ranges follow the full Git commit graph. The lower tag must be an ancestor
of the upper tag, or of `HEAD` when `--until-tag` is omitted. An upper tag may
be outside the current `HEAD` history.

These options constrain Git-derived metrics and timeline history. They do not
remove unchanged files from the current checkout.
````

- [ ] **Step 2: Review documentation rendering**

Run:

```bash
git diff --check
```

Expected: exit status 0 and no whitespace errors.

- [ ] **Step 3: Commit**

```bash
git add docs/content/docs/usage.md
git commit -m "docs: explain git tag history constraints" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 6: Verify the complete feature

**Files:**
- Review: all files changed in Tasks 1-5

- [ ] **Step 1: Format**

Run:

```bash
task fmt
```

Expected: command succeeds. Review and include only formatting changes caused by
this feature.

- [ ] **Step 2: Run focused tests together**

Run:

```bash
go test ./internal/provider/git ./internal/stages ./cmd/codeviz -count=1
```

Expected: all packages pass.

- [ ] **Step 3: Run repository CI through the required delegated workflow**

Dispatch an Explore-equivalent task runner for:

```bash
task ci
```

Require it to report only the exit status, failing test/linter identities,
offending `file:line` messages, or a one-line success note. Expected: build,
tests, and lint all pass.

- [ ] **Step 4: Inspect the final diff**

Run:

```bash
git status --short
git diff --check
git diff --stat main...HEAD
```

Expected: no uncommitted files, no whitespace errors, and only the planned
provider, stage, CLI, test, and documentation changes.

- [ ] **Step 5: Commit formatter-only changes if needed**

If `task fmt` changed tracked files after the prior commits:

```bash
git add internal/provider/git internal/stages cmd/codeviz docs/content/docs/usage.md
git commit -m "style: format git tag constraint changes" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

If formatting produced no diff, do not create an empty commit.
