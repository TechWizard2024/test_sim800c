package serial

import (
	"strings"
	"testing"
)

func TestFormatUSSDText(t *testing.T) {
	in := "\r\n+CUSD: 1,\"Bienvenue au #111#\n\n    1: Acheter un Pass\n      2: Consulter mes soldes\n   \n   3: Renouveler un Pass\t\r\n\n�\n"

	out := formatUSSDText(in)
	if out == "" {
		t.Fatalf("formatUSSDText returned empty string")
	}

	// Ensure some key substrings are present and there are no strange replacement chars
	expected := []string{"Bienvenue", "1: Acheter", "2: Consulter", "3: Renouveler"}
	for _, sub := range expected {
		if !strings.Contains(out, sub) {
			t.Fatalf("formatted output missing expected substring %q: %q", sub, out)
		}
	}

	if strings.Contains(out, "�") {
		t.Fatalf("formatted output contains replacement char: %q", out)
	}
}
