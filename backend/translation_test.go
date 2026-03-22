package main

import (
	"encoding/json"
	"os"
	"reflect"
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
