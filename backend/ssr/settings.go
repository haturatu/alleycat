package main

import (
	"fmt"
	"strings"
)

var (
	defaultSiteName        = getEnv("SITE_NAME", "Example Blog")
	defaultDescription     = getEnv("SITE_DESCRIPTION", "A calm place to write.")
	defaultWelcome         = getEnv("HOME_WELCOME", "Welcome to your blog")
	defaultTopImage        = getEnv("HOME_TOP_IMAGE", "/default-hero.svg")
	defaultTopImageAlt     = getEnv("HOME_TOP_IMAGE_ALT", "Default hero image")
	defaultFooterHTML      = getEnv("FOOTER_HTML", "")
	defaultAnalyticsURL    = getEnv("ANALYTICS_URL", "")
	defaultAnalyticsSiteID = getEnv("ANALYTICS_SITE_ID", "")
	defaultAdsClient       = getEnv("ADS_CLIENT", "")
	defaultTheme           = getEnv("THEME", "ember")
)

func getSettings() SettingsRecord {
	settings, err := fetchList[SettingsRecord](fmt.Sprintf("%s/api/collections/settings/records", pbURL), map[string]string{
		"page":    "1",
		"perPage": "1",
	})
	if err != nil || len(settings.Items) == 0 {
		return defaultSettings()
	}
	item := settings.Items[0]
	item.ApplyDefaults()
	return item
}

func defaultSettings() SettingsRecord {
	return SettingsRecord{
		SiteName:                defaultSiteName,
		Description:             defaultDescription,
		WelcomeText:             defaultWelcome,
		HomeTopImage:            defaultTopImage,
		HomeTopImageAlt:         defaultTopImageAlt,
		FooterHTML:              defaultFooterHTML,
		Theme:                   defaultTheme,
		EnableFeedXML:           true,
		EnableFeedJSON:          true,
		FeedItemsLimit:          20,
		EnableAnalytics:         defaultAnalyticsURL != "" && defaultAnalyticsSiteID != "",
		AnalyticsURL:            defaultAnalyticsURL,
		AnalyticsSiteID:         defaultAnalyticsSiteID,
		EnableAds:               defaultAdsClient != "",
		AdsClient:               defaultAdsClient,
		EnableComments:          false,
		CommentsScriptTag:       "",
		ArchivePageSize:         10,
		HomePageSize:            3,
		ShowArchiveTags:         true,
		ShowTags:                true,
		ShowCategories:          true,
		ShowRelatedPosts:        false,
		ShowArchiveSearch:       true,
		EnableCodeHighlight:     true,
		HighlightTheme:          "github-dark",
		SiteLanguage:            "ja",
		TranslationSourceLocale: "ja",
		TranslationLocales:      "en",
		TranslationModel:        "gemini-1.5-flash",
	}
}

func (item *SettingsRecord) ApplyDefaults() {
	item.SiteName = defaultIfTrimmedBlank(item.SiteName, defaultSiteName)
	item.Description = defaultIfTrimmedBlank(item.Description, defaultDescription)
	item.WelcomeText = defaultIfTrimmedBlank(item.WelcomeText, defaultWelcome)
	item.HomeTopImage = defaultIfTrimmedBlank(item.HomeTopImage, defaultTopImage)
	item.HomeTopImageAlt = defaultIfTrimmedBlank(item.HomeTopImageAlt, defaultTopImageAlt)
	item.FooterHTML = defaultIfBlank(item.FooterHTML, defaultFooterHTML)
	item.Theme = defaultIfTrimmedBlank(item.Theme, defaultTheme)
	item.FeedItemsLimit = defaultIfZero(item.FeedItemsLimit, 20)
	item.ArchivePageSize = defaultIfZero(item.ArchivePageSize, 10)
	item.HomePageSize = defaultIfZero(item.HomePageSize, 3)
	item.SiteLanguage = defaultIfBlank(item.SiteLanguage, "ja")
	item.TranslationSourceLocale = defaultIfTrimmedBlank(item.TranslationSourceLocale, item.SiteLanguage)
	item.TranslationLocales = defaultIfTrimmedBlank(item.TranslationLocales, "en")
	item.TranslationModel = defaultIfTrimmedBlank(item.TranslationModel, "gemini-1.5-flash")
}

func defaultIfTrimmedBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultIfBlank(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultIfZero(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
