package source

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	. "github.com/onsi/gomega"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGitFSConforms(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := newFixtureGitFS(t)

	g.Expect(fstest.TestFS(
		fsys,
		"README.md",
		"cmd/tool/main.go",
		"foo/item.txt",
		"foo.bar",
		"script.sh",
		"link",
	)).To(Succeed())
}

func TestGitFSReadDirSortsByFilename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	entries, err := fs.ReadDir(newFixtureGitFS(t), ".")
	g.Expect(err).NotTo(HaveOccurred())

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	g.Expect(names).To(Equal([]string{"README.md", "cmd", "foo", "foo.bar", "link", "script.sh"}))
}

func TestGitFSRejectsMalformedTreeEntryName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := NewGitFS(&object.Tree{
		Entries: []object.TreeEntry{{Name: ".", Mode: filemode.Dir}},
	}, time.Time{})

	_, err := fs.ReadDir(fsys, ".")
	g.Expect(errors.Is(err, fs.ErrInvalid)).To(BeTrue())
}

func TestGitFSExposesExecutableAndSymlink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := newFixtureGitFS(t)

	info, err := fs.Stat(fsys, "script.sh")
	g.Expect(err).NotTo(HaveOccurred())

	if info == nil {
		t.Fatal("expected executable file info")
	}

	g.Expect(info.Mode().Perm()).To(Equal(fs.FileMode(0o755)))

	linkInfo, err := fs.Lstat(fsys, "link")
	g.Expect(err).NotTo(HaveOccurred())

	if linkInfo == nil {
		t.Fatal("expected symlink file info")
	}

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
	g.Expect(os.Mkdir(filepath.Join(dir, "foo"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "cmd", "tool", "main.go"), []byte("package main\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "foo", "item.txt"), []byte("item\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "foo.bar"), []byte("file\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/sh\n"), 0o600)).To(Succeed())
	g.Expect(os.Chmod(filepath.Join(dir, "script.sh"), 0o700)).To(Succeed())
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

	if commit == nil {
		t.Fatal("expected fixture commit")
	}

	tree, err := commit.Tree()
	g.Expect(err).NotTo(HaveOccurred())

	if tree == nil {
		t.Fatal("expected fixture tree")
	}

	return NewGitFS(tree, commit.Committer.When)
}
