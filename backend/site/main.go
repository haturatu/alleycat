package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})))
	warmStaticSnapshotAtBoot()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(routeHandler))

	slog.Info("site server starting", "listen_addr", listenAddr, "pb_url", pbURL, "public_dir", activePublicDir, "static_export_dir", staticExportDir)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		slog.Error("site server stopped", "error", err)
		os.Exit(1)
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/admin" || strings.HasPrefix(path, "/admin/") {
		http.NotFound(w, r)
		return
	}
	if path == "/__internal/revalidate" {
		handleRevalidate(w, r)
		return
	}
	if path == "/feed.json" || path == "/feed.xml" {
		settings := requestSettings(r)
		if !isFeedRouteEnabled(path, settings) {
			http.NotFound(w, r)
			return
		}
	}
	if locale, ok := extractSitemapLocale(path); ok {
		settings := requestSettings(r)
		if !isEnabledTranslationLocale(settings, locale) {
			http.NotFound(w, r)
			return
		}
	}
	if locale, ok := extractLocalizedPostRouteLocale(path); ok {
		settings := requestSettings(r)
		if !isEnabledTranslationLocale(settings, locale) {
			writeHTMLStatus(w, renderNotFound(settings), http.StatusNotFound)
			return
		}
	}
	if serveStatic(w, r) {
		return
	}

	if path == "/" {
		settings := requestSettings(r)
		html := renderHome(settings)
		writeHTML(w, html)
		return
	}

	if path == "/archive" {
		settings := requestSettings(r)
		searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		html := renderArchive("/archive/", searchQuery, settings)
		writeHTML(w, html)
		return
	}

	if path == "/feed.json" {
		settings := requestSettings(r)
		if !settings.EnableFeedJSON {
			http.NotFound(w, r)
			return
		}
		writeJSONFeed(w, r, settings)
		return
	}

	if path == "/feed.xml" {
		settings := requestSettings(r)
		if !settings.EnableFeedXML {
			http.NotFound(w, r)
			return
		}
		writeRSSFeed(w, r, settings)
		return
	}
	if path == "/robots.txt" {
		settings := requestSettings(r)
		writeRobotsTXT(w, r, settings)
		return
	}
	if path == "/sitemap.xml" {
		settings := requestSettings(r)
		writeSitemap(w, r, settings)
		return
	}
	if locale, ok := extractSitemapLocale(path); ok {
		settings := requestSettings(r)
		writeLocalizedSitemap(w, r, settings, locale)
		return
	}
	if strings.HasPrefix(path, "/og/") {
		settings := requestSettings(r)
		if servePostOGImage(w, path, settings) {
			return
		}
		http.NotFound(w, r)
		return
	}

	if strings.HasPrefix(path, "/archive/") {
		settings := requestSettings(r)
		searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		html := renderArchive(path, searchQuery, settings)
		writeHTML(w, html)
		return
	}

	if strings.HasPrefix(path, "/posts/") {
		var settings SettingsRecord
		var input *postRenderInput
		var found bool
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			settings = withRequestSiteURL(getSettings(), r)
		}()
		go func() {
			defer wg.Done()
			input, found = prefetchPostRenderInput(path)
		}()
		wg.Wait()
		settings = withPreviewTheme(settings, r)
		if !found {
			writeHTMLStatus(w, renderNotFound(settings), http.StatusNotFound)
			return
		}
		html, ok := renderPostFromInput(input, settings)
		if !ok {
			writeHTMLStatus(w, html, http.StatusNotFound)
			return
		}
		writeHTML(w, html)
		return
	}
	if isLocalizedPostPath(path) {
		var settings SettingsRecord
		var input *postRenderInput
		var found bool
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			settings = withRequestSiteURL(getSettings(), r)
		}()
		go func() {
			defer wg.Done()
			input, found = prefetchPostRenderInput(path)
		}()
		wg.Wait()
		settings = withPreviewTheme(settings, r)
		if !found {
			writeHTMLStatus(w, renderNotFound(settings), http.StatusNotFound)
			return
		}
		html, ok := renderPostFromInput(input, settings)
		if !ok {
			writeHTMLStatus(w, html, http.StatusNotFound)
			return
		}
		writeHTML(w, html)
		return
	}

	var settings SettingsRecord
	var page *PageRecord
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		settings = withRequestSiteURL(getSettings(), r)
	}()
	go func() {
		defer wg.Done()
		page = getPageByURL(path)
	}()
	wg.Wait()
	settings = withPreviewTheme(settings, r)
	html, found := renderPageFromRecord(page, settings)
	if !found {
		writeHTMLStatus(w, html, http.StatusNotFound)
		return
	}
	writeHTML(w, html)
}

func isLocalizedPostPath(path string) bool {
	_, _, ok := extractLocalizedPostRoute(path)
	return ok
}

func extractLocalizedPostRouteLocale(path string) (string, bool) {
	locale, _, ok := extractLocalizedPostRoute(path)
	return locale, ok
}

func isFeedRouteEnabled(path string, settings SettingsRecord) bool {
	switch path {
	case "/feed.xml":
		return settings.EnableFeedXML
	case "/feed.json":
		return settings.EnableFeedJSON
	default:
		return false
	}
}

func requestSettings(r *http.Request) SettingsRecord {
	return withPreviewTheme(withRequestSiteURL(getSettings(), r), r)
}

func withPreviewTheme(settings SettingsRecord, r *http.Request) SettingsRecord {
	if r == nil {
		return settings
	}
	previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
	if previewTheme != "" {
		settings.Theme = previewTheme
	}
	return settings
}

func withRequestSiteURL(settings SettingsRecord, r *http.Request) SettingsRecord {
	if normalizeSiteBaseURL(settings.SiteURL) != "" || r == nil {
		return settings
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
		return settings
	}

	settings.SiteURL = scheme + "://" + host
	return settings
}

func writeHTML(w http.ResponseWriter, content string) {
	writeHTMLStatus(w, content, http.StatusOK)
}

func writeHTMLStatus(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	_, _ = w.Write([]byte(content))
}

func serveStatic(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	clean := cleanPath(path)
	if strings.Contains(clean, "..") {
		return false
	}

	if strings.HasPrefix(clean, "/uploads/") {
		if serveMediaUpload(w, r, clean) {
			return true
		}
	}
	if strings.HasSuffix(strings.ToLower(clean), ".css") && r.URL.Query().Get("defer") == "1" {
		if deferredCSS, ok := deferredStylesheetContent(clean); ok {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(deferredCSS))
			return true
		}
	}

	if r.URL.RawQuery == "" {
		if servePrerenderedSnapshot(w, r, clean) {
			return true
		}
	}

	filePath := filepath.Join(activePublicDir, clean)
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}

	http.ServeFile(w, r, filePath)
	return true
}

func servePrerenderedSnapshot(w http.ResponseWriter, r *http.Request, clean string) bool {
	root := getPrerenderedSnapshotDir()
	if root == "" {
		return false
	}
	target, err := snapshotFilePath(root, clean)
	if err != nil {
		return false
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		return false
	}
	http.ServeFile(w, r, target)
	return true
}
