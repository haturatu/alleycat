package site

import "testing"

func TestCleanPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{input: ".", expected: "/"},
		{input: "/", expected: "/"},
		{input: "/archive/../posts/slug", expected: "/posts/slug"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := cleanPath(tt.input); got != tt.expected {
				t.Fatalf("cleanPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
