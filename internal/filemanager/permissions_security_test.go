package filemanager

import (
	"os/exec"
	"testing"
)

func TestShellQuotePreservesShellMetacharacters(t *testing.T) {
	inputs := []string{
		"/sdcard/file name",
		"/sdcard/;id;#",
		"/sdcard/it's a file",
		"/sdcard/$(touch /tmp/should-not-exist)",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", "printf %s "+shellQuote(input))
			got, err := cmd.Output()
			if err != nil {
				t.Fatalf("shell command failed: %v", err)
			}
			if string(got) != input {
				t.Fatalf("shellQuote changed input: got %q, want %q", got, input)
			}
		})
	}
}
