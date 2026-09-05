# Merge Commit Range Constraints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `--from` and `--until` accept tags, commit IDs, and dates while preserving full-graph range semantics and removing the tag-specific flags.

**Architecture:** Keep history bounds as raw strings until the Git provider can resolve them against a repository. Add a focused history-reference resolver for prefixes, precedence, commit abbreviations, and date formats, then feed its typed results into the existing graph iterator used by all Git consumers.

**Tech Stack:** Go 1.26.1, Kong, go-git v5, eris, Gomega, Task

---

## File Structure

- Create `internal/provider/git/history_reference.go` for reference parsing, repository-aware resolution, and date parsing.
- Create `internal/provider/git/history_reference_test.go` for prefixes, precedence, SHA resolution, date formats, and diagnostics.
- Modify `internal/provider/git/history_range.go` to store raw bounds and consume resolved references.
- Modify `internal/provider/git/history_range_test.go` and provider integration tests to exercise revision and mixed ranges through the unified fields.
- Replace `cmd/codeviz/date_range.go` with `cmd/codeviz/history_range.go`, which only transfers raw CLI values into provider options.
- Replace `cmd/codeviz/date_range_test.go` with `cmd/codeviz/history_range_test.go`.
- Modify all command structs and preset forwarding in `cmd/codeviz/*_cmd.go`.
- Modify CLI tests in `cmd/codeviz/main_test.go` and `cmd/codeviz/render_cmd_test.go`.
- Modify `docs/content/docs/usage.md` with the unified syntax.

### Task 1: Parse and Resolve History References

**Files:**
- Create: `internal/provider/git/history_reference.go`
- Create: `internal/provider/git/history_reference_test.go`

- [ ] **Step 1: Write failing tests for explicit prefixes and unprefixed precedence**

Create table-driven tests using `setupTagRangeRepo(t)` and add collision tags:

```go
func TestResolveHistoryReference_UsesExplicitPrefixes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	tag, err := s.resolveHistoryReference("tag:v1.0", lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tag.revision).To(Equal(plumbing.NewHash(fixture.initial)))

	sha, err := s.resolveHistoryReference("sha:"+fixture.main[:8], lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sha.revision).To(Equal(plumbing.NewHash(fixture.main)))

	date, err := s.resolveHistoryReference("date:20250905-1430Z", lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(date.timestamp).To(Equal(time.Date(2025, 9, 5, 14, 30, 0, 0, time.UTC)))
}

func TestResolveHistoryReference_UnprefixedTagWinsOverCommitAndDate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	runGit(t, fixture.dir, "tag", fixture.main[:8], fixture.initial)
	runGit(t, fixture.dir, "tag", "20250501", fixture.main)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	shortCollision, err := s.resolveHistoryReference(fixture.main[:8], lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(shortCollision.revision).To(Equal(plumbing.NewHash(fixture.initial)))

	dateCollision, err := s.resolveHistoryReference("20250501", lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(dateCollision.revision).To(Equal(plumbing.NewHash(fixture.main)))
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // fixed test executable
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	NewWithT(t).Expect(err).NotTo(HaveOccurred(), string(output))
	return strings.TrimSpace(string(output))
}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run:

```bash
go test ./internal/provider/git -run 'TestResolveHistoryReference' -count=1
```

Expected: compilation fails because `resolveHistoryReference`, `lowerBound`, and the resolved reference type do not exist.

- [ ] **Step 3: Implement reference classification and repository resolution**

Create:

```go
type historyBound int

const (
	lowerBound historyBound = iota
	upperBound
)

type resolvedHistoryReference struct {
	revision  plumbing.Hash
	timestamp time.Time
}

func (r resolvedHistoryReference) hasRevision() bool {
	return !r.revision.IsZero()
}

