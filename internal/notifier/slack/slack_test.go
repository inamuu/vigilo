package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/inamuu/vigilo/internal/event"
)

func TestNotifySendsSlackPayload(t *testing.T) {
	var got payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := New(server.URL)
	evt := event.Event{
		Kind:      event.CommandFinished,
		Command:   "terraform apply",
		ExitCode:  1,
		Timestamp: time.Date(2026, time.March, 6, 11, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
	}

	if err := notifier.Notify(context.Background(), evt); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	want := "[vigilo] command finished\ncommand: terraform apply\nexit_code: 1\ntime: 2026-03-06T11:00:00+09:00"
	if got.Text != want {
		t.Fatalf("payload text = %q, want %q", got.Text, want)
	}
}
