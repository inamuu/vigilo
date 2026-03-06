package matcher

import "testing"

func TestCompileRejectsInvalidRegex(t *testing.T) {
	if _, err := Compile([]string{"("}); err == nil {
		t.Fatal("Compile() error = nil, want invalid regex error")
	}
}

func TestMatchReturnsFirstMatchingPattern(t *testing.T) {
	m, err := Compile([]string{"WARN", "ERROR"})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	pattern, ok := m.Match("ERROR: something bad happened")
	if !ok {
		t.Fatal("Match() ok = false, want true")
	}

	if pattern != "ERROR" {
		t.Fatalf("Match() pattern = %q, want %q", pattern, "ERROR")
	}
}
