package source

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	. "github.com/onsi/gomega"
)

func TestGitFSConforms(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := newFixtureGitFS(t)

	g.Expect(fstest.TestFS(
		fsys,
		"README.md",
		"cmd/tool/main.go",
		"script.sh",
		"link",
	)).To(Succeed())
}

func TestGitFSExposesExecutableAndSymlink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := newFixtureGitFS(t)

	info, err := fs.Stat(fsys, "script.sh")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Mode().Perm()).To(Equal(fs.FileMode(0o755)))

	linkInfo, err := fs.Lstat(fsys, "link")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(linkInfo.Mode() & fs.ModeSymlink).NotTo(BeZero())

	target, err := fs.ReadLink(fsys, "link")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target).To(Equal("README.md"))
}

func newFixtureGitFS(t *testing.T) fs.FS {
	t.Helper()
	g := NewWithT(t)
	dir := t.TempDir()

	repo, err := gogit.PlainInit(dir, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(os.MkdirAll(filepath.Join(dir, "cmd", "tool"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "cmd", "tool", "main.go"), []byte("package main\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/sh\n"), 0o755)).To(Succeed())
	g.Expect(os.Symlink("README.md", filepath.Join(dir, "link"))).To(Succeed())

	worktree, err := repo.Worktree()
	g.Expect(err).NotTo(HaveOccurred())
	_, err = worktree.Add(".")
	g.Expect(err).NotTo(HaveOccurred())
	hash, err := worktree.Commit("fixture", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC),
		},
	})
	g.Expect(err).NotTo(HaveOccurred())

	commit, err := repo.CommitObject(hash)
	g.Expect(err).NotTo(HaveOccurred())
	tree, err := commit.Tree()
	g.Expect(err).NotTo(HaveOccurred())

	return NewGitFS(tree, commit.Committer.When)
}
