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
		complexity += complexityContribution(n)

		return true
	})

	return complexity
}

func complexityContribution(node dst.Node) int64 {
	switch node := node.(type) {
	case *dst.IfStmt, *dst.ForStmt, *dst.RangeStmt:
		return 1
	case *dst.CaseClause:
		if node.List != nil {
			return 1
		}
	case *dst.CommClause:
		if node.Comm != nil {
			return 1
		}
	case *dst.BinaryExpr:
		if node.Op == token.LAND || node.Op == token.LOR {
			return 1
		}
	default:
	}

	return 0
}
