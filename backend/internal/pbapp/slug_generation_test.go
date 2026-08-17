package pbapp

import (
	"strings"
	"testing"
)

func TestNormalizeGeneratedSlug(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "Hello World", want: "hello-world"},
		{input: "  GPT-5.4 入門  ", want: "gpt-54"},
		{input: "multi___space   slug", want: "multi-space-slug"},
		{input: "---already-clean---", want: "already-clean"},
	}

	for _, tc := range cases {
		if got := normalizeGeneratedSlug(tc.input); got != tc.want {
			t.Fatalf("normalizeGeneratedSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestProviderErrorIncludesBoundedUpstreamBody(t *testing.T) {
	message := strings.Repeat("x", maxProviderErrorMessageBytes+20)
	err := (&ProviderError{Provider: "opencode-zen", StatusCode: 400, Body: message}).Error()

	if !strings.Contains(err, "status=400") {
		t.Fatalf("error = %q, want status", err)
	}
	if !strings.Contains(err, "body="+strings.Repeat("x", maxProviderErrorMessageBytes)) {
		t.Fatalf("error = %q, want bounded response body", err)
	}
	if !strings.Contains(err, "…(truncated)") {
		t.Fatalf("error = %q, want truncation marker", err)
	}
}
