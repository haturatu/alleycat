package site

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeedAndSitemapResponsesDisableCaching(t *testing.T) {
	t.Parallel()

	settings := SettingsRecord{
		SiteName:       "Alleycat",
		SiteURL:        "https://example.com",
		EnableFeedXML:  true,
		EnableFeedJSON: true,
	}

	tests := []struct {
		name string
		path string
		run  func(*httptest.ResponseRecorder, *http.Request)
	}{
		{
			name: "feed json",
			path: "/feed.json",
			run: func(rec *httptest.ResponseRecorder, req *http.Request) {
				writeJSONFeed(rec, req, settings)
			},
		},
		{
			name: "feed xml",
			path: "/feed.xml",
			run: func(rec *httptest.ResponseRecorder, req *http.Request) {
				writeRSSFeed(rec, req, settings)
			},
		},
		{
			name: "sitemap",
			path: "/sitemap.xml",
			run: func(rec *httptest.ResponseRecorder, req *http.Request) {
				writeSitemap(rec, req, settings)
			},
		},
		{
			name: "robots",
			path: "/robots.txt",
			run: func(rec *httptest.ResponseRecorder, req *http.Request) {
				writeRobotsTXT(rec, req, settings)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "https://example.com"+tt.path, nil)
			rec := httptest.NewRecorder()
			tt.run(rec, req)
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want %q", got, "no-store")
			}
			if got := rec.Header().Get("Pragma"); got != "no-cache" {
				t.Fatalf("Pragma = %q, want %q", got, "no-cache")
			}
			if got := rec.Header().Get("Expires"); got != "0" {
				t.Fatalf("Expires = %q, want %q", got, "0")
			}
		})
	}
}
