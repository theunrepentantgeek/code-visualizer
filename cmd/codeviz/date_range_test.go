package main

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestParseDateRange_ParsesDateOnlyRangeInUTC(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	from, until, err := parseDateRange("2024-01-02", "2024-03-04")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(from.Location()).To(BeIdenticalTo(time.UTC))
	g.Expect(until.Location()).To(BeIdenticalTo(time.UTC))
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

func TestParseDateRange_IncludesLastDayOfMonth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, until, err := parseDateRange("", "2024-01-31")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(until).To(Equal(time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)))
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

func TestParseHistoryRange_AllowsMixedOppositeBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	historyRange, err := parseHistoryRange("2025-01-01", "", "", "v2.0")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(historyRange.From.IsZero()).To(BeFalse())
	g.Expect(historyRange.UntilTag).To(Equal("v2.0"))
}

func TestParseHistoryRange_RejectsSameSideDateAndTag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := parseHistoryRange("2025-01-01", "", "v1.0", "")
	g.Expect(err).To(MatchError("--from and --from-tag are mutually exclusive"))

	_, err = parseHistoryRange("", "2025-12-31", "", "v2.0")
	g.Expect(err).To(MatchError("--until and --until-tag are mutually exclusive"))
}
