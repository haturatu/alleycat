package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var localizedSitemapPattern = regexp.MustCompile(`^/sitemap-([a-z]{2,3}(?:-[a-z0-9]{2,8})*)\.xml$`)

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func writeSitemap(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	baseURL := sitemapBaseURL(r, settings)
	if baseURL == "" {
		http.Error(w, "missing site url", http.StatusBadRequest)
		return
	}

	urls := make([]sitemapURL, 0, 256)
	urls = append(urls,
		sitemapURL{Loc: baseURL + "/"},
		sitemapURL{Loc: baseURL + "/archive/"},
	)

	if settings.EnableFeedXML {
		urls = append(urls, sitemapURL{Loc: baseURL + "/feed.xml"})
	}
	if settings.EnableFeedJSON {
		urls = append(urls, sitemapURL{Loc: baseURL + "/feed.json"})
	}

	for _, page := range listPublishedPages() {
		pageURL := strings.TrimSpace(page.URL)
		if pageURL == "" {
			continue
		}
		if !strings.HasPrefix(pageURL, "/") {
			pageURL = "/" + pageURL
		}
		date := strings.TrimSpace(page.PublishedAt)
		if date == "" {
			date = strings.TrimSpace(page.Date)
		}
		urls = append(urls, sitemapURL{
			Loc:     baseURL + pageURL,
			LastMod: sitemapDate(date),
		})
	}

	for _, post := range listPublishedPosts() {
		slug := strings.TrimSpace(post.Slug)
		if slug == "" {
			continue
		}
		date := strings.TrimSpace(post.PublishedAt)
		if date == "" {
			date = strings.TrimSpace(post.Date)
		}
		urls = append(urls, sitemapURL{
			Loc:     fmt.Sprintf("%s/posts/%s/", baseURL, slug),
			LastMod: sitemapDate(date),
		})
	}

	writeSitemapXML(w, urls)
}

func writeLocalizedSitemap(w http.ResponseWriter, r *http.Request, settings SettingsRecord, locale string) {
	baseURL := sitemapBaseURL(r, settings)
	if baseURL == "" {
		http.Error(w, "missing site url", http.StatusBadRequest)
		return
	}

	if !isEnabledTranslationLocale(settings, locale) {
		http.NotFound(w, r)
		return
	}

	translations := listPublishedTranslationsByLocale(locale)
	urls := make([]sitemapURL, 0, len(translations))
	for _, item := range translations {
		slug := strings.TrimSpace(item.Slug)
		if slug == "" {
			continue
		}
		date := strings.TrimSpace(item.PublishedAt)
		urls = append(urls, sitemapURL{
			Loc:     fmt.Sprintf("%s/%s/posts/%s/", baseURL, locale, slug),
			LastMod: sitemapDate(date),
		})
	}
	writeSitemapXML(w, urls)
}

func extractSitemapLocale(path string) (string, bool) {
	matches := localizedSitemapPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return "", false
	}
	return normalizeLocale(matches[1]), true
}

func isEnabledTranslationLocale(settings SettingsRecord, targetLocale string) bool {
	targetLocale = normalizeLocale(targetLocale)
	if targetLocale == "" {
		return false
	}
	locales := parseTranslationLocales(settings.TranslationLocales)
	for _, locale := range locales {
		if locale == targetLocale {
			return true
		}
	}
	return false
}

func parseTranslationLocales(value string) []string {
	raw := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == ';'
	})
	result := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		locale := normalizeLocale(item)
		if locale == "" {
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

func listPublishedPosts() []PostRecord {
	posts := make([]PostRecord, 0, 256)
	page := 1
	const perPage = 200
	for {
		data, err := getPosts(map[string]string{
			"page":    fmt.Sprintf("%d", page),
			"perPage": fmt.Sprintf("%d", perPage),
			"filter":  "published = true",
			"sort":    "-published_at",
		})
		if err != nil {
			data, err = getPosts(map[string]string{
				"page":    fmt.Sprintf("%d", page),
				"perPage": fmt.Sprintf("%d", perPage),
				"filter":  "published = true",
				"sort":    "-date",
			})
			if err != nil {
				break
			}
		}
		posts = append(posts, data.Items...)
		if len(data.Items) < perPage {
			break
		}
		page++
	}
	return posts
}

func listPublishedPages() []PageRecord {
	pages := make([]PageRecord, 0, 128)
	page := 1
	const perPage = 200
	for {
		data, err := fetchList[PageRecord](fmt.Sprintf("%s/api/collections/pages/records", pbURL), map[string]string{
			"page":    fmt.Sprintf("%d", page),
			"perPage": fmt.Sprintf("%d", perPage),
			"filter":  "published = true",
			"sort":    "-published_at",
		})
		if err != nil {
			data, err = fetchList[PageRecord](fmt.Sprintf("%s/api/collections/pages/records", pbURL), map[string]string{
				"page":    fmt.Sprintf("%d", page),
				"perPage": fmt.Sprintf("%d", perPage),
				"filter":  "published = true",
				"sort":    "-date",
			})
			if err != nil {
				break
			}
		}
		pages = append(pages, data.Items...)
		if len(data.Items) < perPage {
			break
		}
		page++
	}
	return pages
}

func listPublishedTranslationsByLocale(locale string) []PostTranslationRecord {
	items := make([]PostTranslationRecord, 0, 256)
	page := 1
	const perPage = 200
	for {
		data, err := getPostTranslations(map[string]string{
			"page":    fmt.Sprintf("%d", page),
			"perPage": fmt.Sprintf("%d", perPage),
			"filter":  fmt.Sprintf("published = true && locale = \"%s\"", escapeFilter(locale)),
			"sort":    "-published_at",
		})
		if err != nil {
			break
		}
		items = append(items, data.Items...)
		if len(data.Items) < perPage {
			break
		}
		page++
	}
	return items
}

func sitemapBaseURL(r *http.Request, settings SettingsRecord) string {
	if normalized := normalizeURL(settings.SiteURL); normalized != "" {
		return normalized
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func sitemapDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	layoutsWithTZ := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.000Z",
		"2006-01-02 15:04:05Z",
	}
	for _, layout := range layoutsWithTZ {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}

	layoutsNoTZ := []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layoutsNoTZ {
		if parsed, err := time.ParseInLocation(layout, trimmed, time.UTC); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}

	if parsed, err := time.Parse("2006-01-02", trimmed); err == nil {
		return parsed.Format("2006-01-02")
	}

	return ""
}

func writeSitemapXML(w http.ResponseWriter, urls []sitemapURL) {
	payload := sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}
	body, err := xml.MarshalIndent(payload, "", "  ")
	if err != nil {
		http.Error(w, "failed to generate sitemap", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(body)
}
