package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sourceLocaleRegressionContext() *snapshotBuildContext {
	return &snapshotBuildContext{
		postBySlug: map[string]PostRecord{
			"starlinkconoha-vpstcp": {
				ID:          "post-1",
				Title:       "Starlink ConoHa VPS TCP",
				Slug:        "starlinkconoha-vpstcp",
				Body:        "<p>Body</p>",
				Excerpt:     "Excerpt",
				PublishedAt: "2026-03-22T10:11:12Z",
				Published:   true,
			},
		},
		postByID: map[string]PostRecord{
			"post-1": {
				ID:          "post-1",
				Title:       "Starlink ConoHa VPS TCP",
				Slug:        "starlinkconoha-vpstcp",
				Body:        "<p>Body</p>",
				Excerpt:     "Excerpt",
				PublishedAt: "2026-03-22T10:11:12Z",
				Published:   true,
			},
		},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}
}

func sourceLocaleRegressionSettings() SettingsRecord {
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "en,zh-cn"
	settings.SiteURL = "https://example.com"
	settings.EnableOGPImageGeneration = true
	return settings
}

func TestSourceLocaleRegressionPostMetaUsesLocalizedOGImageRoute(t *testing.T) {
	t.Parallel()

	settings := sourceLocaleRegressionSettings()
	input := &postRenderInput{
		post: &PostRecord{
			ID:          "post-1",
			Title:       "Starlink ConoHa VPS TCP",
			Slug:        "starlinkconoha-vpstcp",
			Body:        "<p>Body</p>",
			Excerpt:     "Excerpt",
			Published:   true,
			PublishedAt: "2026-03-22T10:11:12Z",
		},
	}

	html, ok := renderPostFromInput(input, settings)
	if !ok {
		t.Fatalf("renderPostFromInput should succeed")
	}
	if !strings.Contains(html, `property="og:image" content="https://example.com/og/ja/posts/starlinkconoha-vpstcp.png"`) {
		t.Fatalf("post output should point source locale posts at the localized og:image route: %q", html)
	}
}

func TestSourceLocaleRegressionLocalizedOGImageRouteServesPNG(t *testing.T) {
	ctx := sourceLocaleRegressionContext()
	rec := httptest.NewRecorder()

	err := withSnapshotBuildContext(ctx, func() error {
		ok := servePostOGImage(rec, "/og/ja/posts/starlinkconoha-vpstcp.png", sourceLocaleRegressionSettings())
		if !ok {
			t.Fatalf("servePostOGImage should resolve /og/ja/posts/... for source locale posts")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withSnapshotBuildContext returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("servePostOGImage status = %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("servePostOGImage content-type = %q", contentType)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("servePostOGImage should return png bytes")
	}
}
