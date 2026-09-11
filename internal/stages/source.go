package stages

import (
	"io/fs"
	"os"
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
		return eris.Wrap(err, "failed to open working tree source")
	}

	until := ""
	if c.Flags != nil {
		until = strings.TrimSpace(c.Flags.HistoryRange.Until)
	}

	repositoryPath := c.TargetPath
	if until != "" {
		repositoryPath, err = nearestExistingDirectory(tree.RootPath)
		if err != nil {
			return eris.Wrap(err, "failed to locate historical target repository")
		}
	}

	repository, err := resolveRepositorySource(tree, repositoryPath)
	if err != nil {
		return err
	}

	tree = repository.tree
	if repository.discoveryErr == nil {
		c.RepoRoot = repository.root
	}

	if until == "" {
		c.Source = tree

		return nil
	}

	return resolveHistoricalSource(c, tree, repository, until)
}

func resolveHistoricalSource(
	c *CommonState,
	tree source.Tree,
	repository repositorySource,
	until string,
) error {
	if repository.discoveryErr != nil {
		return eris.Wrap(repository.discoveryErr, "--until requires a Git repository")
	}

	snapshot, err := git.ResolveSnapshot(repository.root, until)
	if err != nil {
		return eris.Wrap(err, "failed to resolve historical filesystem snapshot")
	}

	subtree, err := snapshot.Subtree(tree.RepoBase)
	if err != nil {
		return eris.Wrap(err, "failed to scope historical snapshot")
	}

	modTime := snapshot.Commit.Committer.When
	if modTime.IsZero() {
		modTime = snapshot.Commit.Author.When
	}

	tree.FS = source.NewGitFS(subtree, modTime)
	tree.RepoFS = source.NewGitFS(snapshot.Tree, modTime)
	tree.Clock = modTime
	c.Source = tree
	c.Snapshot = &snapshot
	c.ReferenceNow = modTime

	return nil
}

type repositorySource struct {
	tree         source.Tree
	root         string
	discoveryErr error
}

func resolveRepositorySource(tree source.Tree, repositoryPath string) (repositorySource, error) {
	repoRoot, err := git.RepoRootFor(repositoryPath)
	if err != nil {
		return repositorySource{tree: tree, discoveryErr: err}, nil //nolint:nilerr // Valid for live sources.
	}

	tree.RepoRoot = repoRoot

	tree.RepoBase, err = repoRelativeDirectory(repoRoot, tree.RootPath)
	if err != nil {
		return repositorySource{}, err
	}

	repositoryTree, err := source.WorkingTree(repoRoot)
	if err != nil {
		return repositorySource{}, eris.Wrap(err, "failed to open repository source")
	}

	tree.RepoFS = repositoryTree.FS

	return repositorySource{tree: tree, root: repoRoot}, nil
}

func nearestExistingDirectory(name string) (string, error) {
	for {
		info, err := os.Stat(name)
		if err == nil {
			if info.IsDir() {
				return name, nil
			}

			name = filepath.Dir(name)

			continue
		}

		if !os.IsNotExist(err) {
			return "", eris.Wrapf(err, "accessing %s", name)
		}

		parent := filepath.Dir(name)
		if parent == name {
			return "", eris.Wrapf(err, "finding existing ancestor of %s", name)
		}

		name = parent
	}
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