func (s *repoService) resolveHistoryReference(
	value string,
	bound historyBound,
) (resolvedHistoryReference, error) {
	if value == "" {
		return resolvedHistoryReference{}, nil
	}

	prefix, payload, explicit := splitHistoryReference(value)
	if explicit && payload == "" {
		return resolvedHistoryReference{}, eris.Errorf("%s reference cannot be empty", prefix)
	}

	switch prefix {
	case "tag":
		hash, err := s.requireTagCommit(payload)
		return resolvedHistoryReference{revision: hash}, err
	case "sha":
		hash, err := s.requireCommitID(payload)
		return resolvedHistoryReference{revision: hash}, err
	case "date":
		when, err := parseHistoryDate(payload, bound)
		return resolvedHistoryReference{timestamp: when}, err
	}

	if hash, found, err := s.tryResolveTagCommit(value); found {
		return resolvedHistoryReference{revision: hash}, err
	}

	if hash, candidate, err := s.tryResolveCommitID(value); candidate {
		return resolvedHistoryReference{revision: hash}, err
	}

	if when, err := parseHistoryDate(value, bound); err == nil {
		return resolvedHistoryReference{timestamp: when}, nil
	}

	return resolvedHistoryReference{}, eris.Errorf(
		"history reference %q is not a tag, commit ID, or supported date",
		value,
	)
}

func splitHistoryReference(value string) (prefix, payload string, explicit bool) {
	for _, prefix := range []string{"tag", "sha", "date"} {
		marker := prefix + ":"
		if strings.HasPrefix(value, marker) {
			return prefix, strings.TrimPrefix(value, marker), true
		}
	}

	return "", value, false
}
```

Refactor the existing tag peeling logic into `tryResolveTagCommit`, returning
`found=false` only for `plumbing.ErrReferenceNotFound`. Implement
`requireTagCommit` to turn `found=false` into `tag %q not found`.

Implement `tryResolveCommitID` by first requiring 4–40 hexadecimal characters,
then resolving `plumbing.Revision(value)` through go-git and confirming the
result with `CommitObject`. Its boolean means that an object match or ambiguity
was found, not merely that the input is hexadecimal: unknown hexadecimal input
returns `found=false` so a compact date such as `20260905` can fall through to
date parsing. Return `found=false` for non-hex input, preserve ambiguous-revision
errors as `found=true`, and report matching non-commit object IDs explicitly.
`requireCommitID` converts `found=false` into an unknown commit ID error for the
strict `sha:` form.

- [ ] **Step 4: Run the prefix and precedence tests**

Run:

```bash
go test ./internal/provider/git -run 'TestResolveHistoryReference_(UsesExplicitPrefixes|UnprefixedTagWins)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit reference resolution**

```bash
git add internal/provider/git/history_reference.go internal/provider/git/history_reference_test.go
git commit -m "Resolve Git history references"
```

### Task 2: Support All Date and Timestamp Forms

**Files:**
- Modify: `internal/provider/git/history_reference.go`
- Modify: `internal/provider/git/history_reference_test.go`

- [ ] **Step 1: Write failing table tests for date formats and bound expansion**

Add:

```go
func TestParseHistoryDate_SupportsDocumentedFormats(t *testing.T) {
	t.Parallel()
	local := time.FixedZone("test-local", 12*60*60)
	previous := time.Local
	time.Local = local
	t.Cleanup(func() { time.Local = previous })

	tests := []struct {
		value string
		bound historyBound
		want  time.Time
	}{
		{"2026-09-05", lowerBound, time.Date(2026, 9, 5, 0, 0, 0, 0, local)},
		{"20260905", lowerBound, time.Date(2026, 9, 5, 0, 0, 0, 0, local)},
		{"20260905Z", lowerBound, time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)},
		{"20260905-1430", lowerBound, time.Date(2026, 9, 5, 14, 30, 0, 0, local)},
		{"20260905-1430Z", lowerBound, time.Date(2026, 9, 5, 14, 30, 0, 0, time.UTC)},
		{"2026-09-05T14:30:45Z", lowerBound, time.Date(2026, 9, 5, 14, 30, 45, 0, time.UTC)},
		{"2026-09-05T14:30:45+12:00", lowerBound, time.Date(2026, 9, 5, 14, 30, 45, 0, local)},
		{"2026-09-05T14:30:45", lowerBound, time.Date(2026, 9, 5, 14, 30, 45, 0, local)},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := parseHistoryDate(tt.value, tt.bound)
			NewWithT(t).Expect(err).NotTo(HaveOccurred())
			NewWithT(t).Expect(got.Equal(tt.want)).To(BeTrue())
		})
	}
}

func TestParseHistoryDate_UpperDateIncludesEntireDay(t *testing.T) {
	t.Parallel()
	got, err := parseHistoryDate("20260905", upperBound)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	NewWithT(t).Expect(got).To(Equal(
		time.Date(2026, 9, 6, 0, 0, 0, 0, time.Local).Add(-time.Nanosecond),
	))
}
```

