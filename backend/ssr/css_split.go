package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type cssSplitEntry struct {
	modTime  time.Time
	critical string
	deferred string
}

var cssSplitCache = struct {
	mu    sync.RWMutex
	items map[string]cssSplitEntry
}{
	items: map[string]cssSplitEntry{},
}

func splitThemeStylesheetForHead(stylesPath string) (themeStyles string, criticalInline string) {
	if !strings.HasPrefix(stylesPath, "/") {
		return asyncStylesheetTag(stylesPath), ""
	}
	criticalCSS, _, ok := loadSplitStylesheet(stylesPath)
	if !ok {
		return asyncStylesheetTag(stylesPath), ""
	}
	return asyncStylesheetTag(stylesPath + "?defer=1"), fmt.Sprintf("<style data-css-split=\"critical\">%s</style>", criticalCSS)
}

func deferredStylesheetContent(stylesPath string) (string, bool) {
	_, deferred, ok := loadSplitStylesheet(stylesPath)
	if !ok {
		return "", false
	}
	return deferred, true
}

func loadSplitStylesheet(stylesPath string) (critical string, deferred string, ok bool) {
	absPath, ok := resolveStylesheetPath(stylesPath)
	if !ok {
		return "", "", false
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return "", "", false
	}

	cssSplitCache.mu.RLock()
	cached, found := cssSplitCache.items[absPath]
	cssSplitCache.mu.RUnlock()
	if found && cached.modTime.Equal(info.ModTime()) {
		return cached.critical, cached.deferred, true
	}

	raw, err := os.ReadFile(absPath)
	if err != nil {
		return "", "", false
	}

	critical, deferred = splitCSSByPriority(string(raw))
	if strings.TrimSpace(critical) == "" || strings.TrimSpace(deferred) == "" {
		return "", "", false
	}

	cssSplitCache.mu.Lock()
	cssSplitCache.items[absPath] = cssSplitEntry{
		modTime:  info.ModTime(),
		critical: critical,
		deferred: deferred,
	}
	cssSplitCache.mu.Unlock()

	return critical, deferred, true
}

func resolveStylesheetPath(stylesPath string) (string, bool) {
	if !strings.HasPrefix(stylesPath, "/") || !strings.HasSuffix(strings.ToLower(stylesPath), ".css") {
		return "", false
	}
	clean := cleanPath(stylesPath)
	if strings.Contains(clean, "..") {
		return "", false
	}
	absPath := filepath.Join(activePublicDir, clean)
	rel, err := filepath.Rel(activePublicDir, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return absPath, true
}

func splitCSSByPriority(source string) (critical string, deferred string) {
	blocks := cssBlocks(source)
	if len(blocks) == 0 {
		return "", ""
	}
	criticalParts := make([]string, 0, len(blocks))
	deferredParts := make([]string, 0, len(blocks))

	for _, block := range blocks {
		if isCriticalCSSBlock(block) {
			criticalParts = append(criticalParts, block)
		} else {
			deferredParts = append(deferredParts, block)
		}
	}

	return strings.Join(criticalParts, "\n"), strings.Join(deferredParts, "\n")
}

func cssBlocks(source string) []string {
	blocks := []string{}
	start := -1
	depth := 0
	inComment := false
	inString := byte(0)

	for i := 0; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		if i+1 < len(source) {
			next = source[i+1]
		}

		if inComment {
			if ch == '*' && next == '/' {
				inComment = false
				i++
			}
			continue
		}

		if inString != 0 {
			if ch == '\\' && next != 0 {
				i++
				continue
			}
			if ch == inString {
				inString = 0
			}
			continue
		}

		if ch == '/' && next == '*' {
			inComment = true
			i++
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = ch
			continue
		}

		if depth == 0 && start == -1 {
			if ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
				continue
			}
			start = i
		}

		if ch == '{' {
			depth++
			continue
		}
		if ch == '}' {
			if depth > 0 {
				depth--
			}
			if depth == 0 && start >= 0 {
				blocks = append(blocks, strings.TrimSpace(source[start:i+1]))
				start = -1
			}
			continue
		}
		if ch == ';' && depth == 0 && start >= 0 {
			blocks = append(blocks, strings.TrimSpace(source[start:i+1]))
			start = -1
		}
	}

	if start >= 0 && start < len(source) {
		tail := strings.TrimSpace(source[start:])
		if tail != "" {
			blocks = append(blocks, tail)
		}
	}
	return blocks
}

func isCriticalCSSBlock(block string) bool {
	lower := strings.ToLower(strings.TrimSpace(block))
	if lower == "" {
		return false
	}

	if strings.HasPrefix(lower, "@charset") || strings.HasPrefix(lower, "@namespace") {
		return true
	}
	if strings.HasPrefix(lower, "@import") || strings.HasPrefix(lower, "@font-face") {
		return false
	}

	criticalMarkers := []string{
		":root", "[data-theme", "data-theme", "html", "body", "#root", "main", "img", "nav", "header", "button",
		".navbar", ".page-header", ".top-image", ".postlist", ".post",
		".post-header", ".post-title", ".post-details", ".post-link",
		".body-post", ".body-tag", ".button", ".icon", ".theme-toggle",
		".search", ".search-form", ".search-input", ".search-submit", ".search-clear",
	}
	for _, marker := range criticalMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	return false
}
