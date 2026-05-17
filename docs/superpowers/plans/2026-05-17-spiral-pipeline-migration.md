# Spiral Pipeline Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the spiral visualization to the pipeline scaffold proven by treemap and bubbletree, and extract three reusable git-history stages (`LoadGitHistory`, `GroupGitHistoryByFile`, `ExtractFileHistory`) into `internal/stages` so future time-aware visualizations don't re-implement the commit walk.

**Architecture:** Add a richer `git.Commit` type and `git.BulkCommitHistory` next to today's `BulkFileHistory`. Thread commit data through three composable stages writing `GitHistory`, `FileHistory`, and `FileTimeRange` onto `CommonState`. Move `cmd/codeviz/spiral_canvas.go`, `spiral_githistory.go`, and most of `spiral_cmd.go` into a new shape under `internal/spiral` (`render.go`, `inks.go`, `bucketing.go`, `aggregation.go`, `discsize.go`, `state.go`, `stages.go`). Rewrite `SpiralCmd.Run` as a `pipeline.Run` composition.

**Tech Stack:** Go 1.26.1, Kong, eris, Gomega, fogleman/gg, go-git. Toolchain via Taskfile (`task build`, `task test`, `task lint`, `task ci`).

**Reference spec:** [docs/superpowers/specs/2026-05-17-spiral-pipeline-migration-design.md](../specs/2026-05-17-spiral-pipeline-migration-design.md)