Do not run tests that mutate `time.Local` in parallel; isolate timezone setup in
a non-parallel test or pass a location into an internal parser helper.

- [ ] **Step 2: Run date tests and verify they fail**

Run:

```bash
go test ./internal/provider/git -run 'TestParseHistoryDate' -count=1
```

Expected: FAIL for unsupported compact and local timestamp layouts.

- [ ] **Step 3: Implement exact date parsing**

Use a layout table that records whether a format is date-only and whether it has
an explicit timezone:

```go
type historyDateLayout struct {
	layout       string
	dateOnly     bool
	explicitZone bool
}

var historyDateLayouts = []historyDateLayout{
	{layout: time.RFC3339Nano, explicitZone: true},
	{layout: "2006-01-02T15:04:05", explicitZone: false},
	{layout: "2006-01-02T15:04", explicitZone: false},
	{layout: "2006-01-02", dateOnly: true, explicitZone: false},
	{layout: "20060102Z", dateOnly: true, explicitZone: true},
	{layout: "20060102", dateOnly: true, explicitZone: false},
	{layout: "20060102-1504Z", explicitZone: true},
	{layout: "20060102-1504", explicitZone: false},
}
```

Parse zoned layouts with `time.Parse` and local layouts with
`time.ParseInLocation(..., time.Local)`. For an upper date-only bound, return
the next local calendar day minus one nanosecond. Return an error listing the
accepted families rather than every Go layout.

- [ ] **Step 4: Run all reference parser tests**

Run:

