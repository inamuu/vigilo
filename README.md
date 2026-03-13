# vigilo

`vigilo` is a CLI tool that runs an arbitrary command, watches its output, and sends a notification when something important happens.

It supports two monitoring modes:

- Pattern monitoring mode
  - Activated when one or more `--pattern` flags are provided
  - Watches stdout/stderr line by line and notifies on regex matches
- Exit status monitoring mode
  - Activated when no `--pattern` flag is provided
  - Waits for the command to finish and notifies with the exit code

## Install

### Homebrew

```bash
brew install inamuu/tap/vigilo
```

### Build from source

If you already have `go`:

```bash
go build -o vigilo ./cmd/vigilo
```

If you use `mise`:

```bash
mise exec go@1.24 -- go build -o vigilo ./cmd/vigilo
```

## Features

- Runs any command with arguments
- Streams stdout/stderr live to your terminal
- Matches one or more regular expressions
- Sends notifications with `osascript` or Slack Incoming Webhooks
- Returns the child process exit code

## Requirements

- Go 1.24+ to build from source
- macOS if you want to use the `osascript` notifier
- A Slack Incoming Webhook if you want to use the `slack` notifier

## Usage

```bash
vigilo [flags] -- <command> [args...]
```

Examples:

```bash
# Pattern monitoring mode
vigilo -p 'ERROR|FATAL|panic' --notify osascript -- kubectl logs -f deploy/api
vigilo -p 'Error:' -p 'Failed' --notify slack -- terraform apply

# Exit status monitoring mode
vigilo --notify osascript -- terraform apply
vigilo --notify slack -- ./deploy.sh
```

## Notification Backends

### `osascript`

Shows a native macOS notification.

Example:

```bash
vigilo -p 'panic|fatal' --notify osascript -- ./run-long-job.sh
```

### `slack`

Sends a message to a Slack Incoming Webhook URL loaded from config.

Config file lookup order:

1. `$XDG_CONFIG_HOME/vigilo/config.yaml`
2. `~/.config/vigilo/config.yaml`

Example config:

```yaml
slack:
  webhook_url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
```

Example:

```bash
vigilo -p 'ERROR|FATAL' --notify slack -- tail -f app.log
```

## Monitoring Modes

### Pattern Monitoring Mode

Enabled when `--pattern` or `-p` is specified.

Behavior:

- Compiles all provided regex patterns
- Reads stdout/stderr as line streams
- Sends a notification when a line matches
- Prints a diagnostic line such as `[vigilo] matched pattern "ERROR"` to stderr

Useful flags:

- `--once`: notify only on the first match, then keep streaming output
- `--exit-on-match`: stop after the first match
- `--no-stdout`: do not mirror stdout to the terminal
- `--prefix-stderr`: prefix mirrored stderr lines with `stderr: `

Examples:

```bash
vigilo -p 'ERROR' --notify osascript -- kubectl logs -f pod/app
vigilo -p 'timeout|connection refused|5[0-9]{2}' --notify slack -- ./deploy.sh
vigilo -p 'panic|fatal' --notify osascript --once -- ./run-long-job.sh
```

### Exit Status Monitoring Mode

Enabled when no `--pattern` flag is specified.

Behavior:

- Does not inspect stdout/stderr for regex matches
- Waits for the child process to finish
- Sends a notification with:
  - command
  - exit code
  - timestamp
- In the current implementation, notifies for both success and failure exits

Examples:

```bash
vigilo --notify osascript -- terraform apply
vigilo --notify slack -- ./deploy.sh
```

Example Slack message:

```text
[vigilo] command finished
command: terraform apply
exit_code: 1
time: 2026-03-06T11:00:00+09:00
```

## Common Options

- `--pattern`, `-p`: regex pattern to detect, repeatable
- `--notify`: notification backend, `osascript` or `slack`
- `--config`: explicit config file path
- `--once`: notify only once, only valid with `--pattern`
- `--exit-on-match`: exit after first match, only valid with `--pattern`
- `--no-stdout`: do not mirror stdout to the terminal
- `--prefix-stderr`: prefix mirrored stderr lines

## Exit Codes

- `vigilo` returns the child process exit code
- Invalid CLI usage returns a non-zero exit status
- Notification failures are reported to stderr, but do not crash the monitored command by themselves

## Release

Releases are built by GitHub Actions on tag push.

```bash
git tag v0.1.1
git push origin v0.1.1
```

The release workflow runs tests, publishes GitHub release assets with GoReleaser, and updates the Homebrew tap at `inamuu/homebrew-tap` when `HOMEBREW_TAP_GITHUB_TOKEN` is configured in GitHub Actions secrets.

## Development

Run tests:

```bash
mise exec go@1.24 -- go test ./...
```
