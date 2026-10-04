package model

import (
	"path"

	"github.com/rotisserie/eris"
)

type DirectoryScope uint8

const (
	DirectoryScopeInvalid DirectoryScope = iota
	DirectorySubtree
	DirectoryDirectFiles
)

type DirectorySelection struct {
	Directory *Directory
	Scope     DirectoryScope
}

func (s DirectorySelection) WalkFiles(fn func(*File)) error {
	if s.Directory == nil {
		return eris.New("directory selection requires a directory")
	}

	switch s.Scope {
	case DirectorySubtree:
		WalkFiles(s.Directory, fn)
	case DirectoryDirectFiles:
		for _, file := range s.Directory.Files {
			fn(file)
		}
	default:
		return eris.Errorf("invalid directory scope %d", s.Scope)
	}

	return nil
}

func PartitionDirectories(root *Directory, expansions []string) map[string]DirectorySelection {
	result := make(map[string]DirectorySelection)
	if root == nil || CountFiles(root) == 0 {
		return result
	}

	expanded := make(map[string]struct{}, len(expansions))
	for _, expansion := range expansions {
		expanded[path.Join(root.RepoPath, path.Clean(expansion))] = struct{}{}
	}

	partitionDirectory(root, expanded, result)

	return result
}

func partitionDirectory(
	directory *Directory,
	expanded map[string]struct{},
	result map[string]DirectorySelection,
) {
	if len(directory.Files) > 0 {
		result[directory.RepoPath] = DirectorySelection{
			Directory: directory,
			Scope:     DirectoryDirectFiles,
		}
	}

	for _, child := range directory.Dirs {
		if child == nil || CountFiles(child) == 0 {
			continue
		}

		if _, ok := expanded[child.RepoPath]; ok {
			partitionDirectory(child, expanded, result)

			continue
		}

		result[child.RepoPath] = DirectorySelection{
			Directory: child,
			Scope:     DirectorySubtree,
		}
	}
}
