package progress

import (
	"bytes"
	"io"
	"testing"

	. "github.com/onsi/gomega"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]

		return value, ok
	}
}

func TestParseMode_AcceptsSupportedValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for _, value := range []Mode{ModeAuto, ModeTTY, ModePlain} {
		parsed, err := ParseMode(string(value))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(parsed).To(Equal(value))
	}
}

func TestParseMode_RejectsUnsupportedValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := ParseMode("json")

	g.Expect(err).To(MatchError(ContainSubstring(`invalid progress mode "json"`)))
}

func TestResolveMode_AutoUsesTerminalCapability(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	writer := &bytes.Buffer{}

	resolved, err := resolveConfig(Config{
		Mode:       ModeAuto,
		Writer:     writer,
		IsTerminal: func(ioWriter io.Writer) bool { return ioWriter == writer },
		LookupEnv:  env(nil),
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.mode).To(Equal(ModeTTY))
}

func TestResolveMode_AutoUsesPlainForNonTerminalDumbTermAndTruthyCI(t *testing.T) {
	t.Parallel()

	cases := map[string]Config{
		"non-terminal": {
			IsTerminal: func(io.Writer) bool { return false },
			LookupEnv:  env(nil),
		},
		"dumb terminal": {
			IsTerminal: func(io.Writer) bool { return true },
			LookupEnv:  env(map[string]string{"TERM": "dumb"}),
		},
		"CI": {
			IsTerminal: func(io.Writer) bool { return true },
			LookupEnv:  env(map[string]string{"CI": "true"}),
		},
	}

	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			config.Mode = ModeAuto
			config.Writer = &bytes.Buffer{}

			resolved, err := resolveConfig(config)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resolved.mode).To(Equal(ModePlain))
		})
	}
}

func TestResolveMode_FalseCIValuesDoNotDisableTTY(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"false", "FALSE", "0"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			resolved, err := resolveConfig(Config{
				Mode:       ModeAuto,
				Writer:     &bytes.Buffer{},
				IsTerminal: func(io.Writer) bool { return true },
				LookupEnv:  env(map[string]string{"CI": value}),
			})

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resolved.mode).To(Equal(ModeTTY))
		})
	}
}

func TestResolveMode_ExplicitPlainRemainsPlainOnTerminal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolved, err := resolveConfig(Config{
		Mode:       ModePlain,
		Writer:     &bytes.Buffer{},
		IsTerminal: func(io.Writer) bool { return true },
		LookupEnv:  env(nil),
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.mode).To(Equal(ModePlain))
}

func TestResolveMode_ForcedTTYRejectsUnsupportedWriter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := resolveConfig(Config{
		Mode:       ModeTTY,
		Writer:     &bytes.Buffer{},
		IsTerminal: func(io.Writer) bool { return false },
		LookupEnv:  env(nil),
	})

	g.Expect(err).To(MatchError(ContainSubstring("does not support terminal progress")))
}

func TestResolveMode_ForcedTTYAllowsDumbTerminalWithoutColor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolved, err := resolveConfig(Config{
		Mode:       ModeTTY,
		Writer:     &bytes.Buffer{},
		IsTerminal: func(io.Writer) bool { return true },
		LookupEnv:  env(map[string]string{"TERM": "dumb"}),
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.mode).To(Equal(ModeTTY))
	g.Expect(resolved.noColor).To(BeTrue())
}

func TestResolveColor_HonorsEveryDisableInput(t *testing.T) {
	t.Parallel()

	cases := map[string]Config{
		"flag":        {NoColor: true, LookupEnv: env(nil)},
		"NO_COLOR":    {LookupEnv: env(map[string]string{"NO_COLOR": "1"})},
		"FORCE_COLOR": {LookupEnv: env(map[string]string{"FORCE_COLOR": "0"})},
		"dumb term":   {LookupEnv: env(map[string]string{"TERM": "dumb"})},
	}

	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			config.Mode = ModePlain
			config.Writer = &bytes.Buffer{}

			resolved, err := resolveConfig(config)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resolved.noColor).To(BeTrue())
		})
	}
}

func TestResolveColor_EmptyNoColorDoesNotDisableColor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolved, err := resolveConfig(Config{
		Mode:      ModePlain,
		Writer:    &bytes.Buffer{},
		LookupEnv: env(map[string]string{"NO_COLOR": ""}),
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.noColor).To(BeFalse())
}
