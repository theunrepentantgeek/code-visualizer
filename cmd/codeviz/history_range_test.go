package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

func TestParseHistoryRange_PreservesRawReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	historyRange := parseHistoryRange("sha:abc1234", "tag:v2.0")
	g.Expect(historyRange).To(Equal(git.HistoryRange{
		From:  "sha:abc1234",
		Until: "tag:v2.0",
	}))
}

func TestStagesFlagsForCommand_PreservesRawReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := stagesFlagsForCommand(&Flags{}, "20260905", "date:2026-10-01")
	g.Expect(flags.HistoryRange).To(Equal(git.HistoryRange{
		From:  "20260905",
		Until: "date:2026-10-01",
	}))
}
