package metric

import (
	"regexp"
	"strings"

	"github.com/rotisserie/eris"
)

// MetricExpression is the parsed form of a user-provided metric string.
// Format: [filter.]base-metric[.aggregation].
type MetricExpression struct {
	Filter      FilterName
	Base        Name
	Aggregation AggregationName
}

// validSegment matches a kebab-case identifier: lowercase letters, digits, and hyphens.
var validSegment = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ParseExpression parses a metric expression string into its components.
// The grammar is: [filter.]base-name[.aggregation]
// where each component is a kebab-case word separated by dots.
func ParseExpression(input string) (MetricExpression, error) {
	if input == "" {
		return MetricExpression{}, eris.New("metric expression must not be empty")
	}

	segments := strings.Split(input, ".")

	if len(segments) > 3 {
		return MetricExpression{}, eris.Errorf(
			"metric expression %q has too many segments (max 3: filter.base.aggregation)", input,
		)
	}

	for _, seg := range segments {
		if !validSegment.MatchString(seg) {
			return MetricExpression{}, eris.Errorf(
				"metric expression %q contains invalid segment %q (must be lowercase kebab-case)", input, seg,
			)
		}
	}

	switch len(segments) {
	case 1:
		return MetricExpression{Base: Name(segments[0])}, nil
	case 2:
		return parseTwoSegments(segments)
	case 3:
		return parseThreeSegments(segments)
	default:
		return MetricExpression{}, eris.Errorf("metric expression %q is invalid", input)
	}
}

func parseTwoSegments(segments []string) (MetricExpression, error) {
	last := AggregationName(segments[1])
	if last.IsKnown() {
		return MetricExpression{
			Base:        Name(segments[0]),
			Aggregation: last,
		}, nil
	}

	return MetricExpression{
		Filter: FilterName(segments[0]),
		Base:   Name(segments[1]),
	}, nil
}

func parseThreeSegments(segments []string) (MetricExpression, error) {
	last := AggregationName(segments[2])
	if !last.IsKnown() {
		return MetricExpression{}, eris.Errorf(
			"metric expression segment %q is not a known aggregation function", segments[2],
		)
	}

	return MetricExpression{
		Filter:      FilterName(segments[0]),
		Base:        Name(segments[1]),
		Aggregation: last,
	}, nil
}

// String returns the canonical string form of the expression.
func (e MetricExpression) String() string {
	var b strings.Builder

	if !e.Filter.IsZero() {
		b.WriteString(string(e.Filter))
		b.WriteByte('.')
	}

	b.WriteString(string(e.Base))

	if !e.Aggregation.IsZero() {
		b.WriteByte('.')
		b.WriteString(string(e.Aggregation))
	}

	return b.String()
}

// ResultName returns the metric.Name used as the storage key in MetricContainer.
func (e MetricExpression) ResultName() Name {
	return Name(e.String())
}
