package golang

import (
	"go/token"

	"github.com/dave/dst"
)

// cyclomaticComplexity computes cyclomatic complexity for a single function body.
// Base complexity is 1, plus 1 for each decision point:
// if, for, range, case (non-default), &&, ||.
func cyclomaticComplexity(body *dst.BlockStmt) int64 {
	if body == nil {
		return 1
	}

	complexity := int64(1)

	dst.Inspect(body, func(n dst.Node) bool {
		switch node := n.(type) {
		case *dst.IfStmt:
			complexity++
		case *dst.ForStmt:
			complexity++
		case *dst.RangeStmt:
			complexity++
		case *dst.CaseClause:
			if node.List != nil {
				complexity++
			}
		case *dst.CommClause:
			if node.Comm != nil {
				complexity++
			}
		case *dst.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}

		return true
	})

	return complexity
}
