package main

import (
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const dateFormat = "2006-01-02"

func parseDateRange(fromValue string, untilValue string) (from, until time.Time, err error) {
	from, err = parseDate(fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	until, err = parseDate(untilValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if !until.IsZero() {
		until = until.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}

	if !from.IsZero() && !until.IsZero() && from.After(until) {
		return time.Time{}, time.Time{}, eris.New("--from must be before or equal to --until")
	}

	return from, until, nil
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(dateFormat, value)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, eris.Errorf("invalid date %q: expected YYYY-MM-DD", value)
}

func parseHistoryRange(fromValue, untilValue, fromTag, untilTag string) (git.HistoryRange, error) {
	if fromValue != "" && fromTag != "" {
		return git.HistoryRange{}, eris.New("--from and --from-tag are mutually exclusive")
	}

	if untilValue != "" && untilTag != "" {
		return git.HistoryRange{}, eris.New("--until and --until-tag are mutually exclusive")
	}

	from, until, err := parseDateRange(fromValue, untilValue)
	if err != nil {
		return git.HistoryRange{}, err
	}

	return git.HistoryRange{
		From:     from,
		Until:    until,
		FromTag:  fromTag,
		UntilTag: untilTag,
	}, nil
}

func stagesFlagsForCommand(
	flags *Flags,
	fromValue, untilValue, fromTag, untilTag string,
) (*stages.Flags, error) {
	parsedFlags := toStagesFlags(flags)

	historyRange, err := parseHistoryRange(fromValue, untilValue, fromTag, untilTag)
	if err != nil {
		return nil, err
	}

	parsedFlags.HistoryRange = historyRange

	return parsedFlags, nil
}
