package site

import "testing"

func TestSettingsApplyDefaults(t *testing.T) {
	t.Parallel()

	item := SettingsRecord{
		SiteName:                " ",
		Description:             " ",
		WelcomeText:             "",
		HomeTopImage:            " ",
		HomeTopImageAlt:         "",
		FooterHTML:              "",
		Theme:                   " ",
		FeedItemsLimit:          0,
		ArchivePageSize:         0,
		HomePageSize:            0,
		SiteLanguage:            "",
		TranslationSourceLocale: " ",
		TranslationLocales:      " ",
		TranslationModel:        "",
	}

	item.ApplyDefaults()

	if item.SiteName != defaultSiteName {
		t.Fatalf("SiteName = %q, want %q", item.SiteName, defaultSiteName)
	}
	if item.Theme != defaultTheme {
		t.Fatalf("Theme = %q, want %q", item.Theme, defaultTheme)
	}
	if item.FeedItemsLimit != 20 {
		t.Fatalf("FeedItemsLimit = %d, want 20", item.FeedItemsLimit)
	}
	if item.ArchivePageSize != 10 {
		t.Fatalf("ArchivePageSize = %d, want 10", item.ArchivePageSize)
	}
	if item.HomePageSize != 3 {
		t.Fatalf("HomePageSize = %d, want 3", item.HomePageSize)
	}
	if item.SiteLanguage != "ja" {
		t.Fatalf("SiteLanguage = %q, want %q", item.SiteLanguage, "ja")
	}
	if item.TranslationSourceLocale != "ja" {
		t.Fatalf("TranslationSourceLocale = %q, want %q", item.TranslationSourceLocale, "ja")
	}
	if item.TranslationLocales != "en" {
		t.Fatalf("TranslationLocales = %q, want %q", item.TranslationLocales, "en")
	}
	if item.TranslationModel != "gemini-1.5-flash" {
		t.Fatalf("TranslationModel = %q, want %q", item.TranslationModel, "gemini-1.5-flash")
	}
}

func TestDefaultSettingsFeedItemsLimit(t *testing.T) {
	t.Parallel()

	item := defaultSettings()
	if item.FeedItemsLimit != 20 {
		t.Fatalf("FeedItemsLimit = %d, want 20", item.FeedItemsLimit)
	}
}
