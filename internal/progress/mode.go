package progress

import (
	"errors"
	"fmt"
	"strings"
)

type resolvedConfig struct {
	Config
	mode    Mode
	noColor bool
}

func ParseMode(value string) (Mode, error) {
	mode := Mode(value)
	if mode == "" {
		mode = ModeAuto
	}

	switch mode {
	case ModeAuto, ModeTTY, ModePlain:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid progress mode %q: expected auto, tty, or plain", value)
	}
}

//nolint:cyclop,revive // Mode, terminal, CI, and colour resolution are one cohesive decision.
func resolveConfig(config Config) (resolvedConfig, error) {
	mode, err := ParseMode(string(config.Mode))
	if err != nil {
		return resolvedConfig{}, err
	}

	lookupEnv := config.LookupEnv
	if lookupEnv == nil {
		lookupEnv = func(string) (string, bool) { return "", false }
	}

	terminal := config.IsTerminal != nil && config.IsTerminal(config.Writer)
	term, _ := lookupEnv("TERM")
	capable := terminal && term != "dumb"

	switch mode {
	case ModeAuto:
		if !capable || ciEnabled(lookupEnv) {
			mode = ModePlain
		} else {
			mode = ModeTTY
		}
	case ModeTTY:
		if !capable {
			return resolvedConfig{}, errors.New("configured stderr does not support terminal progress")
		}
	case ModePlain:
	default:
		return resolvedConfig{}, errors.New("unreachable progress mode")
	}

	noColor := config.NoColor || term == "dumb"
	if value, ok := lookupEnv("NO_COLOR"); ok && value != "" {
		noColor = true
	}

	if value, ok := lookupEnv("FORCE_COLOR"); ok && value == "0" {
		noColor = true
	}

	return resolvedConfig{Config: config, mode: mode, noColor: noColor}, nil
}

func ciEnabled(lookupEnv func(string) (string, bool)) bool {
	value, ok := lookupEnv("CI")
	if !ok || value == "" {
		return false
	}

	switch strings.ToLower(value) {
	case "false", "0":
		return false
	default:
		return true
	}
}