**Branch:** `feature/spiral-pipeline` (already created off `main` after PR #248 merged).

**Workflow reminder:** Run `task lint` and `task ci` via the `Explore` subagent — never inline — per the agent workflow rules in `.github/copilot-instructions.md`.

---

## File Structure

| Path                                                       | Responsibility                                                                                                                       |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/provider/git/commit.go` (new)                    | `Signature`, `Commit` types + `BulkCommitHistory(repo, tracked, onCommit)`.                                                          |
| `internal/provider/git/commit_test.go` (new)               | Unit tests for `BulkCommitHistory` against an in-tree `setupTestGitRepo`-style fixture.                                              |
| `internal/stages/common.go` (modify)                       | Add `GitHistory`, `FileHistory`, `FileTimeRange` fields to `CommonState`.                                                            |
| `internal/stages/git_history.go` (new)                     | `CommitRef`, `TimeRange` types + `LoadGitHistory`, `GroupGitHistoryByFile`, `ExtractFileHistory` stages.                             |
| `internal/stages/git_history_test.go` (new)                | Unit tests for the three stages using the same fixture style.                                                                        |
| `internal/stages/progress.go` (modify)                     | Add `BuildHistoryProgress(*Flags)` moved from `cmd/codeviz/progress.go`.                                                             |
| `internal/spiral/render.go` (new)                          | All spiral render code, moved from `cmd/codeviz/spiral_canvas.go`. Exports `RenderToCanvas`.                                         |
| `internal/spiral/inks.go` (new)                            | `Inks` struct + `BuildInks(...)` + `buildBucketInk` (spiral-local).                                                                  |
| `internal/spiral/bucketing.go` (new)                       | `assignFilesToBuckets`, `commitTimeRange` (folded from `[]CommitRef`).                                                               |
| `internal/spiral/aggregation.go` (new)                     | `aggregateBucketMetrics`, `aggregateBucket`, `aggregateColourMetric`, `sumNumericMetric`, `modeCategory`.                            |
| `internal/spiral/discsize.go` (new)                        | `applyDiscSizes` (renamed from `applySpiralDiscSizes`), `minDiscRadius`.                                                             |
| `internal/spiral/state.go` (new)                           | `State` struct + `Common()` + `IncludeBinary()` methods.                                                                             |
| `internal/spiral/stages.go` (new)                          | Spiral-specific stages.                                                                                                              |
| `internal/spiral/render_test.go` (new)                     | PNG/SVG/JPG decode + spiralBorderWidth ported from `cmd/codeviz/spiral_canvas_test.go`.                                              |
| `internal/spiral/inks_test.go` (new)                       | `BuildInks` numeric/categorical/no-metrics tests ported from `cmd/codeviz/spiral_canvas_test.go`.                                    |
| `cmd/codeviz/spiral_canvas.go` (delete)                    | Contents moved to `internal/spiral/render.go` + `inks.go`.                                                                           |
| `cmd/codeviz/spiral_canvas_test.go` (delete)               | Tests split between `internal/spiral/render_test.go` and `inks_test.go`.                                                             |
| `cmd/codeviz/spiral_githistory.go` (delete)                | Behaviour replaced by `stages.LoadGitHistory` + `stages.GroupGitHistoryByFile`.                                                      |
| `cmd/codeviz/progress.go` (modify)                         | Remove `buildHistoryProgress` and `startHistoryTicker`. Keep `startProgressTicker` only if still referenced; otherwise delete.       |
| `cmd/codeviz/spiral_cmd.go` (modify)                       | `Run()` becomes a `pipeline.Run` composition. Kong struct, `Validate`, `validateConfig`, `mergeConfigAndValidate`, `applyOverrides` unchanged. All other helpers deleted. |

---

## Task 1: Add `git.BulkCommitHistory` and rich `Commit` type

Lands the new commit-data API additively. No existing callers change yet; `BulkFileHistory` keeps working.

**Files:**
- Create: `internal/provider/git/commit.go`
- Create: `internal/provider/git/commit_test.go`

- [ ] **Step 1: Create `internal/provider/git/commit.go`**

```go
package git

import (
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// Signature mirrors go-git's object.Signature: an author or committer record
// captured at the moment a commit was made.
type Signature struct {
	Name  string
	Email string
	When  time.Time
}

// Commit is a single commit in the project history, carrying enough metadata
// for any downstream consumer (timeline, churn, authorship, message-mining).
// ChangedPaths is restricted to the tracked path set passed to BulkCommitHistory
// so the slice size stays bounded.
//
// Invariant: once BulkCommitHistory returns, no field of any returned Commit
// is mutated. Consumers may hold *Commit references (e.g. via CommitRef) for
// the lifetime of the slice.
type Commit struct {
	Hash         string
	Author       Signature
	Committer    Signature
	Message      string
	ParentHashes []string
	ChangedPaths []string // slash-separated, repo-relative
}

// BulkCommitHistory walks the commit graph once and returns one Commit per
// commit reachable from HEAD that touches at least one path in `tracked`.
// Commits that change no tracked path are omitted.
//
// onCommitProcessed is invoked after each commit is examined (including
// skipped ones), allowing callers to drive a progress meter.
func BulkCommitHistory(
	repoPath string,
	tracked map[string]bool,
	onCommitProcessed func(),
) ([]Commit, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	head, err := s.repo.Head()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD")
	}

	iter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}
	defer iter.Close()

	var commits []Commit

	err = iter.ForEach(func(c *object.Commit) error {
		changed := changedFilesInCommit(c, tracked)

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		if len(changed) == 0 {
			return nil
		}

		commits = append(commits, Commit{
			Hash:         c.Hash.String(),
			Author:       toSignature(c.Author),
			Committer:    toSignature(c.Committer),
			Message:      c.Message,
			ParentHashes: parentHashes(c),
			ChangedPaths: changed,
		})

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return commits, nil
}

func toSignature(s object.Signature) Signature {
	return Signature{Name: s.Name, Email: s.Email, When: s.When}
}

func parentHashes(c *object.Commit) []string {
	if c.NumParents() == 0 {
		return nil
	}

	hashes := make([]string, 0, c.NumParents())
	for _, h := range c.ParentHashes {
		hashes = append(hashes, h.String())
	}

	return hashes
}
```

- [ ] **Step 2: Create `internal/provider/git/commit_test.go`**

```go
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
```

- [ ] **Step 3: Run the new tests**

Run: `go test ./internal/provider/git/...`
Expected: all tests pass.

- [ ] **Step 4: Lint via Explore subagent**

Dispatch `Explore` to run `task lint` and report only failing linters / file:line / message. Fix any issues inline.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git/commit.go internal/provider/git/commit_test.go
git commit -m "feat(git): add BulkCommitHistory with rich Commit type"
```

---

## Task 2: Move `BuildHistoryProgress` to `internal/stages` and add `LoadGitHistory`

Lands the first of three stages, plus the progress helper it consumes. Spiral still uses its old code path; this task only adds new surface area.

**Files:**
- Modify: `internal/stages/common.go`
- Modify: `internal/stages/progress.go`
- Create: `internal/stages/git_history.go`
- Create: `internal/stages/git_history_test.go`
- Modify: `cmd/codeviz/progress.go` (remove `buildHistoryProgress` + `startHistoryTicker`)
- Modify: `cmd/codeviz/spiral_cmd.go` (switch its one call site to the new shared helper)

- [ ] **Step 1: Add new fields to `CommonState`**

In `internal/stages/common.go`, add imports `time` and `"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"` (alongside existing imports), and extend `CommonState`:

```go
// Populated by shared stages during the pipeline:
FilterRules []filter.Rule    // BuildFilterRules
Requested   []metric.Name    // viz-specific ResolveMetrics
Root        *model.Directory // ScanFilesystem
Width       int              // ResolveDimensions
Height      int              // ResolveDimensions
Canvas      *canvas.Canvas   // viz-specific Render

// Git history (populated by LoadGitHistory / GroupGitHistoryByFile / ExtractFileHistory).
// GitHistory is written once and not mutated afterward; consumers may hold
// *Commit references for the lifetime of CommonState.
GitHistory    []git.Commit
FileHistory   map[*model.File][]CommitRef
FileTimeRange map[*model.File]TimeRange
```

(`CommitRef` and `TimeRange` are defined in the new `git_history.go` in step 3.)

- [ ] **Step 2: Add `BuildHistoryProgress` to `internal/stages/progress.go`**

Append to `internal/stages/progress.go`:

```go
// BuildHistoryProgress creates a per-commit callback and (if applicable) starts a
// ticker goroutine that logs commit history loading progress every second.
// The caller must invoke the returned stop function when loading completes.
func BuildHistoryProgress(flags *Flags) (onCommit func(), stop func()) {
	if !flags.Verbose && !flags.Debug {
		return nil, func() {}
	}

	counter := &atomic.Int64{}
	stop = startHistoryTicker(counter)

	return func() { counter.Add(1) }, stop
}

func startHistoryTicker(counter *atomic.Int64) (stop func()) {
	return startProgressTicker(func() {
		slog.Debug("Loading history...", "commits", counter.Load())
	})
}
```

(`startProgressTicker` already exists in `progress.go`; no new helper needed.)

- [ ] **Step 3: Create `internal/stages/git_history.go`**

```go
package stages

import (
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// CommitRef points back into CommonState.GitHistory with the per-file
// when-touched timestamp. Storing a pointer avoids duplicating Author /
// Message / ParentHashes per file-commit pair.
type CommitRef struct {
	Commit *git.Commit
	When   time.Time
}

// TimeRange is the earliest and latest commit times observed for a file.
type TimeRange struct {
	Earliest time.Time
	Latest   time.Time
}

// LoadGitHistory walks the commit graph once and populates Common().GitHistory.
// It returns an error when no commits touch any tracked file — visualizations
// that depend on git history cannot proceed in that case.
func LoadGitHistory[S VizState](s S) error {
	c := s.Common()

	repoRoot, err := git.RepoRootFor(c.Root.Path)
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	tracked := buildTrackedPathSet(c.Root, repoRoot)

	onCommit, stop := BuildHistoryProgress(c.Flags)

	commits, err := git.BulkCommitHistory(repoRoot, tracked, onCommit)

	stop()

	if err != nil {
		return eris.Wrap(err, "failed to load commit history")
	}

	if len(commits) == 0 {
		return eris.New("no commit history found; commit-history-dependent visualizations require git commits")
	}

	c.GitHistory = commits

	return nil
}

// GroupGitHistoryByFile joins Common().GitHistory against Common().Root and
// writes Common().FileHistory: each file maps to the CommitRefs that touched it.
func GroupGitHistoryByFile[S VizState](s S) error {
	c := s.Common()

	repoRoot, err := git.RepoRootFor(c.Root.Path)
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	byPath := indexFilesByRepoRelativePath(c.Root, repoRoot)

	result := make(map[*model.File][]CommitRef)

	for i := range c.GitHistory {
		commit := &c.GitHistory[i]

		for _, path := range commit.ChangedPaths {
			file, ok := byPath[path]
			if !ok {
				continue
			}

			result[file] = append(result[file], CommitRef{
				Commit: commit,
				When:   commit.Author.When,
			})
		}
	}

	c.FileHistory = result

	return nil
}

// ExtractFileHistory folds Common().FileHistory into per-file earliest/latest
// timestamps and writes Common().FileTimeRange.
func ExtractFileHistory[S VizState](s S) error {
	c := s.Common()

	result := make(map[*model.File]TimeRange, len(c.FileHistory))

	for file, refs := range c.FileHistory {
		if len(refs) == 0 {
			continue
		}

		earliest := refs[0].When
		latest := refs[0].When

		for _, r := range refs[1:] {
			if r.When.Before(earliest) {
				earliest = r.When
			}

			if r.When.After(latest) {
				latest = r.When
			}
		}

		result[file] = TimeRange{Earliest: earliest, Latest: latest}
	}

	c.FileTimeRange = result

	return nil
}

// CommitTimeRange folds the per-file ranges in Common().FileTimeRange into a
// single global earliest/latest pair. Returns the zero TimeRange when the map
// is empty.
func CommitTimeRange(fileRanges map[*model.File]TimeRange) TimeRange {
	var (
		set      bool
		earliest time.Time
		latest   time.Time
	)

	for _, r := range fileRanges {
		if !set {
			earliest = r.Earliest
			latest = r.Latest
			set = true

			continue
		}

		if r.Earliest.Before(earliest) {
			earliest = r.Earliest
		}

		if r.Latest.After(latest) {
			latest = r.Latest
		}
	}

	return TimeRange{Earliest: earliest, Latest: latest}
}

func buildTrackedPathSet(root *model.Directory, repoRoot string) map[string]bool {
	tracked := make(map[string]bool)

	model.WalkFiles(root, func(f *model.File) {
		rel, err := filepath.Rel(repoRoot, f.Path)
		if err != nil {
			return
		}

		tracked[filepath.ToSlash(rel)] = true
	})

	return tracked
}

func indexFilesByRepoRelativePath(root *model.Directory, repoRoot string) map[string]*model.File {
	index := make(map[string]*model.File)

	model.WalkFiles(root, func(f *model.File) {
		rel, err := filepath.Rel(repoRoot, f.Path)
		if err != nil {
			return
		}

		index[filepath.ToSlash(rel)] = f
	})

	return index
}

var (
	_ pipeline.Stage[VizState] = LoadGitHistory[VizState]
	_ pipeline.Stage[VizState] = GroupGitHistoryByFile[VizState]
	_ pipeline.Stage[VizState] = ExtractFileHistory[VizState]
)
```

- [ ] **Step 4: Create `internal/stages/git_history_test.go`**

Use the same fixture style as `internal/provider/git/metrics_test.go::setupTestGitRepo`. Copy that helper here, renamed `setupHistoryRepo`, so the test is self-contained.

```go
package stages

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// setupHistoryRepo creates a temp git repo with three commits touching two files.
// Commit 1 (Alice, 2024-01-01): adds a.go and b.go.
// Commit 2 (Bob,   2025-06-15): modifies b.go.
// Commit 3 (Alice, default):   adds c.go.
func setupHistoryRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runAs := func(name, email string, args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email,
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	runAs("Alice", "alice@example.com", "git", "init")
	runAs("Alice", "alice@example.com", "git", "config", "user.name", "Alice")
	runAs("Alice", "alice@example.com", "git", "config", "user.email", "alice@example.com")

	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b\n"), 0o600)

	runAs("Alice", "alice@example.com", "git", "add", ".")
	runAs("Alice", "alice@example.com", "git", "commit", "-m", "initial", "--date=2024-01-01T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b\n// edit\n"), 0o600)
	runAs("Bob", "bob@example.com", "git", "add", "b.go")
	runAs("Bob", "bob@example.com", "git", "commit", "-m", "bob edit", "--date=2025-06-15T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "c.go"), []byte("package c\n"), 0o600)
	runAs("Alice", "alice@example.com", "git", "add", "c.go")
	runAs("Alice", "alice@example.com", "git", "commit", "-m", "add c")

	return dir
}

func buildHistoryState(dir string) *historyTestState {
	root := &model.Directory{Path: dir, Name: filepath.Base(dir)}

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		root.Files = append(root.Files, &model.File{
			Path: filepath.Join(dir, name),
			Name: name,
		})
	}

	return &historyTestState{
		CommonState: CommonState{
			TargetPath: dir,
			Root:       root,
			Flags:      &Flags{Config: &config.Config{}},
		},
	}
}

type historyTestState struct {
	CommonState
}

func (s *historyTestState) Common() *CommonState { return &s.CommonState }

func TestLoadGitHistory_PopulatesGitHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))

	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(state.GitHistory).NotTo(BeEmpty())

	for _, c := range state.GitHistory {
		g.Expect(c.Hash).NotTo(BeEmpty())
		g.Expect(c.Author.When.IsZero()).To(BeFalse())
		g.Expect(c.ChangedPaths).NotTo(BeEmpty())
	}
}

func TestGroupGitHistoryByFile_PointsBackIntoGitHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())

	g.Expect(state.FileHistory).NotTo(BeEmpty())

	for _, refs := range state.FileHistory {
		for _, ref := range refs {
			g.Expect(ref.Commit).NotTo(BeNil())
			g.Expect(ref.When.IsZero()).To(BeFalse())
		}
	}
}

func TestGroupGitHistoryByFile_BFileTouchedByBothAuthors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())

	var bFile *model.File

	for _, f := range state.Root.Files {
		if f.Name == "b.go" {
			bFile = f

			break
		}
	}

	g.Expect(bFile).NotTo(BeNil())
	refs := state.FileHistory[bFile]
	g.Expect(refs).To(HaveLen(2))

	authors := map[string]bool{}
	for _, r := range refs {
		authors[r.Commit.Author.Name] = true
	}

	g.Expect(authors).To(Equal(map[string]bool{"Alice": true, "Bob": true}))
}

func TestExtractFileHistory_ComputesMinMax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())
	g.Expect(ExtractFileHistory(state)).To(Succeed())

	var bFile *model.File

	for _, f := range state.Root.Files {
		if f.Name == "b.go" {
			bFile = f

			break
		}
	}

	tr, ok := state.FileTimeRange[bFile]
	g.Expect(ok).To(BeTrue())

	expectedEarliest := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedLatest := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	g.Expect(tr.Earliest.Equal(expectedEarliest)).To(BeTrue(), "got %v", tr.Earliest)
	g.Expect(tr.Latest.Equal(expectedLatest)).To(BeTrue(), "got %v", tr.Latest)
}

func TestCommitTimeRange_FoldsGlobalMinMax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())
	g.Expect(ExtractFileHistory(state)).To(Succeed())

	global := CommitTimeRange(state.FileTimeRange)

	g.Expect(global.Earliest.IsZero()).To(BeFalse())
	g.Expect(global.Latest.After(global.Earliest) || global.Latest.Equal(global.Earliest)).To(BeTrue())
}

func TestCommitTimeRange_EmptyMap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tr := CommitTimeRange(nil)
	g.Expect(tr.Earliest.IsZero()).To(BeTrue())
	g.Expect(tr.Latest.IsZero()).To(BeTrue())
}
```

- [ ] **Step 5: Remove `buildHistoryProgress` and `startHistoryTicker` from `cmd/codeviz/progress.go`**

After the move, that file no longer needs `sync/atomic` if no other helper uses it. Verify with `goimports` after edits. `startProgressTicker` stays only if any other file in `cmd/codeviz` calls it — check via `grep -rn startProgressTicker cmd/codeviz/`. If the only caller was `startHistoryTicker`, delete `startProgressTicker` too. If the file becomes empty after the deletions, `git rm` it.

- [ ] **Step 6: Update the one spiral call site to use the new helper**

In `cmd/codeviz/spiral_cmd.go`, replace the single line:

```go
histProg, stopHistTicker := buildHistoryProgress(flags)
```

with:

```go
histProg, stopHistTicker := stages.BuildHistoryProgress(toStagesFlags(flags))
```

(Spiral already imports `internal/stages`; no new import needed.)

- [ ] **Step 7: Run all tests**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 8: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix any reported issues.

- [ ] **Step 9: Commit**

```bash
git add internal/stages/common.go internal/stages/progress.go \
        internal/stages/git_history.go internal/stages/git_history_test.go \
        cmd/codeviz/progress.go cmd/codeviz/spiral_cmd.go
git commit -m "feat(stages): add git-history stages and progress helper"
```

---

## Task 3: Move spiral render and inks into `internal/spiral`

Relocate `cmd/codeviz/spiral_canvas.go` into `internal/spiral/render.go` + `inks.go`, exporting symbols at the package boundary.

**Files:**
- Create: `internal/spiral/render.go`
- Create: `internal/spiral/inks.go`
- Create: `internal/spiral/render_test.go`
- Create: `internal/spiral/inks_test.go`
- Modify: `cmd/codeviz/spiral_cmd.go` (one call site each for `buildSpiralInks` and `renderSpiralToCanvas`)
- Delete: `cmd/codeviz/spiral_canvas.go`
- Delete: `cmd/codeviz/spiral_canvas_test.go`

- [ ] **Step 1: Create `internal/spiral/inks.go`**

```go
package spiral

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

var (
	defaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	defaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	trackColour   = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	labelColour   = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bgColour      = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	trackWidth    = 1.0
	labelGap      = 4.0
	trackMinSteps = 500
)

// Inks holds the fill and border Ink instances for a spiral render pass.
type Inks struct {
	Fill   canvas.Ink
	Border canvas.Ink
}

// BuildInks creates fill and border inks from aggregated time-bucket data.
// When fillMetric or borderMetric is empty, the corresponding ink is fixed.
func BuildInks(
	buckets []TimeBucket,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Fill:   canvas.FixedInk(defaultFill),
		Border: canvas.FixedInk(defaultBorder),
	}

	if fillMetric != "" {
		inks.Fill = buildBucketInk(
			buckets, fillMetric, fillPaletteName,
			func(b *TimeBucket) float64 { return b.FillValue },
			func(b *TimeBucket) string { return b.FillLabel },
			defaultFill,
		)
	}

	if borderMetric != "" {
		inks.Border = buildBucketInk(
			buckets, borderMetric, borderPaletteName,
			func(b *TimeBucket) float64 { return b.BorderValue },
			func(b *TimeBucket) string { return b.BorderLabel },
			defaultBorder,
		)
	}

	return inks
}

// buildBucketInk creates an Ink from time-bucket-aggregated metric values.
// Unlike the per-file helpers in internal/inks, spiral operates on already-
// aggregated TimeBucket data, so accessors are required.
func buildBucketInk(
	buckets []TimeBucket,
	m metric.Name,
	palName palette.PaletteName,
	numericFn func(*TimeBucket) float64,
	categoryFn func(*TimeBucket) string,
	fallback color.RGBA,
) canvas.Ink {
	d, ok := provider.GetDescriptor(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		values := make([]float64, len(buckets))
		for i := range buckets {
			values[i] = numericFn(&buckets[i])
		}

		return canvas.NumericInk(m, values, pal)
	}

	seen := map[string]bool{}

	var categories []string

	for i := range buckets {
		cat := categoryFn(&buckets[i])
		if cat != "" && !seen[cat] {
			seen[cat] = true
			categories = append(categories, cat)
		}
	}

	return canvas.CategoricalInk(m, categories, pal)
}
```

- [ ] **Step 2: Create `internal/spiral/render.go`**

Move the entire contents of `cmd/codeviz/spiral_canvas.go` (everything except the `var (...)`/`const (...)` blocks and `Inks`/`BuildInks`/`buildBucketInk`, which are now in `inks.go`) into this new file with these changes:

- Package declaration becomes `package spiral`.
- Drop the imports `internal/spiral` and any others now unused after the inks split.
- Rename `renderSpiralToCanvas` to `RenderToCanvas` (exported).
- Replace every `shapeInks` / `spiralInks` reference with `Inks`. Replace every `inks.fill` / `inks.border` with `inks.Fill` / `inks.Border`.
- Replace every `spiral.` prefix qualifier with nothing (e.g. `spiral.SpiralLayout` → `SpiralLayout`).
- Drop the `spiral` prefix from local constants that are now package-scoped: `spiralBorderWidth` → `borderWidth` (function), and any locals like `spiralBgColour` → `bgColour`. The colour vars already became `bgColour` etc. in `inks.go`; remove the `spiral` prefix everywhere in this file too.
- Keep `spiralBorderWidth` named `spiralBorderWidth` if you prefer minimizing churn — both work. Pick one and apply consistently. The reference plan code below assumes the rename.

Sketch (only the parts that change shape — the rest is mechanical):

```go
package spiral

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

// RenderToCanvas builds a Canvas from a spiral layout and time buckets.
func RenderToCanvas(
	layout SpiralLayout,
	buckets []TimeBucket,
	width, height int,
	inks Inks,
) *canvas.Canvas {
	// ...body identical to today's renderSpiralToCanvas with inks.Fill / inks.Border ...
}

// spiralBorderWidth returns the border width for a disc of the given radius.
func spiralBorderWidth(radius float64) float64 {
	if radius < 8.0 {
		return 2.0
	}

	return 3.0
}
```

Verify the file compiles with:

```bash
go build ./internal/spiral/...
```

Expected: no errors. (cmd/codeviz still references the old unexported names at this point — that gets fixed in step 4.)

- [ ] **Step 3: Create `internal/spiral/inks_test.go`**

Port the `TestBuildSpiralInks_*` tests:

```go
package spiral_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func sampleTimeBuckets() []spiral.TimeBucket {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return []spiral.TimeBucket{
		{
			Start: t0, End: t0.Add(time.Hour),
			Files: []*model.File{
				makeFile("a.go", "go", 100),
				makeFile("b.go", "go", 200),
			},
			SizeValue: 300, FillValue: 300, FillLabel: "go",
		},
		{
			Start: t0.Add(time.Hour), End: t0.Add(2 * time.Hour),
			Files:     []*model.File{makeFile("c.py", "py", 50)},
			SizeValue: 50, FillValue: 50, FillLabel: "py",
		},
		{
			Start:     t0.Add(2 * time.Hour),
			End:       t0.Add(3 * time.Hour),
			Files:     []*model.File{},
			SizeValue: 0,
		},
	}
}

func TestBuildInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inks := spiral.BuildInks(sampleTimeBuckets(), filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inks := spiral.BuildInks(sampleTimeBuckets(), filesystem.FileType, palette.Categorization, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildInks_NoMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inks := spiral.BuildInks(sampleTimeBuckets(), "", "", "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
}
```

- [ ] **Step 4: Create `internal/spiral/render_test.go`**

Port the `TestRenderSpiralToCanvas_*` tests and the in-package `TestSpiralBorderWidth`. Because `spiralBorderWidth` is unexported, use a same-package test file for that one:

`internal/spiral/render_test.go` (external `spiral_test`):

```go
package spiral_test

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

func TestRenderToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 800, 600, spiral.Hourly, spiral.LabelNone)
	inks := spiral.BuildInks(buckets, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "spiral.png")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 400, 300, spiral.Hourly, spiral.LabelNone)
	inks := spiral.BuildInks(buckets, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "spiral.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := decoder.Token()
		if xmlErr != nil {
			break
		}

		if se, ok := tok.(xml.StartElement); ok {
			rootElement = se.Name.Local

			break
		}
	}

	g.Expect(rootElement).To(Equal("svg"))
}

func TestRenderToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 400, 300, spiral.Hourly, spiral.LabelNone)
	inks := spiral.BuildInks(buckets, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "spiral.jpg")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}
```

And `internal/spiral/render_internal_test.go` (same-package, for the unexported helper):

```go
package spiral

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSpiralBorderWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(spiralBorderWidth(7.9)).To(Equal(2.0))
	g.Expect(spiralBorderWidth(8.0)).To(Equal(3.0))
	g.Expect(spiralBorderWidth(10.0)).To(Equal(3.0))
}
```

- [ ] **Step 5: Update `cmd/codeviz/spiral_cmd.go` callers**

In `layoutAndRender`:

```go
inks := spiral.BuildInks(buckets, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
// ...
cv := spiral.RenderToCanvas(layout, buckets, width, height, inks)
```

Replace `inks.fill` with `inks.Fill` and `inks.border` with `inks.Border` in the legend assembly that follows. Remove the now-unused `buildSpiralInks` / `renderSpiralToCanvas` references and the `spiralInks` local struct that lived in `spiral_canvas.go`.

- [ ] **Step 6: Delete the old files**

```bash
git rm cmd/codeviz/spiral_canvas.go cmd/codeviz/spiral_canvas_test.go
```

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 8: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 9: Commit**

```bash
git add internal/spiral/render.go internal/spiral/inks.go \
        internal/spiral/render_test.go internal/spiral/inks_test.go \
        internal/spiral/render_internal_test.go \
        cmd/codeviz/spiral_cmd.go \
        cmd/codeviz/spiral_canvas.go cmd/codeviz/spiral_canvas_test.go
git commit -m "refactor(spiral): move render+inks into internal/spiral"
```

---

## Task 4: Move spiral helpers (bucketing, aggregation, discsize) into `internal/spiral`

Helpers currently in `cmd/codeviz/spiral_cmd.go` move into focused files under `internal/spiral`. They stay unexported and are called only from stages (which arrive in Task 5).

**Files:**
- Create: `internal/spiral/bucketing.go`
- Create: `internal/spiral/aggregation.go`
- Create: `internal/spiral/discsize.go`
- Modify: `cmd/codeviz/spiral_cmd.go` (call sites switch to package-internal helpers via new `internal/spiral` exports)

- [ ] **Step 1: Create `internal/spiral/bucketing.go`**

```go
package spiral

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AssignFilesToBuckets distributes file-history records into pre-built time
// buckets. For each (*model.File, []stages.CommitRef) pair, each commit's
// timestamp is placed into the bucket whose half-open [Start, End) interval
// contains it. Files may appear in multiple buckets when they have multiple
// commits across time.
func AssignFilesToBuckets(
	buckets []TimeBucket,
	fileHistory map[*model.File][]stages.CommitRef,
) {
	for file, refs := range fileHistory {
		for _, ref := range refs {
			i := bucketIndexFor(buckets, ref.When)
			if i < 0 {
				continue
			}

			buckets[i].Files = append(buckets[i].Files, file)
		}
	}
}

func bucketIndexFor(buckets []TimeBucket, t time.Time) int {
	for i := range buckets {
		if !t.Before(buckets[i].Start) && t.Before(buckets[i].End) {
			return i
		}
	}

	return -1
}
```

- [ ] **Step 2: Create `internal/spiral/aggregation.go`**

```go
package spiral

import (
	"maps"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// AggregateBucketMetrics fills in SizeValue, FillValue, FillLabel, BorderValue,
// and BorderLabel for every bucket based on the files assigned to it. When
// sizeMetric is empty, SizeValue defaults to len(b.Files).
func AggregateBucketMetrics(
	buckets []TimeBucket,
	sizeMetric, fillMetric, borderMetric metric.Name,
) {
	for i := range buckets {
		aggregateBucket(&buckets[i], sizeMetric, fillMetric, borderMetric)
	}
}

func aggregateBucket(
	b *TimeBucket,
	sizeMetric, fillMetric, borderMetric metric.Name,
) {
	if sizeMetric != "" {
		b.SizeValue = sumNumericMetric(b.Files, sizeMetric)
	} else {
		b.SizeValue = float64(len(b.Files))
	}

	aggregateColourMetric(b.Files, fillMetric, &b.FillValue, &b.FillLabel)
	aggregateColourMetric(b.Files, borderMetric, &b.BorderValue, &b.BorderLabel)
}

func aggregateColourMetric(files []*model.File, m metric.Name, numVal *float64, catLabel *string) {
	if m == "" {
		return
	}

	d, ok := provider.GetDescriptor(m)
	if !ok {
		return
	}

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		*numVal = sumNumericMetric(files, m)
	} else {
		*catLabel = modeCategory(files, m)
	}
}

func sumNumericMetric(files []*model.File, m metric.Name) float64 {
	var total float64

	for _, f := range files {
		if v, ok := f.Quantity(m); ok {
			total += float64(v)

			continue
		}

		if v, ok := f.Measure(m); ok {
			total += v
		}
	}

	return total
}

func modeCategory(files []*model.File, m metric.Name) string {
	counts := map[string]int{}

	for _, f := range files {
		if cat, ok := f.Classification(m); ok {
			counts[cat]++
		}
	}

	best := ""
	bestCount := 0

	for _, cat := range slices.Sorted(maps.Keys(counts)) {
		if counts[cat] > bestCount {
			best = cat
			bestCount = counts[cat]
		}
	}

	return best
}
```

- [ ] **Step 3: Create `internal/spiral/discsize.go`**

```go
package spiral

import "math"

// minDiscRadius is the minimum visible disc radius for active time buckets.
const minDiscRadius = 3.0

// ApplyDiscSizes sets disc radii on nodes proportional to their bucket
// SizeValue. Empty buckets get zero radius (not drawn). Active buckets are
// clamped between minDiscRadius and maxDisc.
func ApplyDiscSizes(nodes []SpiralNode, buckets []TimeBucket, maxDisc float64) {
	maxSize := 0.0

	for _, b := range buckets {
		if b.SizeValue > maxSize {
			maxSize = b.SizeValue
		}
	}

	for i := range nodes {
		if buckets[i].SizeValue == 0 && len(buckets[i].Files) == 0 {
			nodes[i].DiscRadius = 0

			continue
		}

		if maxSize == 0 {
			nodes[i].DiscRadius = minDiscRadius

			continue
		}

		ratio := buckets[i].SizeValue / maxSize
		scaled := nodes[i].DiscRadius * math.Sqrt(ratio)
		nodes[i].DiscRadius = max(minDiscRadius, min(scaled, maxDisc))
	}
}
```

- [ ] **Step 4: Update `cmd/codeviz/spiral_cmd.go` call sites**

Replace local helper references with `spiral.AssignFilesToBuckets`, `spiral.AggregateBucketMetrics`, `spiral.ApplyDiscSizes`. Today's `assignFilesToBuckets(buckets, records)` takes `[]commitRecord`; switch the call to the new shape, which uses the records reshaped through `FileHistory`. Since Task 5/6 rewires through stages, the cleanest temporary bridge is:

```go
// In buildTimeBuckets, after `records` is loaded — temporary scaffolding until Task 6:
buckets := spiral.BuildTimeBuckets(resolution, startTime, endTime)
if len(buckets) == 0 {
    return nil, eris.New("no time buckets created from commit time range")
}

fileHistory := make(map[*model.File][]stages.CommitRef, len(records))
for _, rec := range records {
    fileHistory[rec.File] = append(fileHistory[rec.File], stages.CommitRef{When: rec.Timestamp})
}

spiral.AssignFilesToBuckets(buckets, fileHistory)
```

And in `layoutAndRender`:

```go
layout := spiral.Layout(buckets, width, height, resolution, labels)
maxDisc := spiral.MaxDiscRadius(len(buckets), width, height, resolution)
spiral.ApplyDiscSizes(layout.Nodes, buckets, maxDisc)
```

And replace `c.aggregateBucketMetrics(buckets, cfg)` with:

```go
spiral.AggregateBucketMetrics(buckets, sizeMetric, fillMetric, borderMetric)
```

Delete the now-dead local helpers from `cmd/codeviz/spiral_cmd.go`: `applySpiralDiscSizes`, `aggregateBucketMetrics`, `aggregateBucket`, `aggregateColourMetric`, `sumNumericMetric`, `modeCategory`, `assignFilesToBuckets`, `commitTimeRange`. Inline `commitTimeRange` into the buckets builder with a small fold over `records`. The bridge code in this task is intentionally short-lived; Task 6 deletes it entirely.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass. Smoke spiral once:

```bash
task build && ./bin/codeviz render spiral . -o /tmp/spiral.png
```

Expected: exit 0, valid PNG, log ends with `Rendered spiral …`.

- [ ] **Step 6: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 7: Commit**

```bash
git add internal/spiral/bucketing.go internal/spiral/aggregation.go \
        internal/spiral/discsize.go cmd/codeviz/spiral_cmd.go
git commit -m "refactor(spiral): move bucketing/aggregation/discsize into internal/spiral"
```

---

## Task 5: Add spiral pipeline `State` and stages

Land the pipeline plumbing. Nothing in `cmd/codeviz` uses it yet; Task 6 wires it up.

**Files:**
- Create: `internal/spiral/state.go`
- Create: `internal/spiral/stages.go`

- [ ] **Step 1: Create `internal/spiral/state.go`**

```go
package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// State is the pipeline state for the spiral visualization.
type State struct {
	stages.CommonState

	Config             *config.Spiral
	IncludeBinaryFiles bool

	// Resolved during the pipeline:
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Resolution    Resolution
	Labels        LabelMode

	Buckets      []TimeBucket
	Inks         Inks
	Layout       SpiralLayout
	LegendConfig *canvas.LegendConfig
}

// Common exposes the embedded CommonState so shared stages can mutate it.
func (s *State) Common() *stages.CommonState { return &s.CommonState }

// IncludeBinary lets State satisfy stages.BinaryFilterToggler.
func (s *State) IncludeBinary() bool { return s.IncludeBinaryFiles }
```

- [ ] **Step 2: Create `internal/spiral/stages.go`**

```go
package spiral

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, border, resolution, and label settings
// from the spiral config and populates Common().Requested.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.Size = metric.Name(stages.PtrString(cfg.Size))
	s.FillMetric = cfg.Fill.MetricName()
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	s.Resolution = resolveResolution(cfg)
	s.Labels = resolveLabels(cfg)

	s.Common().Requested = collectRequestedMetrics(s.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveResolution(cfg *config.Spiral) Resolution {
	if r := stages.PtrString(cfg.Resolution); r == "hourly" {
		return Hourly
	}

	return Daily
}

func resolveLabels(cfg *config.Spiral) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelLaps
}

// collectRequestedMetrics merges size + fill + border into a deduplicated
// metric set. When size is empty (spiral defaults to commit count), only fill
// and border contribute.
func collectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	if size != "" {
		return stages.CollectRequestedMetrics(size, fill, border)
	}

	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return names
}

// BuildTimeBucketsStage builds time buckets from Common().FileTimeRange and
// distributes files into them from Common().FileHistory.
func BuildTimeBucketsStage(s *State) error {
	c := s.Common()

	tr := stages.CommitTimeRange(c.FileTimeRange)
	if tr.Earliest.IsZero() {
		return eris.New("no commit timestamps available to build time buckets")
	}

	buckets := BuildTimeBuckets(s.Resolution, tr.Earliest, tr.Latest)
	if len(buckets) == 0 {
		return eris.New("no time buckets created from commit time range")
	}

	AssignFilesToBuckets(buckets, c.FileHistory)

	s.Buckets = buckets

	return nil
}

// AggregateBucketMetricsStage fills in per-bucket aggregated metric values.
func AggregateBucketMetricsStage(s *State) error {
	AggregateBucketMetrics(s.Buckets, s.Size, s.FillMetric, s.BorderMetric)

	return nil
}

// BuildInksStage builds spiral inks and emits the Rendering image log line.
func BuildInksStage(s *State) error {
	c := s.Common()

	s.Inks = BuildInks(s.Buckets, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the inks.
func BuildLegendStage(s *State) error {
	pos, orient := legend.ResolveOptions(
		stages.PtrString(s.Config.Legend),
		stages.PtrString(s.Config.LegendOrientation),
	)

	s.LegendConfig = legend.Build(
		pos, orient,
		s.Inks.Fill, s.FillMetric,
		s.Inks.Border, s.BorderMetric,
		s.Size,
	)

	return nil
}

// LayoutStage runs the spiral layout algorithm and applies disc sizing.
func LayoutStage(s *State) error {
	c := s.Common()

	layout := Layout(s.Buckets, c.Width, c.Height, s.Resolution, s.Labels)
	maxDisc := MaxDiscRadius(len(s.Buckets), c.Width, c.Height, s.Resolution)

	ApplyDiscSizes(layout.Nodes, s.Buckets, maxDisc)

	s.Layout = layout

	return nil
}

// RenderStage renders the spiral to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(s.Layout, s.Buckets, c.Width, c.Height, s.Inks)

	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary line, matching today's `Rendered spiral …`.
func LogResult(s *State) error {
	c := s.Common()

	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered spiral",
		"files", files,
		"directories", dirs,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(s.Size),
		"fill_metric", string(s.FillMetric),
		"fill_palette", string(s.FillPalette),
		"border_metric", string(s.BorderMetric),
		"border_palette", string(s.BorderPalette),
	)

	return nil
}

var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = BuildTimeBucketsStage
	_ pipeline.Stage[*State] = AggregateBucketMetricsStage
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
```

Add the import `"github.com/theunrepentantgeek/code-visualizer/internal/config"` if `config.Spiral` and `config.MetricSpec` are referenced (they are — in `resolveResolution` etc.). If `canvas` is unused after the edit, drop it.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/spiral/...`
Expected: no errors.

- [ ] **Step 4: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 5: Commit**

```bash
git add internal/spiral/state.go internal/spiral/stages.go
git commit -m "feat(spiral): add pipeline State and stages"
```

---

## Task 6: Rewrite `SpiralCmd.Run` as a pipeline composition

Replace the open-coded `Run` with `pipeline.Run` and delete all the helpers it used to call. Delete `cmd/codeviz/spiral_githistory.go`.

**Files:**
- Modify: `cmd/codeviz/spiral_cmd.go`
- Delete: `cmd/codeviz/spiral_githistory.go`

- [ ] **Step 1: Replace the imports**

After the edit, `cmd/codeviz/spiral_cmd.go` only needs:

```go
import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)
```

(`log/slog`, `maps`, `math`, `slices`, `time`, `internal/export`, `internal/legend`, `internal/model`, `internal/palette`, `internal/scan` are no longer needed.)

- [ ] **Step 2: Replace `Run` with the pipeline composition**

```go
func (c *SpiralCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	state := &spiral.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filter,
		},
		Config:             flags.Config.Spiral,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run[*spiral.State](
		state,
		stages.ValidatePaths,
		stages.ExportConfig,
		stages.BuildFilterRules,
		spiral.ResolveMetrics,
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.FilterBinaryFiles,
		stages.ExportData,
		stages.LoadGitHistory,
		stages.GroupGitHistoryByFile,
		stages.ExtractFileHistory,
		stages.ResolveDimensions,
		spiral.BuildTimeBucketsStage,
		spiral.AggregateBucketMetricsStage,
		spiral.BuildInksStage,
		spiral.BuildLegendStage,
		spiral.LayoutStage,
		spiral.RenderStage,
		stages.WriteCanvas,
		spiral.LogResult,
	)

	return eris.Wrap(err, "spiral pipeline failed")
}
```

- [ ] **Step 3: Delete now-dead helpers from `cmd/codeviz/spiral_cmd.go`**

Delete:

- `scanAndRunProviders`
- `buildTimeBuckets`
- `layoutAndRender`
- `logRendered`
- `aggregateBucketMetrics`
- `aggregateBucket`
- `aggregateColourMetric`
- `sumNumericMetric`
- `modeCategory`
- `commitTimeRange`
- `assignFilesToBuckets`
- `applySpiralDiscSizes`
- `resolveResolution`
- `resolveLabels`
- `resolveFillMetric`
- `collectSpiralMetrics`
- `minDiscRadius` constant

(Anything still referenced by `Validate` / `validateConfig` / `mergeConfigAndValidate` / `applyOverrides` stays.)

- [ ] **Step 4: Delete `cmd/codeviz/spiral_githistory.go`**

```bash
git rm cmd/codeviz/spiral_githistory.go
```

The `commitRecord` type and `loadCommitHistory` function it contained are no longer referenced by anything in `cmd/codeviz`.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 6: Smoke test the binary**

Run:
```bash
task build
./bin/codeviz render spiral . -o /tmp/spiral.png
file /tmp/spiral.png
```

Expected: exit 0, output reports `PNG image data …`, log output ends with a `Rendered spiral …` line whose keys match the pre-migration output (`files`, `directories`, `width`, `height`, `size_metric`, `fill_metric`, `fill_palette`, `border_metric`, `border_palette`).

- [ ] **Step 7: Lint and CI via Explore subagent**

Dispatch `Explore` to run `task ci`. Fix any issues.

- [ ] **Step 8: Commit**

```bash
git add cmd/codeviz/spiral_cmd.go cmd/codeviz/spiral_githistory.go
git commit -m "refactor(spiral): replace Run with pipeline composition"
```

- [ ] **Step 9: Push and open the PR**

```bash
git push -u origin feature/spiral-pipeline
gh pr create --base main --title "Spiral pipeline migration" --body "$(cat <<'EOF'
Migrates the spiral visualization to the pipeline scaffold used by treemap and bubbletree, and extracts three reusable git-history stages into internal/stages.

