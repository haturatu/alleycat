package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsLocalizedPostPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "/ja/posts/hello/", want: true},
		{path: "/en-us/posts/hello/", want: true},
		{path: "/posts/hello/", want: false},
		{path: "/ja/archive/hello/", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := isLocalizedPostPath(tt.path); got != tt.want {
				t.Fatalf("isLocalizedPostPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRouteHandlerRejectsAdminPath(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	rec := httptest.NewRecorder()

	routeHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouteHandlerRevalidateRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/__internal/revalidate", nil)
	rec := httptest.NewRecorder()

	routeHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRouteHandlerFeedIgnoresStaleSnapshotWhenDisabled(t *testing.T) {
	root := t.TempDir()
	target, err := snapshotFilePath(root, "/feed.xml")
	if err != nil {
		t.Fatalf("snapshotFilePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	prevSnapshot := getPrerenderedSnapshotDir()
	setPrerenderedSnapshotDir(root)
	t.Cleanup(func() {
		prerenderedSnapshot.mu.Lock()
		prerenderedSnapshot.dir = prevSnapshot
		prerenderedSnapshot.mu.Unlock()
	})

	prevCache := settingsCache.entry
	settingsCache.mu.Lock()
	settingsCache.entry = settingsCacheEntry{
		expiresAt: time.Now().Add(time.Minute),
		value: SettingsRecord{
			SiteName:       "Alleycat",
			EnableFeedXML:  false,
			EnableFeedJSON: false,
		},
	}
	settingsCache.mu.Unlock()
	t.Cleanup(func() {
		settingsCache.mu.Lock()
		settingsCache.entry = prevCache
		settingsCache.mu.Unlock()
	})

	req := httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	rec := httptest.NewRecorder()

	routeHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouteHandlerLocalizedPostIgnoresStaleSnapshotWhenLocaleDisabled(t *testing.T) {
	root := t.TempDir()
	target, err := snapshotFilePath(root, "/fr/posts/bonjour/")
	if err != nil {
		t.Fatalf("snapshotFilePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	prevSnapshot := getPrerenderedSnapshotDir()
	setPrerenderedSnapshotDir(root)
	t.Cleanup(func() {
		prerenderedSnapshot.mu.Lock()
		prerenderedSnapshot.dir = prevSnapshot
		prerenderedSnapshot.mu.Unlock()
	})

	prevCache := settingsCache.entry
	settingsCache.mu.Lock()
	settingsCache.entry = settingsCacheEntry{
		expiresAt: time.Now().Add(time.Minute),
		value: SettingsRecord{
			SiteName:                "Alleycat",
			TranslationLocales:      "en,ja",
			SiteLanguage:            "ja",
			TranslationSourceLocale: "ja",
		},
	}
	settingsCache.mu.Unlock()
	t.Cleanup(func() {
		settingsCache.mu.Lock()
		settingsCache.entry = prevCache
		settingsCache.mu.Unlock()
	})

	req := httptest.NewRequest(http.MethodGet, "/fr/posts/bonjour/", nil)
	rec := httptest.NewRecorder()

	routeHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServePrerenderedSnapshotIgnoresConditionalCacheHeaders(t *testing.T) {
	root := t.TempDir()
	target, err := snapshotFilePath(root, "/archive/")
	if err != nil {
		t.Fatalf("snapshotFilePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(target, []byte("fresh snapshot"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	prevSnapshot := getPrerenderedSnapshotDir()
	setPrerenderedSnapshotDir(root)
	t.Cleanup(func() {
		prerenderedSnapshot.mu.Lock()
		prerenderedSnapshot.dir = prevSnapshot
		prerenderedSnapshot.mu.Unlock()
	})

	req := httptest.NewRequest(http.MethodGet, "/archive/", nil)
	req.Header.Set("If-Modified-Since", time.Now().UTC().Format(http.TimeFormat))
	rec := httptest.NewRecorder()

	if !servePrerenderedSnapshot(rec, req, "/archive/") {
		t.Fatal("servePrerenderedSnapshot returned false")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "fresh snapshot" {
		t.Fatalf("body = %q, want %q", body, "fresh snapshot")
	}
	if cache := rec.Header().Get("Cache-Control"); cache != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", cache, "no-store")
	}
}

func TestWriteFeedFilesRemovesDisabledSnapshotRoutes(t *testing.T) {
	root := t.TempDir()
	for _, route := range []string{"/feed.xml", "/feed.json"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", route, err)
		}
		if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", route, err)
		}
	}

	if err := writeFeedFiles(root, SettingsRecord{}); err != nil {
		t.Fatalf("writeFeedFiles: %v", err)
	}

	for _, route := range []string{"/feed.xml", "/feed.json"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("expected %q snapshot to be removed, err=%v", route, err)
		}
	}
}

func TestWriteLocalizedSitemapFilesRemovesDisabledLocaleSnapshot(t *testing.T) {
	root := t.TempDir()
	target, err := snapshotFilePath(root, "/sitemap-fr.xml")
	if err != nil {
		t.Fatalf("snapshotFilePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	settings := SettingsRecord{
		SiteURL:            "https://example.com",
		TranslationLocales: "en,ja",
	}
	translation := &PostTranslationRecord{Locale: "fr"}
	if err := writeLocalizedSitemapFiles(root, settings, translation, nil); err != nil {
		t.Fatalf("writeLocalizedSitemapFiles: %v", err)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected localized sitemap snapshot to be removed, err=%v", err)
	}
}

func TestWithRequestSiteURLUsesRequestHostWhenSiteURLMissing(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/posts/hello/", nil)
	req.Host = "soulminingrig.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	settings := withRequestSiteURL(SettingsRecord{}, req)
	if settings.SiteURL != "https://soulminingrig.com" {
		t.Fatalf("withRequestSiteURL SiteURL = %q", settings.SiteURL)
	}
}

func TestWithRequestSiteURLPreservesConfiguredSiteURL(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/posts/hello/", nil)
	req.Host = "soulminingrig.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	settings := withRequestSiteURL(SettingsRecord{SiteURL: "https://example.com"}, req)
	if settings.SiteURL != "https://example.com" {
		t.Fatalf("withRequestSiteURL should preserve configured SiteURL, got %q", settings.SiteURL)
	}
}
