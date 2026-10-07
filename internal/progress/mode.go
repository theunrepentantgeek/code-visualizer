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

func resolveConfig(config Config) (resolvedConfig, error) {
	mode, err := ParseMode(string(config.Mode))
	if err != nil {
		return resolvedConfig{}, err
	}

	lookupEnv := config.LookupEnv
	if lookupEnv == nil {
		lookupEnv = func(string) (string, bool) { return "", false }
	}

	term, _ := lookupEnv("TERM")

	mode, err = resolveMode(mode, config, term, lookupEnv)
	if err != nil {
		return resolvedConfig{}, err
	}

	noColor := config.NoColor ||
		term == "dumb" ||
		envNonEmpty(lookupEnv, "NO_COLOR") ||
		envEquals(lookupEnv, "FORCE_COLOR", "0")

	return resolvedConfig{Config: config, mode: mode, noColor: noColor}, nil
}

func resolveMode(
	mode Mode,
	config Config,
	term string,
	lookupEnv func(string) (string, bool),
) (Mode, error) {
	terminal := config.IsTerminal != nil && config.IsTerminal(config.Writer)

	switch mode {
	case ModeAuto:
		if !terminal || term == "dumb" || ciEnabled(lookupEnv) {
			return ModePlain, nil
		}

		return ModeTTY, nil
	case ModeTTY:
		if !terminal {
			return "", errors.New("configured stderr does not support terminal progress")
		}

		return ModeTTY, nil
	case ModePlain:
		return ModePlain, nil
	default:
		return "", errors.New("unreachable progress mode")
	}
}

func envNonEmpty(lookupEnv func(string) (string, bool), key string) bool {
	value, ok := lookupEnv(key)

	return ok && value != ""
}

func envEquals(lookupEnv func(string) (string, bool), key, expected string) bool {
	value, ok := lookupEnv(key)

	return ok && value == expected
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
