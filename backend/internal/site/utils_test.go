package site

import (
	"reflect"
	"testing"
)

func TestStripHTML(t *testing.T) {
	t.Parallel()

	got := stripHTML("<p>Hello <strong>world</strong></p>\n<p>next</p>")
	want := "Hello world next"
	if got != want {
		t.Fatalf("stripHTML returned %q, want %q", got, want)
	}
}

func TestBuildExcerpt(t *testing.T) {
	t.Parallel()

	got := buildExcerpt("<p>abcdef</p>", 4)
	if got != "abcd..." {
		t.Fatalf("buildExcerpt returned %q, want %q", got, "abcd...")
	}
}

func TestParseTags(t *testing.T) {
	t.Parallel()

	got := parseTags("go, testing, go,  api ")
	want := []string{"go", "testing", "api"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTags returned %#v, want %#v", got, want)
	}
}

func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "2026-03-22T10:11:12Z", want: "2026-03-22"},
		{input: "2026-03-22", want: "2026-03-22"},
		{input: "not-a-date", want: "not-a-date"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := formatDate(tt.input); got != tt.want {
				t.Fatalf("formatDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	if got := normalizeURL(" https://example.com/path/ "); got != "https://example.com/path" {
		t.Fatalf("normalizeURL returned %q", got)
	}
}

func TestCalcReadTime(t *testing.T) {
	t.Parallel()

	if got := calcReadTime("<p>short</p>"); got != 1 {
		t.Fatalf("calcReadTime(short) = %d, want 1", got)
	}
	if got := calcReadTime("<p>" + string(make([]byte, 1400)) + "</p>"); got < 2 {
		t.Fatalf("calcReadTime(long) = %d, want at least 2", got)
	}
}
