package runner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/inamuu/vigilo/internal/cli"
	"github.com/inamuu/vigilo/internal/event"
)

type recordingNotifier struct {
	mu     sync.Mutex
	events []event.Event
}

func (n *recordingNotifier) Notify(_ context.Context, evt event.Event) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, evt)
	return nil
}

func (n *recordingNotifier) Events() []event.Event {
	n.mu.Lock()
	defer n.mu.Unlock()

	cloned := make([]event.Event, len(n.events))
	copy(cloned, n.events)
	return cloned
}

func TestRunnerPatternModeNotifiesOnMatch(t *testing.T) {
	backend := &recordingNotifier{}
	options := cli.Options{
		Patterns: []string{"ERROR"},
		Notify:   "slack",
		Command:  []string{"/bin/sh", "-c", "printf 'INFO\\nERROR\\n'"},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app, err := New(options, backend, bytes.NewReader(nil), &stdout, &stderr)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	exitCode, err := app.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}

	events := backend.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}

	if events[0].Kind != event.PatternMatched {
		t.Fatalf("event kind = %q, want %q", events[0].Kind, event.PatternMatched)
	}

	if events[0].Pattern != "ERROR" {
		t.Fatalf("event pattern = %q, want %q", events[0].Pattern, "ERROR")
	}

	if events[0].Line != "ERROR" {
		t.Fatalf("event line = %q, want %q", events[0].Line, "ERROR")
	}

	if got := stdout.String(); got != "INFO\nERROR\n" {
		t.Fatalf("stdout = %q, want %q", got, "INFO\nERROR\n")
	}

	if got := stderr.String(); got != "[vigilo] matched pattern \"ERROR\"\n" {
		t.Fatalf("stderr = %q, want %q", got, "[vigilo] matched pattern \"ERROR\"\n")
	}
}

func TestRunnerOnceNotifiesOnlyFirstMatch(t *testing.T) {
	backend := &recordingNotifier{}
	options := cli.Options{
		Patterns: []string{"ERROR"},
		Notify:   "slack",
		Once:     true,
		Command:  []string{"/bin/sh", "-c", "printf 'ERROR\\nERROR\\n'"},
	}

	app, err := New(options, backend, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	exitCode, err := app.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}

	if len(backend.Events()) != 1 {
		t.Fatalf("event count = %d, want 1", len(backend.Events()))
	}
}

func TestRunnerExitStatusModeSendsExitNotification(t *testing.T) {
	backend := &recordingNotifier{}
	options := cli.Options{
		Notify:       "slack",
		PrefixStderr: true,
		Command:      []string{"/bin/sh", "-c", "printf 'ERROR\\n' >&2; exit 7"},
	}

	var stderr bytes.Buffer
	app, err := New(options, backend, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	exitCode, err := app.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if exitCode != 7 {
		t.Fatalf("Run() exit code = %d, want 7", exitCode)
	}

	events := backend.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}

	if events[0].Kind != event.CommandFinished {
		t.Fatalf("event kind = %q, want %q", events[0].Kind, event.CommandFinished)
	}

	if events[0].ExitCode != 7 {
		t.Fatalf("event exit code = %d, want 7", events[0].ExitCode)
	}

	if events[0].Pattern != "" || events[0].Line != "" {
		t.Fatalf("exit notification unexpectedly carried match data: pattern=%q line=%q", events[0].Pattern, events[0].Line)
	}

	if got := stderr.String(); got != "stderr: ERROR\n" {
		t.Fatalf("stderr = %q, want %q", got, "stderr: ERROR\n")
	}
}

func TestRunnerPatternModeDoesNotFailWhenUnusedStderrCloses(t *testing.T) {
	backend := &recordingNotifier{}
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputPath, []byte("test\nerror\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	options := cli.Options{
		Patterns: []string{"err"},
		Notify:   "slack",
		Command:  []string{"cat", inputPath},
	}

	app, err := New(options, backend, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	exitCode, err := app.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}

	if len(backend.Events()) != 1 {
		t.Fatalf("event count = %d, want 1", len(backend.Events()))
	}
}
