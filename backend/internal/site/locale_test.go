package site

import "testing"

func TestParseLocaleSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  string
		ok    bool
	}{
		{value: "ja", want: "ja", ok: true},
		{value: "en-us", want: "en-us", ok: true},
		{value: " En_US ", ok: false},
		{value: "", ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			got, ok := parseLocaleSegment(tt.value)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseLocaleSegment(%q) = (%q, %v), want (%q, %v)", tt.value, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExtractLocalizedPostRoute(t *testing.T) {
	t.Parallel()

	locale, slug, ok := extractLocalizedPostRoute("/ja/posts/hello/")
	if !ok || locale != "ja" || slug != "hello" {
		t.Fatalf("extractLocalizedPostRoute localized = (%q, %q, %v)", locale, slug, ok)
	}

	locale, slug, ok = extractLocalizedPostRoute("/posts/hello/")
	if ok || locale != "" || slug != "" {
		t.Fatalf("extractLocalizedPostRoute non-localized = (%q, %q, %v)", locale, slug, ok)
	}
}

func TestIsSourceLocale(t *testing.T) {
	t.Parallel()

	settings := SettingsRecord{
		SiteLanguage:            "ja",
		TranslationSourceLocale: "ja",
		TranslationLocales:      "en,zh-cn",
	}

	if !isSourceLocale(settings, "ja") {
		t.Fatalf("isSourceLocale should treat translation source locale as source")
	}
	if isSourceLocale(settings, "zh-cn") {
		t.Fatalf("isSourceLocale should not treat translation locale as source")
	}
}
