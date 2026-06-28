package golang

import (
	"bytes"
	"go/ast"
	"go/token"
)

// computeCommentRatio computes the ratio of comment lines to total non-blank lines.
// The result is always in [0, 1]. Blank lines are excluded from both counts.
// Lines with both code and a comment count as comment lines. Comment positions
// come from the AST produced by the decorator's single parse pass.
func computeCommentRatio(
	src []byte,
	comments []*ast.CommentGroup,
	fset *token.FileSet,
) float64 {
	commentLineSet := buildCommentLineSet(comments, fset)
	srcLines := bytes.Split(src, []byte("\n"))

	var (
		totalNonBlank int64
		commentCount  int64
	)

	for i, line := range srcLines {
		lineNum := i + 1

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		totalNonBlank++

		if commentLineSet[lineNum] {
			commentCount++
		}
	}

	if totalNonBlank == 0 {
		return 0.0
	}

	return float64(commentCount) / float64(totalNonBlank)
}

// buildCommentLineSet returns the set of line numbers that contain comment text.
func buildCommentLineSet(comments []*ast.CommentGroup, fset *token.FileSet) map[int]bool {
	set := make(map[int]bool, len(comments))

	for _, cg := range comments {
		for _, c := range cg.List {
			start := fset.Position(c.Pos()).Line
			end := fset.Position(c.End()).Line

			for line := start; line <= end; line++ {
				set[line] = true
			}
		}
	}

	return set
}
