# Unify Git History Traversal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the duplicated Git history and cache-prewarm commit walks with one canonical internal traversal.

**Architecture:** `repoService` will own one traversal that iterates tracked changes and optionally records timeline commits or updates a prewarm cache. The two exported history APIs and the cache-only prewarm operation will call that traversal, preserving their existing callback, filtering, and cache-publication behavior.

**Tech Stack:** Go 1.26, go-git v5, Gomega, Task.

---

### Task 1: Unify tracked history walking

**Files:**
- Modify: `internal/provider/git/commit.go`
- Modify: `internal/provider/git/service.go`
- Modify: `internal/provider/git/commit_test.go`
- Test: `internal/provider/git/commit_test.go`

- [ ] **Step 1: Write the failing regression test**

Add a test that invokes the public history-only API and the history-plus-prewarm
API against the same tracked repository, then asserts equal commit metadata and
paths while confirming the requested cache data is present.

```go
g.Expect(historyOnly).To(Equal(historyAndPrewarm))
g.Expect(service.cachedCommitData("tracked.go")).NotTo(BeNil())
```

- [ ] **Step 2: Run the focused test to verify the current duplication is exposed**

Run: `go test ./internal/provider/git -run TestBulkCommitHistory -count=1`

Expected: PASS for behavior; the test establishes the equivalence contract before
removing the duplicate implementation.

- [ ] **Step 3: Implement the canonical walker**

Move commit iteration, tracked-change filtering, progress notification, optional
timeline collection, and optional cache updates behind an unexported
`repoService` helper. Make `BulkCommitHistory`,
`BulkCommitHistoryAndPrewarm`, and `doBulkPrewarm` call it; retain
`mergeBulkPrewarmCache` after successful traversal only.

```go
func (s *repoService) walkTrackedHistory(
    tracked map[string]bool,
    requirements metricRequirements,
    collectHistory bool,
    onCommitProcessed func(),
) ([]Commit, map[string]*commitData, error)
```

- [ ] **Step 4: Run focused Git-provider tests**

Run: `go test ./internal/provider/git -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the refactor**

```bash
git add internal/provider/git/commit.go internal/provider/git/service.go internal/provider/git/commit_test.go
git commit -m "refactor(git): unify history traversal"
```

### Task 2: Verify integration behavior

**Files:**
- Test: `internal/stages/git_history_test.go`

- [ ] **Step 1: Run the spiral history prewarm integration test**

Run: `go test ./internal/stages -run TestLoadGitHistory_PrewarmsRequestedGitMetricsForRunProviders -count=1`

Expected: PASS.

- [ ] **Step 2: Run the project suite**

Run: `task test`

Expected: PASS.
