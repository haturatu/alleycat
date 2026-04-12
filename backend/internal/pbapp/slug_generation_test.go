package pbapp

import "testing"

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
