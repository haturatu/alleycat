package main

import (
	"regexp"
	"sort"
	"strings"
)

var localePathPattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)
var localizedSitemapPattern = regexp.MustCompile(`^/sitemap-([a-z]{2,3}(?:-[a-z0-9]{2,8})*)\.xml$`)

func parseTranslationLocales(value string) []string {
	raw := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == ';'
	})
	result := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		locale, ok := parseLocaleSegment(item)
		if !ok {
			continue
		}
		if _, exists := seen[locale]; exists {
			continue
		}
		seen[locale] = struct{}{}
		result = append(result, locale)
	}
	sort.Strings(result)
	return result
}

func isEnabledTranslationLocale(settings SettingsRecord, targetLocale string) bool {
	locale, ok := parseLocaleSegment(targetLocale)
	if !ok {
		return false
	}
	for _, enabled := range parseTranslationLocales(settings.TranslationLocales) {
		if enabled == locale {
			return true
		}
	}
	return false
}

func parseLocaleSegment(value string) (string, bool) {
	locale := normalizeLocale(value)
	if locale == "" || locale != strings.TrimSpace(value) || !localePathPattern.MatchString(locale) {
		return "", false
	}
	return locale, true
}

func extractSitemapLocale(path string) (string, bool) {
	matches := localizedSitemapPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return "", false
	}
	return parseLocaleSegment(matches[1])
}

func extractLocalizedPostRoute(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[1] != "posts" || strings.TrimSpace(parts[2]) == "" {
		return "", "", false
	}
	locale, ok := parseLocaleSegment(parts[0])
	if !ok {
		return "", "", false
	}
	return locale, strings.TrimSpace(parts[2]), true
}
