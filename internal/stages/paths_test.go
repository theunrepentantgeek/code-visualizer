package stages_test

import (
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestValidatePathsHelper_MissingTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := stages.ValidatePathsHelper("/no/such/path", "out.png")

	var tpe *stages.TargetPathError
	g.Expect(errors.As(err, &tpe)).To(BeTrue())
}

func TestValidatePathsHelper_BadOutputFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	err := stages.ValidatePathsHelper(dir, "out.unknown")

	var ope *stages.OutputPathError
	g.Expect(errors.As(err, &ope)).To(BeTrue())
}

func TestValidatePathsHelper_OK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")

	g.Expect(stages.ValidatePathsHelper(dir, out)).To(Succeed())
}

func TestValidatePaths_WrapsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{TargetPath: "/no/such/path", Output: "out.png"}
	err := stages.ValidatePaths(c)

	g.Expect(err).To(HaveOccurred())

	var tpe *stages.TargetPathError
	g.Expect(errors.As(err, &tpe)).To(BeTrue())
}
