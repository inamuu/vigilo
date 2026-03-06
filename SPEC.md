# vigilo

Terminal output watcher CLI.

## Overview
`vigilo` is a CLI tool that runs an arbitrary command, watches its stdout/stderr in real time, detects lines matching configured patterns, and sends notifications.

It is intended for SRE and DevOps workflows such as:

- `kubectl logs -f`
- `terraform apply`
- `ansible-playbook`
- long-running shell scripts
- build/test commands

The goal is to avoid wrapping every command with `tee | grep ...` manually, and instead provide a reusable tool that can execute the command, inspect output, and notify when something important appears.

vigilo has two monitoring modes:

- Pattern monitoring mode
  - activated when one or more `--pattern` flags are provided
  - monitors stdout/stderr for regex matches
- Exit status monitoring mode
  - activated when no `--pattern` flag is provided
  - waits for the command to finish and notifies with the exit code

---

## Primary Use Cases

### 1. Log monitoring
Detect error patterns while watching logs.

Example:

```bash
vigilo --pattern 'ERROR|FATAL|panic' --notify osascript -- kubectl logs -f deploy/api

2. Terraform / deployment monitoring

Notify when a known failure string appears.

Example:

vigilo --pattern 'Error:|Failed|panic' --notify osascript -- terraform apply

3. Slack alert forwarding

Send matches to a Slack channel using Incoming Webhook.

Example:

vigilo --pattern 'ERROR|FATAL' --notify slack -- tail -f app.log

4. Multiple patterns for noisy commands

Detect only meaningful lines from verbose output.

Example:

vigilo --pattern 'timed out|connection refused|5[0-9]{2}' --notify slack -- ./deploy.sh


⸻

Non-Goals
	•	Not a full log aggregation platform
	•	Not a terminal emulator
	•	Not a replacement for observability systems like Datadog / CloudWatch / Prometheus
	•	Not intended to persist all logs by default
	•	Not intended to parse structured logs deeply in the first version

⸻

Functional Requirements

Command execution
	•	Execute an arbitrary command with arguments
	•	Stream stdout and stderr in real time
	•	Preserve live output to the terminal while monitoring
	•	Return the child process exit code

Pattern matching
	•	Accept one or more regex patterns
	•	Detect matches line by line
	•	Support monitoring both stdout and stderr
	•	Allow case-sensitive matching by default
	•	Future extension: optional ignore-case flag

Exit status monitoring
	•	When no `--pattern` flag is specified, do not inspect stdout/stderr for regex matches
	•	Wait for the child process to finish
	•	Send a notification with the command, exit code, and timestamp
	•	Existing pattern-matching behavior remains unchanged when `--pattern` is specified

Notification

Support at least these two notification backends:

1. osascript
Show a local macOS notification using osascript.

Example notification behavior:
	•	title: vigilo
	•	subtitle or body: matched pattern / matched line summary / command name

Example implementation idea:

osascript -e 'display notification "Matched: ERROR" with title "vigilo"'

2. slack
Send a notification to Slack using an Incoming Webhook URL defined in a config file.

Notification payload should include:
	•	tool name (vigilo)
	•	executed command
	•	matched pattern
	•	matched line
	•	timestamp

Example Slack message:

[vigilo] pattern matched
command: terraform apply
pattern: Error:
line: Error: creating security group ...
time: 2026-03-06T11:00:00+09:00

Example Slack message for exit status monitoring:

[vigilo] command finished
command: terraform apply
exit_code: 1
time: 2026-03-06T11:00:00+09:00


⸻

Configuration

Config file

vigilo should support a config file for notification settings.

Preferred locations:
	1.	$XDG_CONFIG_HOME/vigilo/config.yaml
	2.	~/.config/vigilo/config.yaml

Example:

slack:
  webhook_url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"

Possible future extension:

default_notify: osascript
slack:
  webhook_url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
osascript:
  title: "vigilo"

Environment variable support

Optional future support:

VIGILO_SLACK_WEBHOOK_URL=...

For v1, config file support is enough.

⸻

CLI Design

Basic syntax

vigilo [flags] -- <command> [args...]

Example:

vigilo --pattern 'ERROR|FATAL' --notify osascript -- kubectl logs -f pod/app

Monitoring modes
	•	Pattern monitoring mode
	•	activated when `--pattern` / `-p` is provided
	•	monitors stdout/stderr for regex matches
	•	Exit status monitoring mode
	•	activated when `--pattern` is not provided
	•	notifies when the command finishes

Pattern mode flags
	•	--pattern, -p
	•	regex pattern to detect
	•	may be specified multiple times

Required flags
	•	--notify
	•	notification method
	•	supported values in v1:
	•	osascript
	•	slack

Optional flags
	•	--config
	•	explicit config file path
	•	--once
	•	notify only once, then keep streaming output
	•	--exit-on-match
	•	exit immediately after first match
	•	--no-stdout
	•	do not mirror child output to terminal
	•	mostly for pipe or automation use
	•	--prefix-stderr
	•	prefix stderr lines for easier distinction if needed

Examples

vigilo -p 'ERROR' --notify osascript -- kubectl logs -f pod/app

vigilo -p 'Error:' -p 'Failed' --notify slack -- terraform apply

vigilo -p 'panic|fatal' --notify osascript --once -- ./run-long-job.sh

vigilo -p 'ERROR' --notify slack --config ~/.config/vigilo/config.yaml -- tail -f app.log

vigilo --notify osascript -- terraform apply

vigilo --notify slack -- ./deploy.sh


⸻

Matching Behavior

Initial version
	•	Read stdout/stderr as line streams
	•	Evaluate each complete line against all registered regex patterns
	•	When a line matches:
	•	trigger notification
	•	continue streaming unless --once or --exit-on-match is set

Duplicate notifications

To avoid spamming:
	•	v1 may notify on every match
	•	future improvement:
	•	deduplicate same line for N seconds
	•	cooldown per pattern

Backward compatibility
	•	When `--pattern` is specified, existing pattern-matching behavior remains unchanged

⸻

Exit Status Behavior

Initial version
	•	When no `--pattern` flag is given, vigilo switches to exit status monitoring mode
	•	stdout/stderr continue streaming normally
	•	no regex matching is performed
	•	when the command exits, vigilo sends a notification with:
	•	command
	•	exit code
	•	timestamp
	•	v1 sends a notification for both successful and failed exits

Example:

[vigilo] command finished
command: terraform apply
exit_code: 1
time: 2026-03-06T11:00:00+09:00

⸻

Notification Behavior

osascript backend

When a match occurs:
	•	invoke osascript
	•	show native macOS notification

Suggested format:
	•	Title: vigilo
	•	Body: Matched pattern "<pattern>": <line>

If line is too long:
	•	truncate to reasonable length, e.g. 180-240 chars

slack backend

When a match occurs:
	•	read Slack webhook URL from config
	•	send JSON payload via HTTP POST

Suggested payload structure:

{
  "text": "[vigilo] pattern matched\ncommand: terraform apply\npattern: Error:\nline: Error: creating security group ...\ntime: 2026-03-06T11:00:00+09:00"
}

Failure behavior:
	•	if Slack notification fails, print warning to stderr
	•	do not crash the monitored command solely due to notification failure

Exit status notifications

When a command finishes in exit status monitoring mode:
	•	send a notification containing the command, exit code, and timestamp
	•	if exit code is `0`, send a success notification
	•	if exit code is non-zero, send a failure notification

Suggested Slack payload structure:

{
  "text": "[vigilo] command finished\ncommand: terraform apply\nexit_code: 1\ntime: 2026-03-06T11:00:00+09:00"
}

⸻

Error Handling

Invalid regex
	•	show clear error
	•	exit with non-zero status

Missing command
	•	show usage
	•	exit with non-zero status

Missing Slack webhook config
	•	if --notify slack is used and webhook URL is not configured:
	•	show clear error
	•	exit with non-zero status

osascript unavailable
	•	if osascript command is unavailable:
	•	show clear error
	•	exit with non-zero status

Child process exits
	•	vigilo should exit with the child process exit code unless a tool-specific fatal error occurs before execution

⸻

UX Expectations

Good defaults
	•	easy to run for one-off monitoring
	•	no complicated setup for osascript
	•	Slack works once webhook is placed in config

Output style
	•	preserve normal command output
	•	optionally print a small diagnostic line when match happens, such as:

[vigilo] matched pattern "ERROR"

This should go to stderr to avoid corrupting stdout pipelines.

⸻

Example User Flows

Local desktop alert during terraform apply

vigilo -p 'Error:|Failed' --notify osascript -- terraform apply

Expected behavior:
	•	terraform output continues normally
	•	if line matches, macOS notification appears

Kubernetes log watcher with Slack forwarding

vigilo -p 'ERROR|FATAL|panic' --notify slack -- kubectl logs -f deploy/api

Expected behavior:
	•	logs continue in terminal
	•	matching lines are posted to Slack webhook

Exit status monitoring for a local deploy

vigilo --notify osascript -- terraform apply

Expected behavior:
	•	terraform output continues normally
	•	when the command finishes, a notification appears with the exit code

Exit status monitoring with Slack forwarding

vigilo --notify slack -- ./deploy.sh

Expected behavior:
	•	command output continues in the terminal
	•	when the command finishes, the exit code is posted to Slack

⸻

Suggested Implementation Plan

Language

Go

Reason:
	•	easy single-binary distribution
	•	good process execution support
	•	regex and HTTP support in stdlib
	•	suitable for CLI tooling

Internal package ideas

cmd/vigilo/
internal/cli/
internal/config/
internal/matcher/
internal/notifier/
internal/notifier/osascript/
internal/notifier/slack/
internal/runner/

High-level flow
	1.	Parse CLI args
	2.	Load config
	3.	Compile regex patterns
	4.	Start child command
	5.	Read stdout/stderr line by line
	6.	Mirror output to terminal
	7.	On match:
	•	build event
	•	invoke notifier backend
	8.	Wait for command completion
	9.	Exit with child exit code

⸻

Future Enhancements
	•	multiple notify backends simultaneously
	•	e.g. --notify osascript,slack
	•	match cooldown / deduplication
	•	ignore-case flag
	•	colored match highlighting
	•	YAML-defined named rules
	•	JSON output mode
	•	shell integration
	•	PTY mode for commands that behave differently without a TTY
	•	desktop sound notification
	•	Discord webhook support
	•	custom templates for notification messages

⸻

Example README Summary

vigilo runs a command, watches its output, matches regex patterns, and notifies you when something important happens.

Example:

vigilo -p 'ERROR|FATAL' --notify osascript -- kubectl logs -f pod/app

or

vigilo -p 'Error:' --notify slack -- terraform apply

Slack webhook URL is configured in:

~/.config/vigilo/config.yaml

Example config:

slack:
  webhook_url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"


⸻

Initial Scope for v1

The first version should support:
	•	running an arbitrary command
	•	streaming stdout/stderr
	•	exit status monitoring when no `--pattern` is specified
	•	regex matching with one or more --pattern
	•	--notify osascript
	•	--notify slack
	•	Slack webhook config file
	•	proper exit codes
	•	clear error messages

That is enough for a useful first release.
