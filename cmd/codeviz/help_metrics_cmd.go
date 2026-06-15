package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// HelpMetricsCmd prints help for the supported metric expressions.
type HelpMetricsCmd struct{}

const (
	filesystemMetricsSection = "Filesystem metrics"
	gitMetricsSection        = "Git metrics"
	goMetricsSection         = "Go metrics"
	otherMetricsSection      = "Other metrics"
)

type providerSection struct {
	Name  string
	Title string
}

type metricHelpEntry struct {
	Name         string
	Kind         string
	Description  string
	Aggregations []string
	Filters      []string
}

var providerSectionOrder = []providerSection{
	{Name: "filesystem", Title: filesystemMetricsSection},
	{Name: "git", Title: gitMetricsSection},
	{Name: "go", Title: goMetricsSection},
}

//nolint:unparam // nil error required to satisfy the interface for Kong
func (HelpMetricsCmd) Run(_ *Flags) error {
	fmt.Print(renderHelpMetrics())

	return nil
}

func renderHelpMetrics() string {
	width := consoleWidth()
	baseSections := buildBaseSections(provider.AllBase())
	legacyMetrics := findLegacyMetrics(provider.AllDescriptors())

	content := &strings.Builder{}
	writeWrappedText(content, "Syntax: ", "[filter.]metric[.aggregation]", width)
	writeWrappedText(content, "Examples: ", "file-size.sum, public.types.count, cyclomatic-complexity.max", width)

	for _, section := range providerSectionOrder {
		metrics := baseSections[section.Name]
		if len(metrics) == 0 {
			continue
		}

		content.WriteString("\n")
		writeSectionHeader(content, section.Title)

		providerDescriptor, _ := provider.GetBaseProvider(metrics[0].Name)
		writeProviderFilters(content, providerDescriptor, metrics, width)
		writeBaseMetrics(content, metrics, width)
	}

	if len(legacyMetrics) > 0 {
		content.WriteString("\n")
		writeSectionHeader(content, otherMetricsSection)
		writeLegacyMetrics(content, legacyMetrics, width)
	}

	return content.String()
}

func buildBaseSections(descriptors []provider.BaseMetricDescriptor) map[string][]provider.BaseMetricDescriptor {
	sections := make(map[string][]provider.BaseMetricDescriptor, len(providerSectionOrder))

	for _, desc := range descriptors {
		pd, ok := provider.GetBaseProvider(desc.Name)
		if !ok {
			continue
		}

		sections[pd.Name] = append(sections[pd.Name], desc)
	}

	return sections
}

func findLegacyMetrics(descriptors []provider.MetricDescriptor) []provider.MetricDescriptor {
	baseNames := make(map[metric.Name]struct{}, len(provider.AllBase()))

	for _, desc := range provider.AllBase() {
		baseNames[desc.Name] = struct{}{}
	}

	legacy := make([]provider.MetricDescriptor, 0, len(descriptors))
	seen := make(map[metric.Name]struct{})

	for _, desc := range descriptors {
		if _, ok := baseNames[desc.Name]; ok {
			continue
		}

		if _, ok := seen[desc.Name]; ok {
			continue
		}

		legacy = append(legacy, desc)
		seen[desc.Name] = struct{}{}
	}

	slices.SortFunc(legacy, func(left, right provider.MetricDescriptor) int {
		return cmp.Compare(left.Name, right.Name)
	})

	return legacy
}

func writeSectionHeader(content *strings.Builder, title string) {
	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", utf8.RuneCountInString(title)))
	content.WriteString("\n\n")
}

func writeProviderFilters(
	content *strings.Builder,
	pd provider.ProviderDescriptor,
	metrics []provider.BaseMetricDescriptor,
	width int,
) {
	if len(pd.Filters) == 0 {
		return
	}

	filterNames := orderedProviderFilters(pd, metrics)
	filterDescriptions := make([]string, 0, len(filterNames))

	for _, name := range filterNames {
		description, ok := pd.Filters[name]
		if !ok {
			continue
		}

		filterDescriptions = append(filterDescriptions, fmt.Sprintf("%s (%s)", name, description))
	}

	if len(filterDescriptions) == 0 {
		return
	}

	writeWrappedLabelLine(content, "  ", "Filters: ", strings.Join(filterDescriptions, ", "), width)
	content.WriteString("\n")
}

