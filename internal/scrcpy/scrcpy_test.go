package scrcpy_test

import (
	"testing"

	"bfr-webui-go/internal/scrcpy"
)

func TestSanitizeInput(t *testing.T) {
	// Shell injection attempt with backticks, semicolons, and spaces
	rawInput := "hello world; rm -rf /; `whoami` $PATH"
	sanitized := scrcpy.SanitizeInput(rawInput)

	// Spaces replaced by %s, shell operators stripped
	expected := "hello%sworld%srm%s-rf%s/%swhoami%sPATH"
	if sanitized != expected {
		t.Errorf("expected sanitized string '%s', got '%s'", expected, sanitized)
	}

	// Empty string
	if empty := scrcpy.SanitizeInput(""); empty != "" {
		t.Errorf("expected empty string for empty input, got '%s'", empty)
	}
}
