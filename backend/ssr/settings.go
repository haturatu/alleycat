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
	applySettingsDefaults(&item)
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
		ArchivePageSize:         10,
		HomePageSize:            3,
		ShowArchiveTags:         true,
		ShowTags:                true,
		ShowCategories:          true,
		ShowArchiveSearch:       true,
		EnableCodeHighlight:     true,
		HighlightTheme:          "github-dark",
		SiteLanguage:            "ja",
		TranslationSourceLocale: "ja",
		TranslationLocales:      "en",
		TranslationModel:        "gemini-1.5-flash",
	}
}

func applySettingsDefaults(item *SettingsRecord) {
	if strings.TrimSpace(item.SiteName) == "" {
		item.SiteName = defaultSiteName
	}
	if strings.TrimSpace(item.Description) == "" {
		item.Description = defaultDescription
	}
	if strings.TrimSpace(item.WelcomeText) == "" {
		item.WelcomeText = defaultWelcome
	}
	if strings.TrimSpace(item.HomeTopImage) == "" {
		item.HomeTopImage = defaultTopImage
	}
	if strings.TrimSpace(item.HomeTopImageAlt) == "" {
		item.HomeTopImageAlt = defaultTopImageAlt
	}
	if item.FooterHTML == "" {
		item.FooterHTML = defaultFooterHTML
	}
	if strings.TrimSpace(item.Theme) == "" {
		item.Theme = defaultTheme
	}
	if item.FeedItemsLimit == 0 {
		item.FeedItemsLimit = 20
	}
	if item.ArchivePageSize == 0 {
		item.ArchivePageSize = 10
	}
	if item.HomePageSize == 0 {
		item.HomePageSize = 3
	}
	if item.SiteLanguage == "" {
		item.SiteLanguage = "ja"
	}
	if strings.TrimSpace(item.TranslationSourceLocale) == "" {
		item.TranslationSourceLocale = item.SiteLanguage
	}
	if strings.TrimSpace(item.TranslationLocales) == "" {
		item.TranslationLocales = "en"
	}
	if strings.TrimSpace(item.TranslationModel) == "" {
		item.TranslationModel = "gemini-1.5-flash"
	}
}
