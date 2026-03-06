package cli

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

type Mode string

const (
	PatternMonitoringMode    Mode = "pattern"
	ExitStatusMonitoringMode Mode = "exit-status"
)

type Options struct {
	Patterns     []string
	Notify       string
	ConfigPath   string
	ShowVersion  bool
	Once         bool
	ExitOnMatch  bool
	NoStdout     bool
	PrefixStderr bool
	Command      []string
}

func (o Options) Mode() Mode {
	if len(o.Patterns) > 0 {
		return PatternMonitoringMode
	}

	return ExitStatusMonitoringMode
}

func (o Options) CommandString() string {
	return strings.Join(o.Command, " ")
}

type UsageError struct {
	message string
}

func (e *UsageError) Error() string {
	return e.message
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func Parse(args []string) (Options, error) {
	var options Options
	var patterns multiFlag

	fs := flag.NewFlagSet("vigilo", flag.ContinueOnError)
	fs.SetOutput(new(strings.Builder))
	fs.Var(&patterns, "pattern", "regex pattern to detect (repeatable)")
	fs.Var(&patterns, "p", "regex pattern to detect (repeatable)")
	fs.StringVar(&options.Notify, "notify", "", "notification backend: osascript or slack")
	fs.StringVar(&options.ConfigPath, "config", "", "config file path")
	fs.BoolVar(&options.ShowVersion, "version", false, "print version and exit")
	fs.BoolVar(&options.Once, "once", false, "notify only on the first match")
	fs.BoolVar(&options.ExitOnMatch, "exit-on-match", false, "stop after the first match")
	fs.BoolVar(&options.NoStdout, "no-stdout", false, "do not mirror stdout to the terminal")
	fs.BoolVar(&options.PrefixStderr, "prefix-stderr", false, "prefix stderr lines when mirroring")

	if err := fs.Parse(args); err != nil {
		return Options{}, &UsageError{message: err.Error()}
	}

	options.Patterns = []string(patterns)
	options.Command = fs.Args()

	if options.ShowVersion {
		return options, nil
	}

	if options.Notify == "" {
		return Options{}, &UsageError{message: "missing required --notify flag"}
	}

	if len(options.Command) == 0 {
		return Options{}, &UsageError{message: "missing command to execute"}
	}

	if len(options.Patterns) == 0 {
		if options.Once {
			return Options{}, &UsageError{message: "--once requires at least one --pattern"}
		}

		if options.ExitOnMatch {
			return Options{}, &UsageError{message: "--exit-on-match requires at least one --pattern"}
		}
	}

	return options, nil
}

func Usage(program string) string {
	name := filepath.Base(program)
	return fmt.Sprintf(`Usage:
  %s [flags] -- <command> [args...]

Modes:
  Pattern monitoring mode activates when one or more --pattern flags are present.
  Exit status monitoring mode activates when no --pattern flag is present.

Flags:
  --pattern, -p     Regex pattern to detect (repeatable)
  --notify          Notification backend: osascript or slack
  --config          Explicit config file path
  --version         Print version and exit
  --once            Notify only once, then keep streaming output
  --exit-on-match   Exit after the first match
  --no-stdout       Do not mirror stdout to the terminal
  --prefix-stderr   Prefix stderr lines when mirroring

Examples:
  %s -p 'ERROR|FATAL' --notify osascript -- kubectl logs -f pod/app
  %s --notify slack -- terraform apply
`, name, name, name)
}
