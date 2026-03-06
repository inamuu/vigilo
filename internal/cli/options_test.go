package cli

import "testing"

func TestParseAllowsVersionWithoutNotifyOrCommand(t *testing.T) {
	options, err := Parse([]string{"--version"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !options.ShowVersion {
		t.Fatal("Parse() ShowVersion = false, want true")
	}
}

func TestParseRequiresNotifyWithoutVersion(t *testing.T) {
	if _, err := Parse([]string{"echo", "hello"}); err == nil {
		t.Fatal("Parse() error = nil, want missing --notify error")
	}
}
