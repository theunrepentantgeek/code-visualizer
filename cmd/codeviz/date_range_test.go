package main

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestParseDateRange_ParsesDateOnlyRangeInLocalTimezone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	from, until, err := parseDateRange("2024-01-02", "2024-03-04")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(from.Hour()).To(Equal(0))
	g.Expect(from.Minute()).To(Equal(0))
	g.Expect(from.Second()).To(Equal(0))
	g.Expect(from.Nanosecond()).To(Equal(0))
	g.Expect(from.Year()).To(Equal(2024))
	g.Expect(from.Month()).To(Equal(time.January))
	g.Expect(from.Day()).To(Equal(2))

	expectedUntil := time.Date(2024, 3, 5, 0, 0, 0, 0, until.Location()).Add(-time.Nanosecond)
	g.Expect(until).To(Equal(expectedUntil))
}

func TestParseDateRange_RejectsReversedRange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, _, err := parseDateRange("2024-03-04", "2024-01-02")
	g.Expect(err).To(MatchError(ContainSubstring("--from must be before or equal to --until")))
}

func TestParseDate_RejectsTimestampsAndRequiresDateOnlyFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := parseDate("2024-01-02T15:04:05")
	g.Expect(err).To(MatchError(ContainSubstring("expected YYYY-MM-DD")))
}

func TestParseDate_AllowsEmptyValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	parsed, err := parseDate("")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parsed.IsZero()).To(BeTrue())
}
