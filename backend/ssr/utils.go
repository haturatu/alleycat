package main

import (
	"html"
	"math"
	"regexp"
	"strings"
	"time"
)

var tagStripper = regexp.MustCompile(`(?s)<[^>]*>`)

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
	if strings.TrimSpace(value) == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", value)
	}
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02")
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
