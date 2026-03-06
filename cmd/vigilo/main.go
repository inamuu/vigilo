package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/inamuu/vigilo/internal/cli"
	"github.com/inamuu/vigilo/internal/notifier"
	"github.com/inamuu/vigilo/internal/runner"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	options, err := cli.Parse(os.Args[1:])
	if err != nil {
		var usageErr *cli.UsageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr, usageErr.Error())
			fmt.Fprint(os.Stderr, cli.Usage(os.Args[0]))
			os.Exit(2)
		}

		fmt.Fprintf(os.Stderr, "vigilo: %v\n", err)
		os.Exit(1)
	}

	if options.ShowVersion {
		fmt.Printf("vigilo version %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	backend, err := notifier.New(options.Notify, options.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vigilo: %v\n", err)
		os.Exit(1)
	}

	app, err := runner.New(options, backend, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vigilo: %v\n", err)
		os.Exit(1)
	}

	exitCode, err := app.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "vigilo: %v\n", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}