```bash
go test ./internal/provider/git -run 'Test(ParseHistoryDate|ResolveHistoryReference)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit date parsing**

```bash
git add internal/provider/git/history_reference.go internal/provider/git/history_reference_test.go
git commit -m "Parse Git history reference dates"
```

### Task 3: Integrate Unified Bounds with Graph Selection

**Files:**
- Modify: `internal/provider/git/history_range.go`
- Modify: `internal/provider/git/history_range_test.go`
- Modify: `internal/provider/git/commit_test.go`
- Modify: `internal/provider/git/author_history_range_test.go`
- Modify: `internal/stages/git_history_test.go`

- [ ] **Step 1: Convert range tests to raw references and add mixed-bound coverage**

Change the public type and existing test inputs:

```go
type HistoryRange struct {
	From  string
	Until string
}
```

```go
commits, err := s.commitIterator(HistoryRange{From: "v1.0", Until: "v2.0"})
```

Add a mixed case:

```go
func TestHistoryRange_MixesRevisionAndDateBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

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
```

Add a date/date reversal test expecting
`--from must be before or equal to --until`.

- [ ] **Step 2: Run graph tests and verify they fail**

Run:

```bash
go test ./internal/provider/git ./internal/stages -run 'TestHistoryRange|TestLoadGitHistory_.*Range' -count=1
```

Expected: compilation failures because the iterator still expects typed date and tag fields.

- [ ] **Step 3: Resolve both bounds before selecting commits**

Refactor `resolveHistoryRange`:

```go
func (s *repoService) resolveHistoryRange(r HistoryRange) (resolvedHistoryRange, error) {
	from, err := s.resolveHistoryReference(r.From, lowerBound)
	if err != nil {
		return resolvedHistoryRange{}, eris.Wrap(err, "invalid --from")
	}

	until, err := s.resolveHistoryReference(r.Until, upperBound)
	if err != nil {
		return resolvedHistoryRange{}, eris.Wrap(err, "invalid --until")
	}

	if !from.timestamp.IsZero() && !until.timestamp.IsZero() &&
		from.timestamp.After(until.timestamp) {
		return resolvedHistoryRange{}, eris.New("--from must be before or equal to --until")
	}

	tip, tipLabel, err := s.resolveRangeTip(until)
	if err != nil {
		return resolvedHistoryRange{}, err
	}

	resolved := resolvedHistoryRange{
		tip:   tip,
		from:  from.timestamp,
		until: until.timestamp,
	}
	if !from.hasRevision() {
		return resolved, nil
	}

	return s.excludeLowerRevision(resolved, from.revision, r.From, tipLabel)
}
```

Make `resolveRangeTip` accept a resolved reference, use its revision when set,
and otherwise resolve `HEAD`. Generalize ancestry diagnostics from “tag” to
“history reference”.

Remove the old `resolveTagCommit` from `history_range.go`; tag resolution now
lives in `history_reference.go`.

- [ ] **Step 4: Update compatibility wrappers**

The date-based exported wrappers in `commit.go` and `author_history.go` must
retain their public behavior. Convert their `time.Time` arguments to strict
`date:` references with `time.RFC3339Nano`, which preserves the instant and
offset:

```go
func historyRangeFromTimes(from, until time.Time) HistoryRange {
	var result HistoryRange
	if !from.IsZero() {
		result.From = "date:" + from.Format(time.RFC3339Nano)
	}
	if !until.IsZero() {
		result.Until = "date:" + until.Format(time.RFC3339Nano)
	}
	return result
}
```

Use this helper in `CommitTotalInRange`, `BulkCommitHistoryInRange`,
`BulkCommitHistoryAndPrewarmInRange`, and the authorship date wrapper.

- [ ] **Step 5: Update all provider and stage fixtures**

Replace `FromTag`/`UntilTag` literals with `From`/`Until` raw references in:

```text
internal/provider/git/commit_test.go
internal/provider/git/author_history_range_test.go
internal/stages/git_history_test.go
```

Preserve assertions that totals, tracked history, authorship, and metric
prewarming all select the same commits.

- [ ] **Step 6: Run provider and stage tests**

Run:

```bash
go test ./internal/provider/git ./internal/stages -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit graph integration**

```bash
git add internal/provider/git internal/stages/git_history_test.go
git commit -m "Apply unified Git history ranges"
```

### Task 4: Replace the CLI Surface

**Files:**
- Delete: `cmd/codeviz/date_range.go`
- Delete: `cmd/codeviz/date_range_test.go`
- Create: `cmd/codeviz/history_range.go`
- Create: `cmd/codeviz/history_range_test.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/donuttree_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/render_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/main_test.go`
- Modify: `cmd/codeviz/render_cmd_test.go`

- [ ] **Step 1: Write failing CLI tests for unified and removed flags**

Replace the old tag flag test:

```go
func TestCLI_ParsesUnifiedHistoryReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cli := CLI{}
	parser, err := kong.New(&cli, kong.Name("codeviz"), filterMapperOption(), kong.Exit(func(int) {}))
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"tree-map", ".", "-o", "out.png",
		"--from", "sha:abc1234",
		"--until", "tag:v2.0",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.TreeMap.From).To(Equal("sha:abc1234"))
	g.Expect(cli.TreeMap.Until).To(Equal("tag:v2.0"))
}

func TestCLI_RejectsRemovedTagRangeFlags(t *testing.T) {
	t.Parallel()
	cli := CLI{}
	parser, err := kong.New(&cli, kong.Name("codeviz"), filterMapperOption(), kong.Exit(func(int) {}))
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"tree-map", ".", "-o", "out.png", "--from-tag", "v1.0",
	})
	NewWithT(t).Expect(err).To(HaveOccurred())
}
```

