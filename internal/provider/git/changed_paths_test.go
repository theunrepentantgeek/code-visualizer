package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestChangedPathsInHistoryRange_SelectsCurrentPathsChangedByDate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	changed, err := ChangedPathsInHistoryRange(
		dir,
		map[string]bool{"old.go": true, "shared.go": true, "new.go": true},
		HistoryRange{
			From:  "date:2025-01-01T00:00:00Z",
			Until: "date:2025-12-31T23:59:59Z",
		},
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(changed).To(Equal(map[string]bool{"shared.go": true}))
}

func TestChangedPathsInHistoryRange_UsesUnifiedTagBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	changed, err := ChangedPathsInHistoryRange(
		fixture.dir,
		map[string]bool{"shared.go": true, "main.go": true, "feature.go": true},
		HistoryRange{From: "tag:v1.0", Until: "tag:v2.0"},
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(changed).To(Equal(map[string]bool{
		"main.go":    true,
		"feature.go": true,
	}))
}

func TestChangedPathsInHistoryRange_UsesCurrentRenameDestinationAndOmitsDeletion(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupChangedPathsRepo(t)

	changed, err := ChangedPathsInHistoryRange(
		dir,
		map[string]bool{"renamed.go": true, "stable.go": true},
		HistoryRange{From: "tag:before-changes"},
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(changed).To(Equal(map[string]bool{"renamed.go": true}))
}

func TestChangedPathsInHistoryRange_ReturnsEmptySetWhenNoCurrentPathChanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	changed, err := ChangedPathsInHistoryRange(
		fixture.dir,
		map[string]bool{"shared.go": true},
		HistoryRange{From: "tag:v1.0", Until: "tag:v2.0"},
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(changed).To(BeEmpty())
}

func setupChangedPathsRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
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

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, output)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	writeTestFile(t, filepath.Join(dir, "old.go"), "package old\n")
	writeTestFile(t, filepath.Join(dir, "deleted.go"), "package deleted\n")
	writeTestFile(t, filepath.Join(dir, "stable.go"), "package stable\n")
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")
	run("git", "tag", "before-changes")

	run("git", "mv", "old.go", "renamed.go")
	run("git", "rm", "deleted.go")
	run("git", "commit", "-m", "rename and delete")

	return dir
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
