package stages

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/source"
)

// ResolveSource selects the live working tree or the committed --until tree.
func ResolveSource(c *CommonState) error {
	tree, err := source.WorkingTree(c.TargetPath)
	if err != nil {
		return err
	}

	repoRoot, repoErr := git.RepoRootFor(c.TargetPath)
	if repoErr == nil {
		tree.RepoRoot = repoRoot
		tree.RepoBase, err = repoRelativeDirectory(repoRoot, tree.RootPath)
		if err != nil {
			return err
		}
		c.RepoRoot = repoRoot
	}

	until := ""
	if c.Flags != nil {
		until = strings.TrimSpace(c.Flags.HistoryRange.Until)
	}
	if until == "" {
		c.Source = tree

		return nil
	}
	if repoErr != nil {
		return eris.Wrap(repoErr, "--until requires a Git repository")
	}

	snapshot, err := git.ResolveSnapshot(c.TargetPath, until)
	if err != nil {
		return eris.Wrap(err, "failed to resolve historical filesystem snapshot")
	}
	subtree, err := snapshot.Subtree(tree.RepoBase)
	if err != nil {
		return err
	}

	modTime := snapshot.Commit.Committer.When
	if modTime.IsZero() {
		modTime = snapshot.Commit.Author.When
	}
	tree.FS = source.NewGitFS(subtree, modTime)
	tree.Clock = modTime
	c.Source = tree
	c.Snapshot = &snapshot
	c.ReferenceNow = modTime

	return nil
}

func repoRelativeDirectory(repoRoot, targetPath string) (string, error) {
	relative, err := filepath.Rel(repoRoot, targetPath)
	if err != nil {
		return "", eris.Wrap(err, "failed to resolve repository-relative target")
	}

	relative = filepath.ToSlash(relative)
	if relative == ".." || strings.HasPrefix(relative, "../") {
		return "", &fs.PathError{Op: "resolve", Path: targetPath, Err: fs.ErrInvalid}
	}

	return relative, nil
}
