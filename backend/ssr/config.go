package main

import (
	"os"
	"path/filepath"
	"regexp"
)

var (
	pbURL            = getEnv("PB_URL", "http://127.0.0.1:8090")
	adminURL         = getEnv("ADMIN_URL", "http://admin:5173")
	publicDir        = getEnv("PUBLIC_DIR", "/public")
	defaultPublicDir = getEnv("DEFAULT_PUBLIC_DIR", "/default-public-asset")
	activePublicDir  = resolvePublicDir()
	listenAddr       = getEnv("LISTEN_ADDR", ":5173")
)

var headingRe = regexp.MustCompile(`(?is)<h([23])([^>]*)>(.*?)</h[23]>`)
var mediaFileRe = regexp.MustCompile("(?i)(?:https?://[^\"'\\s)]+)?/api/files/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/([^\"'\\s)]+)")

func resolvePublicDir() string {
	if hasPublicAssets(publicDir) {
		return publicDir
	}
	if hasPublicAssets(defaultPublicDir) {
		return defaultPublicDir
	}
	return publicDir
}

func hasPublicAssets(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		return true
	}
	return false
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func cleanPath(path string) string {
	clean := filepath.Clean(path)
	if clean == "." {
		return "/"
	}
	return clean
}
