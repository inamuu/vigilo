package osascript

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/inamuu/vigilo/internal/event"
)

const title = "vigilo"
const backendName = "osascript"

type Notifier struct{}

func New() (*Notifier, error) {
	if _, err := exec.LookPath("osascript"); err != nil {
		return nil, fmt.Errorf("notify backend %q requires the osascript command", backendName)
	}

	return &Notifier{}, nil
}

func (n *Notifier) Notify(ctx context.Context, evt event.Event) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, escape(formatBody(evt)), title)
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run osascript notifier: %w", err)
	}

	return nil
}

func formatBody(evt event.Event) string {
	switch evt.Kind {
	case event.PatternMatched:
		body := fmt.Sprintf(`Matched "%s": %s (command: %s)`, evt.Pattern, evt.Line, evt.Command)
		return truncate(body, 220)
	case event.CommandFinished:
		body := fmt.Sprintf("Command finished (exit %d): %s at %s", evt.ExitCode, evt.Command, evt.Timestamp.Format("2006-01-02T15:04:05-07:00"))
		return truncate(body, 220)
	default:
		return "Notification"
	}
}

func escape(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}

func truncate(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}

	return string(runes[:maxRunes-3]) + "..."
}
