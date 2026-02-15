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
)

var localePathPattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/admin/", adminProxy())
	mux.Handle("/", http.HandlerFunc(routeHandler))

	log.Printf("SSR server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
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

	settings := getSettings()
	previewTheme := strings.TrimSpace(r.URL.Query().Get("theme"))
	if previewTheme != "" {
		settings.Theme = previewTheme
	}

	if path == "/" {
		html := renderHome(settings)
		writeHTML(w, html)
		return
	}

	if path == "/archive" {
		html := renderArchive("/archive/", settings)
		writeHTML(w, html)
		return
	}

	if path == "/feed.json" {
		if !settings.EnableFeedJSON {
			http.NotFound(w, r)
			return
		}
		writeJSONFeed(w, r, settings)
		return
	}

	if path == "/feed.xml" {
		if !settings.EnableFeedXML {
			http.NotFound(w, r)
			return
		}
		writeRSSFeed(w, r, settings)
		return
	}

	if strings.HasPrefix(path, "/archive/") {
		html := renderArchive(path, settings)
		writeHTML(w, html)
		return
	}

	if strings.HasPrefix(path, "/posts/") {
		html := renderPost(path, settings)
		writeHTML(w, html)
		return
	}
	if isLocalizedPostPath(path) {
		html := renderPost(path, settings)
		writeHTML(w, html)
		return
	}

	html := renderPage(path, settings)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
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

	filePath := filepath.Join(activePublicDir, clean)
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}

	http.ServeFile(w, r, filePath)
	return true
}

func adminProxy() http.Handler {
	target, _ := url.Parse(adminURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = "localhost:5173"
	}
	return proxy
}