func orderedProviderFilters(
	pd provider.ProviderDescriptor,
	metrics []provider.BaseMetricDescriptor,
) []metric.FilterName {
	seen := make(map[metric.FilterName]struct{}, len(pd.Filters))
	order := collectUsedProviderFilters(pd, metrics, seen)
	remaining := remainingProviderFilters(pd, seen, len(pd.Filters)-len(order))

	slices.SortFunc(remaining, cmp.Compare)

	return append(order, remaining...)
}

func collectUsedProviderFilters(
	pd provider.ProviderDescriptor,
	metrics []provider.BaseMetricDescriptor,
	seen map[metric.FilterName]struct{},
) []metric.FilterName {
	order := make([]metric.FilterName, 0, len(pd.Filters))

	for _, desc := range metrics {
		for _, filter := range desc.Filters {
			if _, ok := pd.Filters[filter]; !ok {
				continue
			}

			if _, ok := seen[filter]; ok {
				continue
			}

			order = append(order, filter)
			seen[filter] = struct{}{}
		}
	}

	return order
}

func remainingProviderFilters(
	pd provider.ProviderDescriptor,
	seen map[metric.FilterName]struct{},
	capHint int,
) []metric.FilterName {
	remaining := make([]metric.FilterName, 0, capHint)

	for filter := range pd.Filters {
		if _, ok := seen[filter]; ok {
			continue
		}

		remaining = append(remaining, filter)
	}

	return remaining
}

func writeBaseMetrics(content *strings.Builder, metrics []provider.BaseMetricDescriptor, width int) {
	nameWidth, kindWidth := metricColumnWidths(metrics)

	for _, desc := range metrics {
		writeMetricBlock(
			content,
			nameWidth,
			kindWidth,
			metricHelpEntry{
				Name:         string(desc.Name),
				Kind:         kindLabel(desc.Kind),
				Description:  desc.Description,
				Aggregations: aggregationLabels(desc.Aggregations),
				Filters:      filterLabels(desc.Filters),
			},
			width,
		)
	}
}

func writeLegacyMetrics(content *strings.Builder, metrics []provider.MetricDescriptor, width int) {
	nameWidth, kindWidth := legacyMetricColumnWidths(metrics)

	for _, desc := range metrics {
		writeMetricBlock(
			content,
			nameWidth,
			kindWidth,
			metricHelpEntry{
				Name:        string(desc.Name),
				Kind:        kindLabel(desc.Kind),
				Description: desc.Description,
			},
			width,
		)
	}
}

func metricColumnWidths(metrics []provider.BaseMetricDescriptor) (nameWidth int, kindWidth int) {
	for _, desc := range metrics {
		nameWidth = max(nameWidth, utf8.RuneCountInString(string(desc.Name)))
		kindWidth = max(kindWidth, utf8.RuneCountInString(kindLabel(desc.Kind)))
	}

	return nameWidth, kindWidth
}

func legacyMetricColumnWidths(metrics []provider.MetricDescriptor) (nameWidth int, kindWidth int) {
	for _, desc := range metrics {
		nameWidth = max(nameWidth, utf8.RuneCountInString(string(desc.Name)))
		kindWidth = max(kindWidth, utf8.RuneCountInString(kindLabel(desc.Kind)))
	}

	return nameWidth, kindWidth
}

func writeMetricBlock(
	content *strings.Builder,
	nameWidth int,
	kindWidth int,
	entry metricHelpEntry,
	width int,
) {
	prefix := fmt.Sprintf("  %-*s  %-*s  ", nameWidth, entry.Name, kindWidth, entry.Kind)
	indent := strings.Repeat(" ", 2+nameWidth+2+kindWidth+2)

	if usesCompactMetricLayout(prefix, indent, width) {
		writeCompactMetricBlock(content, entry, width)

		return
	}

	writeWrappedText(content, prefix, entry.Description, width)

	if len(entry.Aggregations) > 0 {
		writeWrappedLabelLine(content, indent, "Aggregations: ", strings.Join(entry.Aggregations, ", "), width)
	}

	if len(entry.Filters) > 0 {
		writeWrappedLabelLine(content, indent, "Filters: ", strings.Join(entry.Filters, ", "), width)
	}

	content.WriteString("\n")
}

