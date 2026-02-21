package cmd

import (
	"fmt"
	"os"

	"github.com/czy/carbon-guard/internal/output"
)

func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	asJSON := false
	for i, arg := range os.Args {
		if arg == "--json" || arg == "-json" || arg == "--output=json" || arg == "-output=json" {
			asJSON = true
		} else if (arg == "--output" || arg == "-output") && i+1 < len(os.Args) && os.Args[i+1] == "json" {
			asJSON = true
		}
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "suggest":
		err = suggest(os.Args[2:])
	case "run-aware":
		err = runAware(os.Args[2:])
	case "optimize":
		err = optimize(os.Args[2:])
	case "optimize-global":
		err = optimizeGlobal(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}

	output.HandleExit(err, asJSON)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: carbon-guard <run|suggest|run-aware|optimize|optimize-global> [flags]")
}
