package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/inamuu/vigilo/internal/config"
	"github.com/inamuu/vigilo/internal/event"
	osascriptbackend "github.com/inamuu/vigilo/internal/notifier/osascript"
	slackbackend "github.com/inamuu/vigilo/internal/notifier/slack"
)

type Notifier interface {
	Notify(ctx context.Context, evt event.Event) error
}

type Method string

const (
	MethodOSAScript Method = "osascript"
	MethodSlack     Method = "slack"
)

func ParseMethod(value string) (Method, error) {
	switch Method(strings.ToLower(strings.TrimSpace(value))) {
	case MethodOSAScript:
		return MethodOSAScript, nil
	case MethodSlack:
		return MethodSlack, nil
	default:
		return "", fmt.Errorf("unsupported notify backend %q", value)
	}
}

func New(rawMethod string, configPath string) (Notifier, error) {
	method, err := ParseMethod(rawMethod)
	if err != nil {
		return nil, err
	}

	switch method {
	case MethodOSAScript:
		return osascriptbackend.New()
	case MethodSlack:
		cfg, resolvedPath, err := config.Load(configPath)
		if err != nil {
			return nil, err
		}

		if cfg.Slack.WebhookURL == "" {
			if configPath != "" {
				return nil, fmt.Errorf("slack webhook URL is not configured in %q", configPath)
			}

			if resolvedPath == "" {
				return nil, fmt.Errorf("slack webhook URL is not configured; expected config at $XDG_CONFIG_HOME/vigilo/config.yaml or ~/.config/vigilo/config.yaml")
			}

			return nil, fmt.Errorf("slack webhook URL is not configured in %q", resolvedPath)
		}

		return slackbackend.New(cfg.Slack.WebhookURL), nil
	default:
		return nil, fmt.Errorf("unsupported notify backend %q", rawMethod)
	}
}
