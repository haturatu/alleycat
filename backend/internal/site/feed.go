package site

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const feedCacheTTL = 60 * time.Second

type feedCacheEntry struct {
	expiresAt time.Time
	items     []feedItem
}

var feedItemsCache = struct {
	mu    sync.RWMutex
	items map[string]feedCacheEntry
}{
	items: map[string]feedCacheEntry{},
}

func writeJSONFeed(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	items := fetchFeedItems(settings)
	baseURL := normalizeSiteBaseURL(settings.SiteURL)
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
	setNoStoreCacheHeaders(w)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(feed)
}

func writeRSSFeed(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	items := fetchFeedItems(settings)
	baseURL := normalizeSiteBaseURL(settings.SiteURL)
	updated := time.Now().UTC().Format(time.RFC3339)
	builder := strings.Builder{}
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<feed xmlns=\"http://www.w3.org/2005/Atom\">\n")
	builder.WriteString("  <title>" + escapeHTML(settings.SiteName) + "</title>\n")
	if baseURL != "" {
		builder.WriteString("  <link href=\"" + baseURL + "/\"/>\n")
		builder.WriteString("  <link href=\"" + baseURL + "/feed.xml\" rel=\"self\"/>\n")
	}
	builder.WriteString("  <updated>" + updated + "</updated>\n")
	builder.WriteString("  <id>" + escapeHTML(defaultString(baseURL, settings.SiteName)) + "</id>\n")
	for _, item := range items {
		builder.WriteString("  <entry>\n")
		builder.WriteString("    <title>" + escapeHTML(item.Title) + "</title>\n")
		if item.URL != "" {
			builder.WriteString("    <link href=\"" + escapeHTML(item.URL) + "\"/>\n")
			builder.WriteString("    <id>" + escapeHTML(item.URL) + "</id>\n")
		}
		if item.Date != "" {
			builder.WriteString("    <updated>" + escapeHTML(item.Date) + "</updated>\n")
		}
		builder.WriteString("    <summary>" + escapeHTML(item.Summary) + "</summary>\n")
		builder.WriteString("  </entry>\n")
	}
	builder.WriteString("</feed>")

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	setNoStoreCacheHeaders(w)
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
	key := fmt.Sprintf(
		"limit=%d|excerpt=%d|site=%s",
		settings.FeedItemsLimit,
		settings.ExcerptLength,
		normalizeSiteBaseURL(settings.SiteURL),
	)
	now := time.Now()
	feedItemsCache.mu.RLock()
	cached, ok := feedItemsCache.items[key]
	feedItemsCache.mu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		return append([]feedItem(nil), cached.items...)
	}

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
	baseURL := normalizeSiteBaseURL(settings.SiteURL)
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
			length := settings.ExcerptLength
			if length <= 0 {
				length = 160
			}
			excerpt = buildExcerpt(body, length)
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
	sortFeedItems(items)

	feedItemsCache.mu.Lock()
	feedItemsCache.items[key] = feedCacheEntry{
		expiresAt: now.Add(feedCacheTTL),
		items:     append([]feedItem(nil), items...),
	}
	feedItemsCache.mu.Unlock()
	return items
}

func feedItemTime(item feedItem) time.Time {
	value := strings.TrimSpace(item.Date)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000Z",
		"2006-01-02 15:04:05Z",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func sortFeedItems(items []feedItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a := feedItemTime(items[i])
		b := feedItemTime(items[j])
		if !a.Equal(b) {
			return a.After(b)
		}
		return items[i].URL < items[j].URL
	})
}