func usesCompactMetricLayout(prefix string, indent string, width int) bool {
	return utf8.RuneCountInString(prefix) >= width ||
		utf8.RuneCountInString(indent+"Aggregations: ") >= width ||
		utf8.RuneCountInString(indent+"Filters: ") >= width
}

func writeCompactMetricBlock(content *strings.Builder, entry metricHelpEntry, width int) {
	nameLine := entry.Name
	if entry.Kind != "" {
		nameLine += "  " + entry.Kind
	}

	writeWrappedText(content, "  ", nameLine, width)
	writeWrappedText(content, "    ", entry.Description, width)

	if len(entry.Aggregations) > 0 {
		writeWrappedLabelLine(content, "    ", "Aggregations: ", strings.Join(entry.Aggregations, ", "), width)
	}

	if len(entry.Filters) > 0 {
		writeWrappedLabelLine(content, "    ", "Filters: ", strings.Join(entry.Filters, ", "), width)
	}

	content.WriteString("\n")
}

func writeWrappedLabelLine(content *strings.Builder, indent string, label string, value string, width int) {
	prefix := indent + label
	if width-utf8.RuneCountInString(prefix) >= longestWordWidth(value) {
		writeWrappedText(content, prefix, value, width)

		return
	}

	writeWrappedText(content, "    "+label, value, width)
}

func writeWrappedText(content *strings.Builder, prefix string, text string, width int) {
	prefixWidth := utf8.RuneCountInString(prefix)
	if prefixWidth >= width {
		content.WriteString(strings.TrimRight(prefix, " "))
		content.WriteString("\n")

		for _, line := range wrapText(text, max(width-2, 1)) {
			fmt.Fprintf(content, "  %s\n", line)
		}

		return
	}

	lines := wrapText(text, max(width-prefixWidth, 1))
	fmt.Fprintf(content, "%s%s\n", prefix, lines[0])

	if len(lines) == 1 {
		return
	}

	indent := strings.Repeat(" ", utf8.RuneCountInString(prefix))
	for _, line := range lines[1:] {
		fmt.Fprintf(content, "%s%s\n", indent, line)
	}
}

func wrapText(text string, width int) []string {
	if text == "" {
		return []string{""}
	}

	words := tokenizeWrappedWords(text, width)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, len(words))
	current := words[0]

	for _, word := range words[1:] {
		candidate := current + " " + word
		if utf8.RuneCountInString(candidate) > width {
			lines = append(lines, current)
			current = word

			continue
		}

		current = candidate
	}

	lines = append(lines, current)

	return lines
}

func tokenizeWrappedWords(text string, width int) []string {
	words := make([]string, 0)
	for word := range strings.FieldsSeq(text) {
		words = append(words, splitLongWord(word, width)...)
	}

	return words
}

func splitLongWord(word string, width int) []string {
	if utf8.RuneCountInString(word) <= width {
		return []string{word}
	}

	runes := []rune(word)
	parts := make([]string, 0, (len(runes)+width-1)/width)

	for start := 0; start < len(runes); start += width {
		end := min(start+width, len(runes))
		parts = append(parts, string(runes[start:end]))
	}

	return parts
}

func longestWordWidth(text string) int {
	longest := 0
	for word := range strings.FieldsSeq(text) {
		longest = max(longest, utf8.RuneCountInString(word))
	}

	return longest
}

func aggregationLabels(names []metric.AggregationName) []string {
	labels := make([]string, 0, len(names))
	for _, name := range names {
		labels = append(labels, string(name))
	}

	return labels
}

func filterLabels(names []metric.FilterName) []string {
	labels := make([]string, 0, len(names))
	for _, name := range names {
		labels = append(labels, string(name))
	}

	return labels
}

func kindLabel(k metric.Kind) string {
	switch k {
	case metric.Quantity:
		return "quantity"
	case metric.Measure:
		return "measure"
	case metric.Classification:
		return "category"
	default:
		return "unknown"
	}
}

// consoleWidth returns the width of the terminal, falling back to 120.
func consoleWidth() int {
	const defaultWidth = 120

	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w, err := strconv.Atoi(cols); err == nil && w > 0 {
			return w
		}
	}

	return defaultWidth
}
