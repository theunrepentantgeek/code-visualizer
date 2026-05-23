package golang

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dave/dst"
)

// classifyImports categorizes each import in dstFile as stdlib, internal, or
// external, and populates the corresponding stats fields.
//
//nolint:nilaway,nolintlint // caller guarantees non-nil after successful parse
func classifyImports(dstFile *dst.File, modulePath string, stats *fileStats) {
	for _, imp := range dstFile.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		stats.imports++

		switch {
		case isStdlib(path):
			stats.stdlibImports++
		case modulePath != "" && isInternalImport(path, modulePath):
			stats.internalImports++
		default:
			stats.externalImports++
		}
	}
}

// isStdlib reports whether importPath is a Go standard library package.
// Stdlib packages have no dot in the first path element.
func isStdlib(importPath string) bool {
	firstElem, _, _ := strings.Cut(importPath, "/")

	return !strings.Contains(firstElem, ".")
}

// isInternalImport reports whether importPath belongs to the given module.
// It requires either an exact match or a path boundary (modulePath + "/").
func isInternalImport(importPath string, modulePath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}

// moduleCache caches go.mod module path lookups per directory.
type moduleCache struct {
	mu      sync.RWMutex
	modules map[string]string
}

func newModuleCache() *moduleCache {
	return &moduleCache{
		modules: make(map[string]string),
	}
}

// findModulePath walks up from dir looking for go.mod and returns the module
// path. Returns "" if no go.mod is found. Results are cached per directory.
func (mc *moduleCache) findModulePath(dir string) string {
	mc.mu.RLock()

	if path, ok := mc.modules[dir]; ok {
		mc.mu.RUnlock()

		return path
	}

	mc.mu.RUnlock()

	return mc.scanForModulePath(dir)
}

func (mc *moduleCache) scanForModulePath(startDir string) string {
	var visited []string

	dir := startDir

	for {
		mc.mu.RLock()

		if path, ok := mc.modules[dir]; ok {
			mc.mu.RUnlock()
			mc.cacheAll(visited, path)

			return path
		}

		mc.mu.RUnlock()

		visited = append(visited, dir)

		goModPath := filepath.Join(dir, "go.mod")
		if modPath := readModulePath(goModPath); modPath != "" {
			mc.cacheAll(visited, modPath)

			return modPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	mc.cacheAll(visited, "")

	return ""
}

func (mc *moduleCache) cacheAll(dirs []string, modulePath string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, d := range dirs {
		mc.modules[d] = modulePath
	}
}

// readModulePath reads the module path from a go.mod file.
// Returns "" if the file doesn't exist or doesn't contain a module directive.
func readModulePath(goModPath string) string {
	f, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			// Strip trailing inline comments
			if idx := strings.Index(mod, "//"); idx >= 0 {
				mod = strings.TrimSpace(mod[:idx])
			}

			return mod
		}
	}

	if scanner.Err() != nil {
		return ""
	}

	return ""
}
