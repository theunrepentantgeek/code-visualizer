package golang

import (
	"bytes"
	"go/ast"
	"go/token"
)

// computeCommentRatio computes the ratio of comment lines to code lines.
// Blank lines are excluded from both counts. Lines with both code and a comment
// count for both totals. Comment positions come from the AST produced by the
// decorator's single parse pass — not a separate parse.
func computeCommentRatio(
	src []byte,
	comments []*ast.CommentGroup,
	fset *token.FileSet,
) float64 {
	commentLineSet := buildCommentLineSet(comments, fset)
	commentOnlySet := buildCommentOnlyLineSet(src, comments, fset)
	srcLines := bytes.Split(src, []byte("\n"))

	var (
		codeCount    int64
		commentCount int64
	)

	for i, line := range srcLines {
		lineNum := i + 1

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		if commentLineSet[lineNum] {
			commentCount++
		}

		if !commentOnlySet[lineNum] {
			codeCount++
		}
	}

	if codeCount == 0 {
		return 0.0
	}

	return float64(commentCount) / float64(codeCount)
}

// buildCommentLineSet returns the set of line numbers that contain comment text.
func buildCommentLineSet(comments []*ast.CommentGroup, fset *token.FileSet) map[int]bool {
	set := make(map[int]bool)

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

// buildCommentOnlyLineSet returns the set of line numbers where the entire
// non-whitespace content is comment text (no code on the same line).
func buildCommentOnlyLineSet(
	src []byte,
	comments []*ast.CommentGroup,
	fset *token.FileSet,
) map[int]bool {
	set := make(map[int]bool)
	srcLines := bytes.Split(src, []byte("\n"))

	for _, cg := range comments {
		for _, c := range cg.List {
			markCommentOnlyLines(c, srcLines, fset, set)
		}
	}

	return set
}

func markCommentOnlyLines(
	c *ast.Comment,
	srcLines [][]byte,
	fset *token.FileSet,
	set map[int]bool,
) {
	startPos := fset.Position(c.Pos())
	endPos := fset.Position(c.End())

	// Interior lines of multi-line block comments are always comment-only.
	for line := startPos.Line + 1; line < endPos.Line; line++ {
		set[line] = true
	}

	markStartLine(startPos, srcLines, set)
	markEndLine(startPos, endPos, srcLines, set)
}

func markStartLine(startPos token.Position, srcLines [][]byte, set map[int]bool) {
	if startPos.Line > len(srcLines) {
		return
	}

	line := srcLines[startPos.Line-1]
	trimmed := bytes.TrimSpace(line)

	if bytes.HasPrefix(trimmed, []byte("//")) || bytes.HasPrefix(trimmed, []byte("/*")) {
		set[startPos.Line] = true
	}
}

func markEndLine(startPos, endPos token.Position, srcLines [][]byte, set map[int]bool) {
	if endPos.Line == startPos.Line || endPos.Line > len(srcLines) {
		return
	}

	line := srcLines[endPos.Line-1]

	_, after, found := bytes.Cut(line, []byte("*/"))
	if found && len(bytes.TrimSpace(after)) == 0 {
		set[endPos.Line] = true
	}
}
