package main

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func parseHistoryRange(fromValue, untilValue string) git.HistoryRange {
	return git.HistoryRange{
		From:  fromValue,
		Until: untilValue,
	}
}

func stagesFlagsForCommand(flags *Flags, fromValue, untilValue string) *stages.Flags {
	parsedFlags := toStagesFlags(flags)
	parsedFlags.HistoryRange = parseHistoryRange(fromValue, untilValue)

	return parsedFlags
}
