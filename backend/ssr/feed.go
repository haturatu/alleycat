package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func writeJSONFeed(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	items := fetchFeedItems(settings)
	baseURL := normalizeURL(settings.SiteURL)
	feed := map[string]any{
		"version": "https://jsonfeed.org/version/1",
		"title":   settings.SiteName,
		"home_page_url": func() string {
			if baseURL == "" {
				return ""
			}
			return baseURL + "/"
		}(),
		"feed_url": func() string {
			if baseURL == "" {
				return ""
			}
			return baseURL + "/feed.json"
		}(),
		"items": items,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(feed)
}

func writeRSSFeed(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	items := fetchFeedItems(settings)
	baseURL := normalizeURL(settings.SiteURL)
	updated := time.Now().UTC().Format(time.RFC3339)
	builder := strings.Builder{}
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<feed xmlns=\"http://www.w3.org/2005/Atom\">\n")
	builder.WriteString(fmt.Sprintf("  <title>%s</title>\n", escapeHTML(settings.SiteName)))
	if baseURL != "" {
		builder.WriteString(fmt.Sprintf("  <link href=\"%s/\"/>\n", baseURL))
		builder.WriteString(fmt.Sprintf("  <link href=\"%s/feed.xml\" rel=\"self\"/>\n", baseURL))
	}
	builder.WriteString(fmt.Sprintf("  <updated>%s</updated>\n", updated))
	builder.WriteString(fmt.Sprintf("  <id>%s</id>\n", escapeHTML(defaultString(baseURL, settings.SiteName))))
	for _, item := range items {
		builder.WriteString("  <entry>\n")
		builder.WriteString(fmt.Sprintf("    <title>%s</title>\n", escapeHTML(item.Title)))
		if item.URL != "" {
			builder.WriteString(fmt.Sprintf("    <link href=\"%s\"/>\n", escapeHTML(item.URL)))
			builder.WriteString(fmt.Sprintf("    <id>%s</id>\n", escapeHTML(item.URL)))
		}
		if item.Date != "" {
			builder.WriteString(fmt.Sprintf("    <updated>%s</updated>\n", escapeHTML(item.Date)))
		}
		builder.WriteString(fmt.Sprintf("    <summary>%s</summary>\n", escapeHTML(item.Summary)))
		builder.WriteString("  </entry>\n")
	}
	builder.WriteString("</feed>")

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	_, _ = w.Write([]byte(builder.String()))
}

type feedItem struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Date    string `json:"date_published"`
	Summary string `json:"summary"`
}

func fetchFeedItems(settings SettingsRecord) []feedItem {
	limit := settings.FeedItemsLimit
	if limit <= 0 {
		limit = 20
	}
	posts, err := getPosts(map[string]string{
		"page":    "1",
		"perPage": fmt.Sprintf("%d", limit),
		"filter":  "published = true",
		"sort":    "-published_at",
	})
	if err != nil {
		posts, _ = getPosts(map[string]string{
			"page":    "1",
			"perPage": fmt.Sprintf("%d", limit),
			"filter":  "published = true",
			"sort":    "-date",
		})
	}
	baseURL := normalizeURL(settings.SiteURL)
	items := make([]feedItem, 0, len(posts.Items))
	for _, post := range posts.Items {
		slug := strings.TrimSpace(post.Slug)
		url := ""
		if baseURL != "" && slug != "" {
			url = fmt.Sprintf("%s/posts/%s/", baseURL, slug)
		}
		body := post.Body
		if body == "" {
			body = post.Content
		}
		excerpt := post.Excerpt
		if strings.TrimSpace(excerpt) == "" {
			excerpt = buildExcerpt(body, 160)
		}
		date := post.PublishedAt
		if date == "" {
			date = post.Date
		}
		items = append(items, feedItem{
			ID:      url,
			URL:     url,
			Title:   defaultString(post.Title, slug),
			Date:    date,
			Summary: excerpt,
		})
	}
	return items
}
