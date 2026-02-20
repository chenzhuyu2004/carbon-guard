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
	runner := fs.String("runner", "ubuntu", "runner type (ubuntu/windows/macos)")
	region := fs.String("region", "global", "region carbon intensity")
	load := fs.Float64("load", 0.6, "CPU load factor (0-1)")
	asJSON := fs.Bool("json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "duration must be > 0")
		os.Exit(1)
	}

	if *load < 0 || *load > 1 {
		fmt.Fprintln(os.Stderr, "load must be between 0 and 1")
		os.Exit(1)
	}

	fmt.Print(report.Build(*duration, *asJSON, *runner, *region, *load))
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: carbon-guard run --duration <seconds> [--json]")
}
