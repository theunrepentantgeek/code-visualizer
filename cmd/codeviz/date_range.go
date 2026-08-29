package main

import (
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

var dateFormats = []string{
	"2006-01-02",
}

func parseDateRange(fromValue, untilValue string) (from, until time.Time, err error) {
	from, err = parseDate(fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	until, err = parseDate(untilValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if !until.IsZero() {
		until = time.Date(
			until.Year(),
			until.Month(),
			until.Day()+1,
			0,
			0,
			0,
			0,
			until.Location(),
		).Add(-time.Nanosecond)
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

	for _, layout := range dateFormats {
		//nolint:gosmopolitan // --from/--until are explicitly interpreted in local time.
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, eris.Errorf("invalid date %q: expected YYYY-MM-DD", value)
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
