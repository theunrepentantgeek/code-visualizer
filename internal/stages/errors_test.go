package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestGitRequiredError_Error_ContainsMetricAndTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	e := &stages.GitRequiredError{Metric: "file-age", Target: "/some/path"}
	msg := e.Error()
	g.Expect(msg).To(ContainSubstring("file-age"))
	g.Expect(msg).To(ContainSubstring("/some/path"))
}

func TestTargetPathError_Error_ReturnsMsg(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	e := &stages.TargetPathError{Msg: "target does not exist"}
	g.Expect(e.Error()).To(Equal("target does not exist"))
}

func TestOutputPathError_Error_ReturnsMsg(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	e := &stages.OutputPathError{Msg: "cannot write to /dev/null/file.png"}
	g.Expect(e.Error()).To(Equal("cannot write to /dev/null/file.png"))
}

func TestNoFilesAfterFilterError_Error_ReturnsMsg(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	e := &stages.NoFilesAfterFilterError{Msg: stages.NoFilesAfterFilterMsg}
	g.Expect(e.Error()).To(Equal(stages.NoFilesAfterFilterMsg))
}
