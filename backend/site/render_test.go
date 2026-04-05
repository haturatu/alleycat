package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type renderPostCase struct {
	Name           string         `json:"name"`
	Settings       SettingsRecord `json:"settings"`
	Post           PostRecord     `json:"post"`
	MustContain    []string       `json:"must_contain"`
	MustNotContain []string       `json:"must_not_contain"`
}

func loadRenderPostCases(t *testing.T) []renderPostCase {
	t.Helper()

	data, err := os.ReadFile("testdata/render_post_cases.json")
	if err != nil {
		t.Fatalf("read render fixtures: %v", err)
	}
	var fixtures []renderPostCase
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("unmarshal render fixtures: %v", err)
	}
	return fixtures
}

func TestHighlightStylesheets(t *testing.T) {
	t.Parallel()

	dark, light := highlightStylesheets(SettingsRecord{HighlightTheme: "dracula"})
	if !strings.Contains(dark, "dracula") {
		t.Fatalf("dark stylesheet = %q, want dracula theme", dark)
	}
	if !strings.Contains(light, "github.min.css") {
		t.Fatalf("light stylesheet = %q, want github fallback", light)
	}
}

func TestRenderHeadRespectsFeatureFlags(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "en"
	settings.EnableAnalytics = true
	settings.AnalyticsURL = "https://analytics.example/script.js"
	settings.AnalyticsSiteID = "site-1"
	settings.EnableAds = true
	settings.AdsClient = "ca-pub-123"
	settings.EnableCodeHighlight = false

	html := renderHead("Home", settings)
	if !strings.Contains(html, settings.AnalyticsURL) {
		t.Fatalf("renderHead should include analytics script")
	}
	if !strings.Contains(html, settings.AdsClient) {
		t.Fatalf("renderHead should include ads script")
	}
	if strings.Contains(html, "highlight.min.js") {
		t.Fatalf("renderHead should omit highlight assets when disabled")
	}
	if !strings.Contains(html, `<meta name="robots" content="max-image-preview:large" />`) {
		t.Fatalf("renderHead should include robots max-image-preview meta tag")
	}
}

func TestRenderHeadOmitsDisabledFeedAlternates(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.EnableFeedXML = false
	settings.EnableFeedJSON = false

	html := renderHead("Home", settings)
	if strings.Contains(html, `/feed.xml`) {
		t.Fatalf("renderHead should omit xml feed alternate when disabled")
	}
	if strings.Contains(html, `/feed.json`) {
		t.Fatalf("renderHead should omit json feed alternate when disabled")
	}
}

func TestRenderFeedLinkListRespectsEnabledFeeds(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.EnableFeedXML = true
	settings.EnableFeedJSON = false

	got := renderFeedLinkList(settings)
	if !strings.Contains(got, `/feed.xml`) {
		t.Fatalf("renderFeedLinkList should include enabled xml feed")
	}
	if strings.Contains(got, `/feed.json`) {
		t.Fatalf("renderFeedLinkList should omit disabled json feed")
	}

	settings.EnableFeedXML = false
	got = renderFeedLinkList(settings)
	if got != "" {
		t.Fatalf("renderFeedLinkList should be empty when all feeds disabled, got %q", got)
	}
}

func TestParseArchiveRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path  string
		want  archiveRoute
		label string
	}{
		{
			label: "root",
			path:  "/archive/",
			want:  archiveRoute{pageNumber: 1, basePath: "/archive", filter: `published = true`, title: "Archive"},
		},
		{
			label: "root-page",
			path:  "/archive/3/",
			want:  archiveRoute{pageNumber: 3, basePath: "/archive", filter: `published = true`, title: "Archive"},
		},
		{
			label: "tag",
			path:  "/archive/go/",
			want:  archiveRoute{pageNumber: 1, basePath: "/archive/go", filter: `published = true && tags ~ "go"`, title: "tag: go"},
		},
		{
			label: "category",
			path:  "/archive/category/news/",
			want:  archiveRoute{pageNumber: 1, basePath: "/archive/category/news", filter: `published = true && category = "news"`, title: "category: news"},
		},
		{
			label: "category-falls-back-to-tag",
			path:  "/archive/category/",
			want:  archiveRoute{pageNumber: 1, basePath: "/archive/category", filter: `published = true && tags ~ "category"`, title: "tag: category"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()
			if got := parseArchiveRoute(tt.path); got != tt.want {
				t.Fatalf("parseArchiveRoute(%q) = %#v, want %#v", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildTOC(t *testing.T) {
	t.Parallel()

	body, toc := buildTOC("<h2>Intro</h2><p>x</p><h3>Details</h3>", true)
	if !strings.Contains(body, `id="intro"`) {
		t.Fatalf("body = %q, want generated intro heading id", body)
	}
	if !strings.Contains(body, `id="details"`) {
		t.Fatalf("body = %q, want generated details heading id", body)
	}
	if !strings.Contains(toc, "#intro") || !strings.Contains(toc, "#details") {
		t.Fatalf("toc = %q, want links to generated heading ids", toc)
	}
}

func TestRenderPostTags(t *testing.T) {
	t.Parallel()

	html := renderPostTags([]string{"go", "testing"}, true)
	if !strings.Contains(html, `/archive/go/`) || !strings.Contains(html, `testing`) {
		t.Fatalf("renderPostTags output missing tags: %q", html)
	}
	if hidden := renderPostTags([]string{"go"}, false); hidden != "" {
		t.Fatalf("renderPostTags should be empty when hidden, got %q", hidden)
	}
}

func TestRenderCommentsSection(t *testing.T) {
	t.Parallel()

	if got := renderCommentsSection(SettingsRecord{}); got != "" {
		t.Fatalf("comments should be empty when disabled, got %q", got)
	}

	settings := SettingsRecord{
		EnableComments:    true,
		CommentsScriptTag: `<script src="https://giscus.app/client.js"></script>`,
	}
	got := renderCommentsSection(settings)
	if !strings.Contains(got, "giscus.app/client.js") {
		t.Fatalf("comments section missing script: %q", got)
	}
}

func TestRenderRelatedPosts(t *testing.T) {
	t.Parallel()

	got := renderRelatedPosts([]PostRecord{{Title: "One", Slug: "one"}}, "/posts/")
	if !strings.Contains(got, "/posts/one/") || !strings.Contains(got, "Related Posts") {
		t.Fatalf("renderRelatedPosts output = %q", got)
	}
}

func TestRenderPageFromRecord(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	page := &PageRecord{Title: "About", Body: "<p>Hello</p>"}
	html, ok := renderPageFromRecord(page, settings)
	if !ok {
		t.Fatalf("renderPageFromRecord should succeed")
	}
	if !strings.Contains(html, "<h1 class=\"post-title\">About</h1>") {
		t.Fatalf("page output missing title: %q", html)
	}
}

func TestRenderPostFromInputRespectsFlags(t *testing.T) {
	t.Parallel()

	for _, fixture := range loadRenderPostCases(t) {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			t.Parallel()

			settings := defaultSettings()
			settings.ShowCategories = fixture.Settings.ShowCategories
			settings.ShowTags = fixture.Settings.ShowTags
			settings.ShowRelatedPosts = fixture.Settings.ShowRelatedPosts
			settings.EnableComments = fixture.Settings.EnableComments
			settings.CommentsScriptTag = fixture.Settings.CommentsScriptTag
			settings.ShowToc = fixture.Settings.ShowToc
			settings.TranslationSourceLocale = fixture.Settings.TranslationSourceLocale
			settings.SiteLanguage = fixture.Settings.SiteLanguage

			input := &postRenderInput{post: &fixture.Post}

			html, ok := renderPostFromInput(input, settings)
			if !ok {
				t.Fatalf("renderPostFromInput should succeed")
			}

			for _, token := range fixture.MustContain {
				if !strings.Contains(html, token) {
					t.Fatalf("post output missing %q: %q", token, html)
				}
			}
			for _, token := range fixture.MustNotContain {
				if strings.Contains(html, token) {
					t.Fatalf("post output should not contain %q: %q", token, html)
				}
			}
		})
	}
}

func TestRenderPostFromInputAddsOGImageMetaWhenEnabled(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.SiteURL = "https://example.com"
	settings.EnableOGPImageGeneration = true

	input := &postRenderInput{
		post: &PostRecord{
			ID:          "post-og",
			Title:       "Share Title",
			Slug:        "share-title",
			Body:        "<p>Body without inline image.</p>",
			Excerpt:     "Custom excerpt",
			Published:   true,
			PublishedAt: "2026-03-22T10:11:12Z",
		},
	}

	html, ok := renderPostFromInput(input, settings)
	if !ok {
		t.Fatalf("renderPostFromInput should succeed")
	}
	if !strings.Contains(html, `property="og:image" content="https://example.com/og/posts/share-title.png"`) {
		t.Fatalf("post output missing generated og:image meta: %q", html)
	}
	if !strings.Contains(html, `name="twitter:card" content="summary_large_image"`) {
		t.Fatalf("post output missing twitter large image card meta: %q", html)
	}
}

func TestRenderPostFromInputAddsLocalizedOGImageMetaWhenLocalized(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.SiteURL = "https://example.com"
	settings.EnableOGPImageGeneration = true

	input := &postRenderInput{
		locale: "en",
		post: &PostRecord{
			ID:      "post-og-en",
			Title:   "Share Title",
			Slug:    "share-title",
			Body:    "<p>Localized body</p>",
			Excerpt: "Localized excerpt",
		},
		translation: &PostTranslationRecord{
			ID:         "translation-og-en",
			SourcePost: "post-og-source",
			Locale:     "en",
			Title:      "Share Title",
			Slug:       "share-title",
			Body:       "<p>Localized body</p>",
			Excerpt:    "Localized excerpt",
			Published:  true,
		},
	}

	html, ok := renderPostFromInput(input, settings)
	if !ok {
		t.Fatalf("renderPostFromInput should succeed")
	}
	if !strings.Contains(html, `property="og:image" content="https://example.com/og/en/posts/share-title.png"`) {
		t.Fatalf("localized post output missing localized og:image meta: %q", html)
	}
}
