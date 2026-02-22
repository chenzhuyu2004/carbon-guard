package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chenzhuyu2004/carbon-guard/internal/output"
)

func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]
	asJSON := detectJSONOutput(command, args)

	var err error
	switch command {
	case "run":
		err = run(args)
	case "suggest":
		err = suggest(args)
	case "run-aware":
		err = runAware(args)
	case "optimize":
		err = optimize(args)
	case "optimize-global":
		err = optimizeGlobal(args)
	default:
		printUsage()
		os.Exit(1)
	}

	output.HandleExit(err, asJSON)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: carbon-guard <run|suggest|run-aware|optimize|optimize-global> [flags]")
}

func detectJSONOutput(command string, args []string) bool {
	switch command {
	case "run":
		if enabled, ok := parseBoolFlag(args, "json"); ok {
			return enabled
		}
		return false
	case "optimize", "optimize-global":
		if mode, ok := parseStringFlag(args, "output"); ok {
			return strings.EqualFold(mode, "json")
		}
		return false
	default:
		return false
	}
}

func parseBoolFlag(args []string, name string) (bool, bool) {
	longName := "--" + name
	shortName := "-" + name

	for _, arg := range args {
		switch {
		case arg == longName || arg == shortName:
			return true, true
		case strings.HasPrefix(arg, longName+"="):
			value := strings.TrimPrefix(arg, longName+"=")
			parsed, err := strconv.ParseBool(value)
			if err == nil {
				return parsed, true
			}
		case strings.HasPrefix(arg, shortName+"="):
			value := strings.TrimPrefix(arg, shortName+"=")
			parsed, err := strconv.ParseBool(value)
			if err == nil {
				return parsed, true
			}
		}
	}

	return false, false
}

func parseStringFlag(args []string, name string) (string, bool) {
	longName := "--" + name
	shortName := "-" + name

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == longName || arg == shortName:
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		case strings.HasPrefix(arg, longName+"="):
			return strings.TrimPrefix(arg, longName+"="), true
		case strings.HasPrefix(arg, shortName+"="):
			return strings.TrimPrefix(arg, shortName+"="), true
		}
	}

	return "", false
}
