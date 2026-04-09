package main

import (
	"bytes"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
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

	got = buildAbsoluteSiteURL(SettingsRecord{SiteURL: "https://www.soulminingrig.com/www.soulminingrig.com"}, "/posts/self-hosted-edgedns-research/")
	if got != "https://www.soulminingrig.com/posts/self-hosted-edgedns-research" {
		t.Fatalf("buildAbsoluteSiteURL should discard site_url path, got %q", got)
	}
}

func TestServePostOGImageDoesNotRequireFeatureFlag(t *testing.T) {
	ctx := &snapshotBuildContext{
		postBySlug: map[string]PostRecord{
			"self-hosted-edgedns-research": {
				ID:          "post-1",
				Title:       "Self Hosted EdgeDNS Research JA",
				Slug:        "self-hosted-edgedns-research",
				Body:        "<p>Body</p>",
				PublishedAt: "2026-03-22T10:11:12Z",
				Published:   true,
			},
		},
		postByID: map[string]PostRecord{
			"post-1": {
				ID:          "post-1",
				Title:       "Self Hosted EdgeDNS Research JA",
				Slug:        "self-hosted-edgedns-research",
				Body:        "<p>Body</p>",
				PublishedAt: "2026-03-22T10:11:12Z",
				Published:   true,
			},
		},
		pageByURL: map[string]PageRecord{},
		translationByKey: map[string]PostTranslationRecord{
			"zh-cn|self-hosted-edgedns-research": {
				ID:          "translation-1",
				SourcePost:  "post-1",
				Locale:      "zh-cn",
				Slug:        "self-hosted-edgedns-research",
				Title:       "Self Hosted EdgeDNS Research ZH",
				Body:        "<p>Translated body</p>",
				Published:   true,
				PublishedAt: "2026-03-23T10:11:12Z",
			},
		},
		translationsBySource: map[string][]PostTranslationRecord{
			"post-1": {
				{
					ID:          "translation-1",
					SourcePost:  "post-1",
					Locale:      "zh-cn",
					Slug:        "self-hosted-edgedns-research",
					Title:       "Self Hosted EdgeDNS Research ZH",
					Body:        "<p>Translated body</p>",
					Published:   true,
					PublishedAt: "2026-03-23T10:11:12Z",
				},
			},
		},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	rec := httptest.NewRecorder()
	err := withSnapshotBuildContext(ctx, func() error {
		ok := servePostOGImage(rec, "/og/posts/self-hosted-edgedns-research.png", SettingsRecord{
			SiteName:     "Alleycat",
			SiteLanguage: "ja",
		})
		if !ok {
			t.Fatalf("servePostOGImage should serve existing post even when feature flag is disabled")
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

func TestServePostOGImageUsesSourceLocaleForLocalizedSourceRoute(t *testing.T) {
	ctx := &snapshotBuildContext{
		postBySlug: map[string]PostRecord{
			"starlinkconoha-vpstcp": {
				ID:          "post-1",
				Title:       "Starlink ConoHa VPS TCP",
				Slug:        "starlinkconoha-vpstcp",
				Body:        "<p>Body</p>",
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

	rec := httptest.NewRecorder()
	err := withSnapshotBuildContext(ctx, func() error {
		ok := servePostOGImage(rec, "/og/ja/posts/starlinkconoha-vpstcp.png", SettingsRecord{
			SiteName:                "Alleycat",
			SiteLanguage:            "ja",
			TranslationSourceLocale: "ja",
			TranslationLocales:      "en,zh-cn",
		})
		if !ok {
			t.Fatalf("servePostOGImage should resolve source locale route using source post")
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

func TestDrawStringNoPanicRecovers(t *testing.T) {
	t.Parallel()

	drawer := &font.Drawer{
		Dst:  image.NewRGBA(image.Rect(0, 0, 10, 10)),
		Src:  image.NewUniform(color.Black),
		Face: panicFace{Face: basicfont.Face7x13},
		Dot:  fixed.P(0, 8),
	}
	if drawStringNoPanic(drawer, "boom") {
		t.Fatal("drawStringNoPanic should return false when DrawString panics")
	}
}

type panicFace struct {
	font.Face
}

func (p panicFace) Glyph(dot fixed.Point26_6, r rune) (
	dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool,
) {
	panic("glyph panic")
}
