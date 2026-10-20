package hotspot

import "testing"

// The rule parser must handle the exact format "ip rule show" prints on
// Android, where the table name is the final token and carries no trailing
// space:
//
//	7010:\tfrom all iif wlan1 lookup wlan0
func TestRuleTokens_StripsPriorityPrefix(t *testing.T) {
	tokens := ruleTokens("7010:\tfrom all iif wlan1 lookup wlan0")
	want := []string{"from", "all", "iif", "wlan1", "lookup", "wlan0"}

	if len(tokens) != len(want) {
		t.Fatalf("ruleTokens returned %d tokens (%v), want %d", len(tokens), tokens, len(want))
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token %d = %q, want %q (full: %v)", i, tokens[i], want[i], tokens)
		}
	}
}

func TestTokensContain_KeywordValuePairs(t *testing.T) {
	tokens := ruleTokens("7010:\tfrom all iif wlan1 lookup wlan0")

	cases := []struct {
		name    string
		keyword string
		value   string
		want    bool
	}{
		{"iif matches interface", "iif", "wlan1", true},
		{"lookup matches table", "lookup", "wlan0", true},
		{"lookup does not match other table", "lookup", "wlan1", false},
		{"iif does not match other interface", "iif", "wlan0", false},
		{"to is absent", "to", "192.168.1.0/24", false},
		{"from all present", "from", "all", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tokensContain(tokens, tc.keyword, tc.value); got != tc.want {
				t.Fatalf("tokensContain(%q, %q) = %v, want %v", tc.keyword, tc.value, got, tc.want)
			}
		})
	}
}

// A rule that only shares the priority but targets a different interface or
// table must not be mistaken for the rule BFR installed.
func TestTokensContain_RejectsForeignRuleAtSamePriority(t *testing.T) {
	foreign := ruleTokens("7010:\tfrom all iif lo lookup 100")
	bfr := ruleTokens("7010:\tfrom all iif wlan1 lookup wlan0")

	if tokensContain(foreign, "iif", "wlan1") {
		t.Error("foreign rule must not match the BFR interface")
	}
	if tokensContain(foreign, "lookup", "wlan0") {
		t.Error("foreign rule must not match the BFR table")
	}
	if !tokensContain(bfr, "iif", "wlan1") {
		t.Error("BFR rule must match its own interface")
	}
}

func TestRuleTokens_SubnetRule(t *testing.T) {
	tokens := ruleTokens("7000:\tfrom all to 192.168.1.0/24 lookup main")

	if !tokensContain(tokens, "to", "192.168.1.0/24") {
		t.Fatalf("expected subnet token, got %v", tokens)
	}
	if !tokensContain(tokens, "lookup", "main") {
		t.Fatalf("expected lookup main token, got %v", tokens)
	}
	if tokensContain(tokens, "to", "192.168.2.0/24") {
		t.Fatalf("must not match a different subnet, got %v", tokens)
	}
}
