package main

import (
	"fmt"
	"net/http"
	"strings"
)

func writeRobotsTXT(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	lines := []string{
		"User-agent: *",
		"Allow: /",
	}

	baseURL := sitemapBaseURL(r, settings)
	if baseURL != "" {
		lines = append(lines, fmt.Sprintf("Sitemap: %s/sitemap.xml", baseURL))
		for _, locale := range parseTranslationLocales(settings.TranslationLocales) {
			lines = append(lines, fmt.Sprintf("Sitemap: %s/sitemap-%s.xml", baseURL, locale))
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strings.Join(lines, "\n") + "\n"))
}
