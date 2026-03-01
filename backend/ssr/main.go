package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var localePathPattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(routeHandler))

	log.Printf("SSR server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/admin" || strings.HasPrefix(path, "/admin/") {
		http.NotFound(w, r)
		return
	}
	if path == "/theme-status" {
		w.Header().Set("Content-Type", "application/json")
		status := `{"publicAssets":false}`
		if activePublicDir == publicDir {
			status = `{"publicAssets":true}`
		}
		_, _ = w.Write([]byte(status))
		return
	}
	if strings.HasPrefix(path, "/api/files/") {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(path, "/api/") {
		proxyPocketBase(w, r)
		return
	}
	if serveStatic(w, r) {
		return
	}

	if path == "/" {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		html := renderHome(settings)
		writeHTML(w, html)
		return
	}

	if path == "/archive" {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		html := renderArchive("/archive/", searchQuery, settings)
		writeHTML(w, html)
		return
	}

	if path == "/feed.json" {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		if !settings.EnableFeedJSON {
			http.NotFound(w, r)
			return
		}
		writeJSONFeed(w, r, settings)
		return
	}

	if path == "/feed.xml" {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		if !settings.EnableFeedXML {
			http.NotFound(w, r)
			return
		}
		writeRSSFeed(w, r, settings)
		return
	}
	if path == "/robots.txt" {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		writeRobotsTXT(w, r, settings)
		return
	}
	if path == "/sitemap.xml" {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		writeSitemap(w, r, settings)
		return
	}
	if locale, ok := extractSitemapLocale(path); ok {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
		writeLocalizedSitemap(w, r, settings, locale)
		return
	}

	if strings.HasPrefix(path, "/archive/") {
		settings := getSettings()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
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
			settings = getSettings()
		}()
		go func() {
			defer wg.Done()
			input, found = prefetchPostRenderInput(path)
		}()
		wg.Wait()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
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
			settings = getSettings()
		}()
		go func() {
			defer wg.Done()
			input, found = prefetchPostRenderInput(path)
		}()
		wg.Wait()
		previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
		if previewTheme != "" {
			settings.Theme = previewTheme
		}
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
		settings = getSettings()
	}()
	go func() {
		defer wg.Done()
		page = getPageByURL(path)
	}()
	wg.Wait()
	previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
	if previewTheme != "" {
		settings.Theme = previewTheme
	}
	html, found := renderPageFromRecord(page, settings)
	if !found {
		writeHTMLStatus(w, html, http.StatusNotFound)
		return
	}
	writeHTML(w, html)
}

func isLocalizedPostPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[1] != "posts" || strings.TrimSpace(parts[2]) == "" {
		return false
	}
	locale := normalizeLocale(parts[0])
	return locale == parts[0] && localePathPattern.MatchString(locale)
}

func proxyPocketBase(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(pbURL)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = r.URL.Path
		req.URL.RawQuery = r.URL.RawQuery
		req.Host = target.Host
	}
	proxy.ServeHTTP(w, r)
}

func writeHTML(w http.ResponseWriter, content string) {
	writeHTMLStatus(w, content, http.StatusOK)
}

func writeHTMLStatus(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(content))
}

func serveStatic(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path == "/" {
		return false
	}

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

	filePath := filepath.Join(activePublicDir, clean)
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}

	http.ServeFile(w, r, filePath)
	return true
}