Change the render forwarding test to assert `From` and `Until`.

- [ ] **Step 2: Run command tests and verify they fail**

Run:

```bash
go test ./cmd/codeviz -run 'Test(CLI_.*History|CLI_RejectsRemoved|RenderCmd_Forwards)' -count=1
```

Expected: FAIL because the old tag options still parse.

- [ ] **Step 3: Replace CLI parsing with raw range transfer**

Create `cmd/codeviz/history_range.go`:

```go
package main

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func parseHistoryRange(fromValue, untilValue string) git.HistoryRange {
	return git.HistoryRange{From: fromValue, Until: untilValue}
}

func stagesFlagsForCommand(
	flags *Flags,
	fromValue, untilValue string,
) *stages.Flags {
	parsedFlags := toStagesFlags(flags)
	parsedFlags.HistoryRange = parseHistoryRange(fromValue, untilValue)
	return parsedFlags
}
```

Remove CLI date validation because interpretation requires a repository. Delete
the old date parsing functions and tests.

- [ ] **Step 4: Update every command**

For each visualization and `RenderCmd`:

1. remove `FromTag` and `UntilTag`;
2. update help to `Git history lower bound: tag, commit ID, or date. Prefix with tag:, sha:, or date: to disambiguate.` and the corresponding upper-bound wording;
3. remove `Validate` calls that only parsed history dates;
4. call `stagesFlagsForCommand(flags, c.From, c.Until)` without an error branch; and
5. forward only `From` and `Until` from presets.

Keep other validation logic in `RenderCmd.Validate` unchanged.

- [ ] **Step 5: Run all command tests**

Run:

```bash
go test ./cmd/codeviz -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the CLI migration**

```bash
git add cmd/codeviz
git commit -m "Unify Git history range flags"
```

### Task 5: Document and Validate the Feature

**Files:**
- Modify: `docs/content/docs/usage.md`

- [ ] **Step 1: Update user documentation**

Replace the four-option table with:

```markdown
| Flag            | Includes                                                     |
| --------------- | ------------------------------------------------------------ |
| `--from <ref>`  | Commits after a revision, or on/after a date or timestamp    |
| `--until <ref>` | A revision and its history, or commits on/before a timestamp |
```

Document:

```markdown
`<ref>` is resolved as an exact tag, then a unique short or full commit ID,
then a date. Use `tag:`, `sha:`, or `date:` to force an interpretation.

Dates accept ISO 8601 or `YYYYMMDD[-HHMM][Z]`. Values without a timezone use
local time; a date-only `--until` includes the complete day.
```

Retain the full-graph, lower-exclusive, upper-inclusive, ancestry, and current
checkout explanations. Update examples to show unprefixed tags, an explicit
short SHA, and a compact date.

- [ ] **Step 2: Format the changed Go files**

Run:

```bash
task fmt
```

Expected: exit status 0.

- [ ] **Step 3: Run targeted tests**

Run:

```bash
go test ./cmd/codeviz ./internal/provider/git ./internal/stages -count=1
```

Expected: PASS.

- [ ] **Step 4: Run repository CI through the required noisy-command agent**

Dispatch an Explore-equivalent task agent to run:

```bash
task ci
```

Require only the exit status, failing test/linter identities, file:line
diagnostics, or a one-line success result. Expected: exit status 0 with no
failing tests or linters.

- [ ] **Step 5: Commit documentation and formatting**

```bash
git add docs/content/docs/usage.md cmd internal
git commit -m "Document unified Git history references"
```

- [ ] **Step 6: Review the complete diff**

Run:

```bash
git status --short
git diff --check main...HEAD
git diff --stat main...HEAD
```

Expected: clean worktree, no whitespace errors, and changes limited to the
design/plan, CLI range options, Git provider range resolution, tests, and usage
documentation.
