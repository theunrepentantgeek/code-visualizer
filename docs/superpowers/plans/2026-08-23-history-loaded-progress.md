# History Loaded Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit a concise completion message with the commit count after git history loads successfully.

**Architecture:** Keep success reporting at the `LoadGitHistory` operation boundary. After validating and storing the non-empty result, log `History loaded` with `len(commits)`; preserve quiet-mode suppression and all existing error paths.

**Tech Stack:** Go 1.26.1, `log/slog`, Gomega

---

## File Structure

- Modify `internal/stages/git_history.go`: emit the successful history completion event.
- Modify `internal/stages/git_history_test.go`: verify the exact commit count and quiet-mode suppression through `LoadGitHistory`.

### Task 1: Report Successful History Loading

**Files:**
- Modify: `internal/stages/git_history.go:62-64`
- Test: `internal/stages/git_history_test.go:85-100`

- [ ] **Step 1: Write the failing completion and quiet-mode tests**

Replace `TestLoadGitHistory_ReportsInitialProgressInDefaultMode` and add the quiet-mode test:

```go
//nolint:paralleltest // mutates global slog default logger
func TestLoadGitHistory_ReportsProgressAndCompletionInDefaultMode(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	state := buildHistoryState(setupHistoryRepo(t))

	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading git history"`))
	g.Expect(buf.String()).To(ContainSubstring(`msg="History loaded" commits=3`))
}

//nolint:paralleltest // mutates global slog default logger
func TestLoadGitHistory_QuietModeOmitsCompletion(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	state := buildHistoryState(setupHistoryRepo(t))
	state.Flags.Quiet = true

	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(buf.String()).NotTo(ContainSubstring("History loaded"))
}
```

- [ ] **Step 2: Run the focused tests to verify the completion test fails**

Run:

```bash
go test ./internal/stages -run 'TestLoadGitHistory_(ReportsProgressAndCompletionInDefaultMode|QuietModeOmitsCompletion)$' -count=1
```

Expected: FAIL because the default-mode output does not contain `msg="History loaded" commits=3`.

- [ ] **Step 3: Add the minimal completion log**

Update the successful end of `LoadGitHistory`:

```go
c.GitHistory = commits

if !c.Flags.Quiet {
	slog.Info("History loaded", "commits", len(commits))
}

return nil
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run:

```bash
go test ./internal/stages -run 'TestLoadGitHistory_(ReportsProgressAndCompletionInDefaultMode|QuietModeOmitsCompletion)$' -count=1
```

Expected: PASS.

- [ ] **Step 5: Run the complete stages package tests**

Run:

```bash
go test ./internal/stages
```

Expected: PASS.

- [ ] **Step 6: Commit the implementation**

```bash
git add internal/stages/git_history.go internal/stages/git_history_test.go
git commit -m "feat: report completed history loading" -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>"
```
