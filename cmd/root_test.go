package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootNoArgsShowsHelp(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute root: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Usage:") {
		t.Fatalf("expected help usage output, got:\n%s", got)
	}
	if !strings.Contains(got, "doctor") {
		t.Fatalf("expected command list in help output, got:\n%s", got)
	}
}
