package event

import "time"

type Kind string

const (
	PatternMatched  Kind = "pattern_matched"
	CommandFinished Kind = "command_finished"
)

type Event struct {
	Kind      Kind
	Command   string
	Pattern   string
	Line      string
	ExitCode  int
	Timestamp time.Time
}
