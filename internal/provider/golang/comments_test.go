package golang

import (
	"go/ast"
	"go/token"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/dave/dst/decorator"
)

func TestComputeCommentRatio(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		src  string
		want float64
	}{
		"no comments": {
			src:  "package p\n\nfunc f() {}\n",
			want: 0.0,
		},
		"all comments": {
			src:  "package p\n// comment only\n// another comment\n",
			want: 2.0 / 3.0,
		},
		"mixed code and comments": {
			src: `package p
// a comment
func f() {}
`,
			want: 1.0 / 3.0,
		},
		"inline comment counts both": {
			src: `package p
func f() {} // inline
`,
			want: 0.5,
		},
		"block comment": {
			src: `package p
/* block comment */
func f() {}
`,
			want: 1.0 / 3.0,
		},
		"multi-line block comment": {
			src: `package p
/*
multi-line
block
*/
func f() {}
`,
			want: 4.0 / 6.0,
		},
		"blank lines ignored": {
			src: `package p

// comment

func f() {}

`,
			want: 1.0 / 3.0,
		},
		"code only": {
			src: `package p

func f() {}
func g() {}
`,
			want: 0.0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			fset := token.NewFileSet()
			dec := decorator.NewDecorator(fset)
			dstFile, err := dec.Parse(tt.src)
			g.Expect(err).NotTo(HaveOccurred())

			astFile, ok := dec.Ast.Nodes[dstFile].(*ast.File)
			g.Expect(ok).To(BeTrue(), "decorator node map should contain *ast.File")

			ratio := computeCommentRatio([]byte(tt.src), astFile.Comments, fset)
			g.Expect(ratio).To(BeNumerically("~", tt.want, 0.01))
		})
	}
}
