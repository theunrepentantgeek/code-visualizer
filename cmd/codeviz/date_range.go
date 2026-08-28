package main

import (
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

var dateFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseDateRange(fromValue, untilValue string) (time.Time, time.Time, error) {
	from, err := parseDate(fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	until, err := parseDate(untilValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if !until.IsZero() && isDateOnly(untilValue) {
		until = until.Add(24*time.Hour - time.Nanosecond)
	}

	if !from.IsZero() && !until.IsZero() && from.After(until) {
		return time.Time{}, time.Time{}, eris.New("--from must be before or equal to --until")
	}

	return from, until, nil
}

func isDateOnly(value string) bool {
	if value == "" {
		return false
	}

	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	for _, layout := range dateFormats {
		if t, err := time.Parse(layout, value); err == nil {
			if layout == "2006-01-02" {
				return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
			}

			return t, nil
		}
	}

	return time.Time{}, eris.Errorf("invalid date %q: expected YYYY-MM-DD or RFC3339 timestamp", value)
}

func stagesFlagsForCommand(flags *Flags, fromValue, untilValue string) (*stages.Flags, error) {
	parsedFlags := toStagesFlags(flags)
	from, until, err := parseDateRange(fromValue, untilValue)
	if err != nil {
		return nil, err
	}
	parsedFlags.From = from
	parsedFlags.Until = until

	return parsedFlags, nil
}
