package site

import (
	"strings"
	"testing"
)

func TestRenderPostSEOUsesFeaturedImageAuthorAndModifiedDate(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteURL = "https://example.com"
	settings.SiteLanguage = "ja"
	settings.EnableOGPImageGeneration = true
	post := PostRecord{
		ID:            "post-1",
		Title:         "ESP32-C6 wiring",
		Slug:          "esp32-c6-wiring",
		Body:          "<p>Body</p>",
		Excerpt:       "A wiring guide.",
		Published:     true,
		PublishedAt:   "2026-08-22T09:00:00Z",
		Updated:       "2026-08-22T10:00:00Z",
		FeaturedImage: "board photo.webp",
		Expand: &PostExpand{Author: &AuthorRecord{
			ID:   "author-1",
			Name: "Ada Lovelace",
		}},
	}
	ctx := &snapshotBuildContext{
		publishedPosts:       []PostRecord{post},
		postBySlug:           map[string]PostRecord{post.Slug: post},
		postByID:             map[string]PostRecord{post.ID: post},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	var rendered string
	err := withSnapshotBuildContext(ctx, func() error {
		var ok bool
		rendered, ok = renderPostFromInput(&postRenderInput{
			path: "/posts/esp32-c6-wiring/",
			post: &post,
		}, settings)
		if !ok {
			t.Fatalf("renderPostFromInput should succeed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withSnapshotBuildContext returned error: %v", err)
	}

	for _, want := range []string{
		`property="og:image" content="https://example.com/api/files/posts/post-1/board%20photo.webp"`,
		`name="twitter:card" content="summary_large_image"`,
		`property="article:modified_time" content="2026-08-22T10:00:00Z"`,
		`<p class="post-author">Written by <a href="/about/">Ada Lovelace</a></p>`,
		`<figure class="post-featured-image"><img src="https://example.com/api/files/posts/post-1/board%20photo.webp"`,
		`"@type":"BlogPosting"`,
		`"datePublished":"2026-08-22T09:00:00Z"`,
		`"dateModified":"2026-08-22T10:00:00Z"`,
		`"author":{"@type":"Person","name":"Ada Lovelace"}`,
		`"publisher":{"@type":"Organization","name":"Alleycat"}`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered post missing %q: %s", want, rendered)
		}
	}
	if strings.Contains(rendered, `/og/ja/posts/esp32-c6-wiring.png`) {
		t.Fatalf("featured image should take priority over generated OGP image")
	}
}

func TestFeaturedImageURLUsesTranslationCollection(t *testing.T) {
	t.Parallel()

	got := featuredImageURL(
		&PostRecord{ID: "translation-1", FeaturedImage: "source.jpg"},
		&PostTranslationRecord{ID: "translation-1", FeaturedImage: "localized.jpg"},
		SettingsRecord{SiteURL: "https://example.com"},
	)
	want := "https://example.com/api/files/post_translations/translation-1/localized.jpg"
	if got != want {
		t.Fatalf("featuredImageURL() = %q, want %q", got, want)
	}
}
