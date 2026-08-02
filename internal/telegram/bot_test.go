package telegram

import (
	"testing"
)

func TestIsValidBotToken(t *testing.T) {
	tests := []struct {
		token string
		valid bool
	}{
		{"123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ", true},
		{"987654321:AA-BB_CC1234", true},
		{"invalidtoken", false},
		{"abc:123456", false},
		{"123456:", false},
		{":secret", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isValidBotToken(tt.token)
		if got != tt.valid {
			t.Errorf("isValidBotToken(%q) = %v; want %v", tt.token, got, tt.valid)
		}
	}
}

func TestChatAllowed(t *testing.T) {
	m := &Manager{
		config: Config{
			AllowedChatIDs: []int64{1001, 1002, 1003},
		},
	}

	if !m.isChatAllowed(1001) {
		t.Errorf("Expected chat ID 1001 to be allowed")
	}
	if m.isChatAllowed(9999) {
		t.Errorf("Expected chat ID 9999 to be denied")
	}
}
