package main

import (
	"html"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

var tagStripper = regexp.MustCompile(`(?s)<[^>]*>`)
const utilCacheMaxEntries = 2048

var formatDateCache = struct {
	mu    sync.RWMutex
	items map[string]string
}{
	items: map[string]string{},
}

var parseTagsCache = struct {
	mu    sync.RWMutex
	items map[string][]string
}{
	items: map[string][]string{},
}

func escapeHTML(value string) string {
	return html.EscapeString(value)
}

func stripHTML(value string) string {
	if value == "" {
		return ""
	}
	clean := tagStripper.ReplaceAllString(value, " ")
	clean = strings.TrimSpace(clean)
	clean = strings.Join(strings.Fields(clean), " ")
	return clean
}

func formatDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	formatDateCache.mu.RLock()
	cached, ok := formatDateCache.items[trimmed]
	formatDateCache.mu.RUnlock()
	if ok {
		return cached
	}

	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", trimmed)
	}
	result := trimmed
	if err != nil {
		setFormatDateCache(trimmed, result)
		return result
	}
	result = parsed.Format("2006-01-02")
	setFormatDateCache(trimmed, result)
	return result
}

func buildExcerpt(value string, length int) string {
	text := stripHTML(value)
	if length <= 0 {
		length = 160
	}
	runes := []rune(text)
	if len(runes) <= length {
		return text
	}
	return string(runes[:length]) + "..."
}

func parseTags(value string) []string {
	if value == "" {
		return nil
	}

	parseTagsCache.mu.RLock()
	cached, ok := parseTagsCache.items[value]
	parseTagsCache.mu.RUnlock()
	if ok {
		return append([]string(nil), cached...)
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	setParseTagsCache(value, result)
	return result
}

func calcReadTime(body string) int {
	length := len([]rune(stripHTML(body)))
	minutes := int(math.Ceil(float64(length) / 700.0))
	if minutes < 1 {
		return 1
	}
	return minutes
}

func normalizeURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeLocale(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(trimmed, "_", "-")
}

func setFormatDateCache(key string, value string) {
	formatDateCache.mu.Lock()
	if len(formatDateCache.items) >= utilCacheMaxEntries {
		formatDateCache.items = map[string]string{}
	}
	formatDateCache.items[key] = value
	formatDateCache.mu.Unlock()
}

func setParseTagsCache(key string, value []string) {
	parseTagsCache.mu.Lock()
	if len(parseTagsCache.items) >= utilCacheMaxEntries {
		parseTagsCache.items = map[string][]string{}
	}
	parseTagsCache.items[key] = append([]string(nil), value...)
	parseTagsCache.mu.Unlock()
}
