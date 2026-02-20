package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/czy/carbon-guard/internal/report"
)

func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func run(args []string) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	duration := fs.Int("duration", 0, "duration in seconds")
	asJSON := fs.Bool("json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "duration must be > 0")
		os.Exit(1)
	}

	fmt.Print(report.Build(*duration, *asJSON))
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: carbon-guard run --duration <seconds> [--json]")
}