## Changes

- New `internal/provider/git.Commit` + `BulkCommitHistory`: richer commit data than `BulkFileHistory`, additive (existing callers untouched).
- New `internal/stages` stages: `LoadGitHistory`, `GroupGitHistoryByFile`, `ExtractFileHistory`. New `CommonState` fields: `GitHistory`, `FileHistory`, `FileTimeRange`. `BuildHistoryProgress` moved from `cmd/codeviz/progress.go`.
- Spiral render + inks moved into `internal/spiral` (`render.go`, `inks.go`); `cmd/codeviz/spiral_canvas{,_test}.go` deleted.
- Spiral helpers (bucketing, aggregation, disc sizing) moved into focused files under `internal/spiral`.
- New `internal/spiral/state.go` + `stages.go`; `SpiralCmd.Run` rewritten as `pipeline.Run`.
- `cmd/codeviz/spiral_githistory.go` deleted (behaviour now in `stages.LoadGitHistory` + `GroupGitHistoryByFile`).

## Verification

- `task ci` (build + test + lint) green.
- Smoke test: `./bin/codeviz render spiral . -o /tmp/spiral.png` renders successfully; `Rendered spiral …` log line carries the same keys as before.

Plan: `docs/superpowers/plans/2026-05-17-spiral-pipeline-migration.md`
Spec: `docs/superpowers/specs/2026-05-17-spiral-pipeline-migration-design.md`
EOF
)"
```

---

## Self-Review

**Spec coverage:**

- Goal 1 (move spiral code into `internal/spiral`): Tasks 3–5 ✓
- Goal 2 (rewrite `Run` as `pipeline.Run`): Task 6 ✓
- Goal 3 (extract three reusable git-history stages): Tasks 1, 2 ✓
- Non-goal "preserve CLI/output/log identity": Step 6 of Task 6 explicitly diff-checks the log keys ✓
- Non-goal "no promotion of `buildBucketInk`": `buildBucketInk` stays in `internal/spiral/inks.go` (Task 3) ✓
- Non-goal "no render-test scope expansion": Task 3 step 4 ports the existing tests as-is ✓
- Non-goal "preserve `BulkFileHistory`": Task 1 leaves it intact; `BulkCommitHistory` is additive ✓
- Section 3.1.1 `Commit`/`Signature` shape: Task 1 ✓
- Section 3.1.2 `CommitRef`/`TimeRange` + three stages: Task 2 ✓
- Section 3.2 spiral package layout: Tasks 3–5 ✓
- Section 3.3 spiral `State` shape: Task 5 step 1 ✓
- Section 3.4 stage signatures: Task 5 step 2 ✓
- Section 3.5 pipeline composition order: Task 6 step 2 ✓
- Section 7 risks (memory, pointer invariant, decomposition): doc comments in Task 1 (`Commit` invariant) and Task 2 (`CommonState.GitHistory` invariant) ✓

**Placeholder scan:** no TBDs, no "add error handling", no "similar to Task N". Every code step shows actual code. The one place where the plan defers to grep-driven cleanup (Task 2 step 5, `startProgressTicker` retention) is explicitly scoped to a single helper and gives the verification command.

**Type consistency:**

- `git.Commit` / `git.Signature` introduced in Task 1 and used as `*git.Commit` in `stages.CommitRef.Commit` in Task 2 ✓
- `stages.CommitRef` / `stages.TimeRange` introduced in Task 2 and consumed by `spiral.AssignFilesToBuckets` (Task 4) and `spiral.BuildTimeBucketsStage` (Task 5) ✓
- `spiral.Inks` / `spiral.RenderToCanvas` / `spiral.BuildInks` introduced in Task 3 and consumed by `spiral.BuildInksStage` / `RenderStage` in Task 5 ✓
- `spiral.State` field names (`Size`, `FillMetric`, `FillPalette`, `BorderMetric`, `BorderPalette`, `Resolution`, `Labels`, `Buckets`, `Inks`, `Layout`, `LegendConfig`) used consistently across Task 5 step 2 and the `LogResult` summary line ✓
- `spiral.ApplyDiscSizes` (exported in Task 4) consistent with the call site in `LayoutStage` (Task 5) ✓
- `stages.LoadGitHistory` / `GroupGitHistoryByFile` / `ExtractFileHistory` (Task 2) consistent with the pipeline composition in Task 6 ✓
- `stages.CommitTimeRange` (Task 2) consistent with the call site in `BuildTimeBucketsStage` (Task 5) ✓

**Known soft spots:**

- Task 4 introduces a brief "bridge" where `cmd/codeviz/spiral_cmd.go` still calls the old `loadCommitHistory` + reshapes the result into `map[*model.File][]stages.CommitRef` to feed `spiral.AssignFilesToBuckets`. This keeps that intermediate commit green. Task 6 deletes the bridge along with `spiral_githistory.go`. If the implementer would rather collapse Tasks 4–6 into a single bigger step to skip the bridge, that is acceptable — the commit history at the end is what matters.
- Task 2 step 3 uses `stages.PtrString`. Verify this helper exists (it's the same one referenced by the bubbletree migration plan and is used elsewhere under `internal/stages`). If the codebase has renamed it, update the references.

---

## Execution Handoff

Plan complete and saved to [docs/superpowers/plans/2026-05-17-spiral-pipeline-migration.md](docs/superpowers/plans/2026-05-17-spiral-pipeline-migration.md).
