package golang

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/dave/dst/decorator"
	. "github.com/onsi/gomega"
)

func TestComputeCommentRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want float64
	}{
		{
			name: "no comments",
			src:  "package p\n\nfunc f() {}\n",
			want: 0.0,
		},
		{
			name: "all comments",
			src:  "package p\n// comment only\n// another comment\n",
			want: 2.0,
		},
		{
			name: "mixed code and comments",
			src: `package p
// a comment
func f() {}
`,
			want: 0.5,
		},
		{
			name: "inline comment counts both",
			src: `package p
func f() {} // inline
`,
			want: 0.5,
		},
		{
			name: "block comment",
			src: `package p
/* block comment */
func f() {}
`,
			want: 0.5,
		},
		{
			name: "multi-line block comment",
			src: `package p
/*
multi-line
block
*/
func f() {}
`,
			want: 2.0,
		},
		{
			name: "blank lines ignored",
			src: `package p

// comment

func f() {}

`,
			want: 0.5,
		},
		{
			name: "code only",
			src: `package p

func f() {}
func g() {}
`,
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			fset := token.NewFileSet()
			dec := decorator.NewDecorator(fset)
			dstFile, err := dec.Parse(tt.src)
			g.Expect(err).NotTo(HaveOccurred())

			astFile := dec.Ast.Nodes[dstFile].(*ast.File)

			ratio := computeCommentRatio([]byte(tt.src), astFile.Comments, fset)
			g.Expect(ratio).To(BeNumerically("~", tt.want, 0.01))
		})
	}
}
