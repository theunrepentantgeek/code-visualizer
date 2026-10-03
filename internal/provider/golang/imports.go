package golang

import (
	"strings"

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
