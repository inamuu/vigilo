package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/inamuu/vigilo/internal/event"
)

type Notifier struct {
	webhookURL string
	client     *http.Client
}

type payload struct {
	Text string `json:"text"`
}

func New(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *Notifier) Notify(ctx context.Context, evt event.Event) error {
	body, err := json.Marshal(payload{Text: formatMessage(evt)})
	if err != nil {
		return fmt.Errorf("encode slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("send slack notification: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(snippet)))
	}

	return nil
}

func formatMessage(evt event.Event) string {
	switch evt.Kind {
	case event.PatternMatched:
		return fmt.Sprintf("[vigilo] pattern matched\ncommand: %s\npattern: %s\nline: %s\ntime: %s",
			evt.Command,
			evt.Pattern,
			evt.Line,
			evt.Timestamp.Format(time.RFC3339),
		)
	case event.CommandFinished:
		return fmt.Sprintf("[vigilo] command finished\ncommand: %s\nexit_code: %d\ntime: %s",
			evt.Command,
			evt.ExitCode,
			evt.Timestamp.Format(time.RFC3339),
		)
	default:
		return "[vigilo] notification"
	}
}
