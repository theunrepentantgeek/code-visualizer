package stages_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// fakeState is the minimal VizState used by stage tests in this package.
type fakeState struct {
	common stages.CommonState
}

func (f *fakeState) Common() *stages.CommonState { return &f.common }

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

func TestValidatePaths_Stage_WrapsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{TargetPath: "/nope", Output: "out.png"}}
	err := stages.ValidatePaths[*fakeState](s)

	g.Expect(err).To(HaveOccurred())

	var tpe *stages.TargetPathError

	g.Expect(errors.As(err, &tpe)).To(BeTrue())
}

func TestValidatePaths_Stage_OK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")
	// also ensure parent dir exists
	g.Expect(os.MkdirAll(filepath.Dir(out), 0o755)).To(Succeed())

	s := &fakeState{common: stages.CommonState{TargetPath: dir, Output: out}}
	g.Expect(stages.ValidatePaths[*fakeState](s)).To(Succeed())
}
