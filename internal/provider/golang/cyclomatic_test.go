package golang

import (
	"go/token"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TestCyclomaticComplexity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		src  string
		want int64
	}{
		"empty function": {
			src:  `package p; func f() {}`,
			want: 1,
		},
		"single if": {
			src:  `package p; func f() { if true {} }`,
			want: 2,
		},
		"if with else": {
			src:  `package p; func f() { if true {} else {} }`,
			want: 2,
		},
		"for loop": {
			src:  `package p; func f() { for i := 0; i < 10; i++ {} }`,
			want: 2,
		},
		"range loop": {
			src:  `package p; func f() { for range []int{} {} }`,
			want: 2,
		},
		"switch with 2 cases": {
			src:  `package p; func f() { switch { case true: case false: } }`,
			want: 3,
		},
		"switch with default only": {
			src:  `package p; func f() { switch { default: } }`,
			want: 1,
		},
		"select with 2 cases": {
			src: `package p
import "time"
func f() {
	ch := make(chan int)
	select {
	case <-ch:
	case <-time.After(0):
	}
}`,
			want: 3,
		},
		"logical AND": {
			src:  `package p; func f() { var a, b bool; if a && b {} }`,
			want: 3,
		},
		"logical OR": {
			src:  `package p; func f() { var a, b bool; if a || b {} }`,
			want: 3,
		},
		"nested if-for": {
			src: `package p
func f() {
	for i := 0; i < 10; i++ {
		if i > 5 {}
	}
}`,
			want: 3,
		},
		"nil body": {
			src:  `package p; func f()`,
			want: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			dec := decorator.NewDecorator(token.NewFileSet())
			dstFile, err := dec.Parse(tt.src)
			g.Expect(err).NotTo(HaveOccurred())

			var funcDecl *dst.FuncDecl

			for _, decl := range dstFile.Decls {
				if fd, ok := decl.(*dst.FuncDecl); ok {
					funcDecl = fd

					break
				}
			}

			g.Expect(funcDecl).NotTo(BeNil(), "no func decl found in source")

			g.Expect(cyclomaticComplexity(funcDecl.Body)).To(Equal(tt.want))
		})
	}
}
