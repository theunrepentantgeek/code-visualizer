package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func TestRuleMapper_PopulatesFiltersDuringParseInCommandLineOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var cli struct {
		Include []filter.Rule `type:"filterrule" name:"include"`
		Exclude []filter.Rule `type:"filterrule" name:"exclude"`
	}

	parser, err := kong.New(
		&cli,
		kong.NamedMapper(ruleMapperName, ruleMapper{}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"--exclude", ".*",
		"--include", ".github/**",
		"--exclude", "**/*.log",
	})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(cli.Include).To(HaveLen(1))
	g.Expect(cli.Include[0].Pattern).To(Equal(".github/**"))
	g.Expect(cli.Include[0].Mode).To(Equal(filter.Include))

	g.Expect(cli.Exclude).To(HaveLen(2))
	g.Expect(cli.Exclude[0].Pattern).To(Equal(".*"))
	g.Expect(cli.Exclude[1].Pattern).To(Equal("**/*.log"))

	merged := filter.Merge(cli.Include, cli.Exclude)
	g.Expect(merged).To(HaveLen(3))
	g.Expect(merged[0].Pattern).To(Equal(".*"))
	g.Expect(merged[0].Mode).To(Equal(filter.Exclude))
	g.Expect(merged[1].Pattern).To(Equal(".github/**"))
	g.Expect(merged[1].Mode).To(Equal(filter.Include))
	g.Expect(merged[2].Pattern).To(Equal("**/*.log"))
	g.Expect(merged[2].Mode).To(Equal(filter.Exclude))
}
