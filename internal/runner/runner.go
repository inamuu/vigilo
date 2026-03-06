package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/inamuu/vigilo/internal/cli"
	"github.com/inamuu/vigilo/internal/event"
	"github.com/inamuu/vigilo/internal/matcher"
	"github.com/inamuu/vigilo/internal/notifier"
)

type Runner struct {
	options  cli.Options
	notifier notifier.Notifier
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	matcher  *matcher.Matcher

	outputMu      sync.Mutex
	matchMu       sync.Mutex
	matchNotified bool
	exitTriggered bool
}

func New(options cli.Options, backend notifier.Notifier, stdin io.Reader, stdout io.Writer, stderr io.Writer) (*Runner, error) {
	var compiledMatcher *matcher.Matcher
	if options.Mode() == cli.PatternMonitoringMode {
		var err error
		compiledMatcher, err = matcher.Compile(options.Patterns)
		if err != nil {
			return nil, fmt.Errorf("compile pattern: %w", err)
		}
	}

	return &Runner{
		options:  options,
		notifier: backend,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		matcher:  compiledMatcher,
	}, nil
}

func (r *Runner) Run(ctx context.Context) (int, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, r.options.Command[0], r.options.Command[1:]...)
	cmd.Stdin = r.stdin

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("capture stdout: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("capture stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start command: %w", err)
	}

	streamErrs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		streamErrs <- r.monitorStream(runCtx, stdoutPipe, r.stdout, streamOptions{
			name:    "stdout",
			mirror:  !r.options.NoStdout,
			prefix:  "",
			onMatch: cancel,
			command: r.options.CommandString(),
		})
	}()

	go func() {
		defer wg.Done()
		prefix := ""
		if r.options.PrefixStderr {
			prefix = "stderr: "
		}

		streamErrs <- r.monitorStream(runCtx, stderrPipe, r.stderr, streamOptions{
			name:    "stderr",
			mirror:  true,
			prefix:  prefix,
			onMatch: cancel,
			command: r.options.CommandString(),
		})
	}()

	wg.Wait()
	close(streamErrs)

	var readErr error
	for err := range streamErrs {
		if err != nil && readErr == nil {
			readErr = err
		}
	}

	waitErr := cmd.Wait()

	if readErr != nil {
		return 0, readErr
	}

	exitCode, err := exitCode(waitErr)
	if err != nil {
		return 0, err
	}

	if r.options.Mode() == cli.ExitStatusMonitoringMode {
		r.notify(runCtx, event.Event{
			Kind:      event.CommandFinished,
			Command:   r.options.CommandString(),
			ExitCode:  exitCode,
			Timestamp: time.Now(),
		})
	}

	if r.options.ExitOnMatch && r.wasExitTriggered() {
		return 0, nil
	}

	return exitCode, nil
}

type streamOptions struct {
	name    string
	mirror  bool
	prefix  string
	onMatch context.CancelFunc
	command string
}

func (r *Runner) monitorStream(ctx context.Context, reader io.Reader, mirror io.Writer, options streamOptions) error {
	buffered := bufio.NewReader(reader)
	for {
		text, err := buffered.ReadString('\n')
		if len(text) > 0 {
			line := strings.TrimRight(text, "\r\n")
			if options.mirror {
				r.writeOutput(mirror, options.prefix, text)
			}

			if r.matcher != nil {
				if pattern, ok := r.matcher.Match(line); ok {
					r.handleMatch(ctx, options.command, pattern, line, options.onMatch)
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
				return nil
			}

			return fmt.Errorf("read %s: %w", options.name, err)
		}
	}
}

func (r *Runner) handleMatch(ctx context.Context, command string, pattern string, line string, cancel context.CancelFunc) {
	notify, shouldCancel := r.beginMatch()
	if !notify {
		return
	}

	r.writeDiagnostic(fmt.Sprintf("[vigilo] matched pattern %q\n", pattern))
	r.notify(ctx, event.Event{
		Kind:      event.PatternMatched,
		Command:   command,
		Pattern:   pattern,
		Line:      line,
		Timestamp: time.Now(),
	})

	if shouldCancel {
		cancel()
	}
}

func (r *Runner) beginMatch() (bool, bool) {
	r.matchMu.Lock()
	defer r.matchMu.Unlock()

	if r.options.ExitOnMatch && r.exitTriggered {
		return false, false
	}

	if r.options.Once && r.matchNotified {
		return false, false
	}

	if r.options.Once {
		r.matchNotified = true
	}

	shouldCancel := false
	if r.options.ExitOnMatch {
		r.exitTriggered = true
		shouldCancel = true
	}

	return true, shouldCancel
}

func (r *Runner) wasExitTriggered() bool {
	r.matchMu.Lock()
	defer r.matchMu.Unlock()
	return r.exitTriggered
}

func (r *Runner) notify(ctx context.Context, evt event.Event) {
	if err := r.notifier.Notify(ctx, evt); err != nil {
		r.writeDiagnostic(fmt.Sprintf("[vigilo] warning: notification failed: %v\n", err))
	}
}

func (r *Runner) writeOutput(writer io.Writer, prefix string, value string) {
	r.outputMu.Lock()
	defer r.outputMu.Unlock()

	if prefix == "" {
		_, _ = io.WriteString(writer, value)
		return
	}

	hasTrailingNewline := strings.HasSuffix(value, "\n")
	trimmed := strings.TrimSuffix(value, "\n")
	_, _ = io.WriteString(writer, prefix+trimmed)
	if hasTrailingNewline {
		_, _ = io.WriteString(writer, "\n")
	}
}

func (r *Runner) writeDiagnostic(message string) {
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	_, _ = io.WriteString(r.stderr, message)
}

func exitCode(waitErr error) (int, error) {
	if waitErr == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}

	return 0, fmt.Errorf("wait for command: %w", waitErr)
}
