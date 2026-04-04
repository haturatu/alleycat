package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExtractPostOGImageRequest(t *testing.T) {
	t.Parallel()

	locale, slug, ok := extractPostOGImageRequest("/og/ja/posts/hello-world.png")
	if !ok || locale != "ja" || slug != "hello-world" {
		t.Fatalf("unexpected localized og request parse result: ok=%v locale=%q slug=%q", ok, locale, slug)
	}

	locale, slug, ok = extractPostOGImageRequest("/og/posts/hello-world.png")
	if !ok || locale != "" || slug != "hello-world" {
		t.Fatalf("unexpected base og request parse result: ok=%v locale=%q slug=%q", ok, locale, slug)
	}
}

func TestRenderGeneratedPostOGImage(t *testing.T) {
	t.Parallel()

	body, err := renderGeneratedPostOGImage(&PostRecord{
		Title:       "Generated Image Title",
		Slug:        "generated-image-title",
		Body:        "<p>This is an article without inline images.</p>",
		PublishedAt: "2026-03-22T10:11:12Z",
	}, "ja", SettingsRecord{
		SiteName:     "Alleycat",
		SiteLanguage: "ja",
	})
	if err != nil {
		t.Fatalf("renderGeneratedPostOGImage returned error: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("renderGeneratedPostOGImage should return bytes")
	}
	if !bytes.HasPrefix(body, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("renderGeneratedPostOGImage should return png bytes")
	}
}

func TestBuildAbsoluteSiteURL(t *testing.T) {
	t.Parallel()

	got := buildAbsoluteSiteURL(SettingsRecord{SiteURL: "https://example.com/"}, "/og/posts/hello.png")
	if got != "https://example.com/og/posts/hello.png" {
		t.Fatalf("buildAbsoluteSiteURL returned %q", got)
	}
	if rel := buildAbsoluteSiteURL(SettingsRecord{}, "/og/posts/hello.png"); !strings.HasPrefix(rel, "/og/posts/hello.png") {
		t.Fatalf("buildAbsoluteSiteURL should keep relative path when site_url is empty, got %q", rel)
	}
}
