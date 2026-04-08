package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type localeCase struct {
	Name       string   `json:"name"`
	Input      string   `json:"input"`
	Locales    []string `json:"locales"`
	Normalized string   `json:"normalized"`
}

func loadLocaleCases(t *testing.T) []localeCase {
	t.Helper()

	data, err := os.ReadFile("testdata/locale_cases.json")
	if err != nil {
		t.Fatalf("read locale cases: %v", err)
	}
	var cases []localeCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("unmarshal locale cases: %v", err)
	}
	return cases
}

func TestParseLocaleList(t *testing.T) {
	t.Parallel()

	for _, tc := range loadLocaleCases(t) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			got := parseLocaleList(tc.Input)
			if !reflect.DeepEqual(got, tc.Locales) {
				t.Fatalf("parseLocaleList(%q) = %#v, want %#v", tc.Input, got, tc.Locales)
			}
		})
	}
}

func TestNormalizeLocale(t *testing.T) {
	t.Parallel()

	cases := loadLocaleCases(t)
	if got := normalizeLocale(" En_US "); got != cases[0].Normalized {
		t.Fatalf("normalizeLocale returned %q, want %q", got, cases[0].Normalized)
	}
	if got := normalizeLocale(" ja_JP "); got != "ja-jp" {
		t.Fatalf("normalizeLocale returned %q, want %q", got, "ja-jp")
	}
}

func TestSplitTranslationBodyKeepsSmallBodyWhole(t *testing.T) {
	t.Parallel()

	body := "<p>Hello</p>\n<p>World</p>"
	got := splitTranslationBody(body, 100)
	want := []string{body}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitTranslationBody() = %#v, want %#v", got, want)
	}
}

func TestSplitTranslationBodySplitsOnBlockBoundaries(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		"<p>first paragraph with enough text</p>",
		"<p>second paragraph with enough text</p>",
		"<p>third paragraph with enough text</p>",
	}, "")

	got := splitTranslationBody(body, 50)
	if len(got) != 3 {
		t.Fatalf("chunk count = %d, want 3; chunks=%#v", len(got), got)
	}
	for _, chunk := range got {
		if len([]rune(chunk)) > 50 {
			t.Fatalf("chunk too large: %d runes in %q", len([]rune(chunk)), chunk)
		}
		if !strings.HasSuffix(chunk, "</p>") {
			t.Fatalf("expected paragraph boundary chunk, got %q", chunk)
		}
	}
}

func TestSplitTranslationBodySplitsOversizedSegment(t *testing.T) {
	t.Parallel()

	body := "<p>" + strings.Repeat("a", 120) + "</p>"
	got := splitTranslationBody(body, 40)
	if len(got) < 3 {
		t.Fatalf("expected oversized segment to split, got %#v", got)
	}
	for _, chunk := range got {
		if len([]rune(chunk)) > 40 {
			t.Fatalf("chunk too large: %d runes in %q", len([]rune(chunk)), chunk)
		}
	}
}
