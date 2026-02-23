package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func rewriteMediaURLs(body string) string {
	matches := mediaFileRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return body
	}
	ids := make([]string, 0, len(matches))
	seenIDs := map[string]struct{}{}
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		collection := match[1]
		id := match[2]
		if collection != "media" && collection != "pbc_2708086759" {
			continue
		}
		if _, ok := seenIDs[id]; ok {
			continue
		}
		seenIDs[id] = struct{}{}
		ids = append(ids, id)
	}

	mediaByID := getMediaByIDs(ids)
	cache := map[string]string{}
	for id, media := range mediaByID {
		path := strings.TrimSpace(media.Path)
		if path == "" {
			path = strings.TrimSpace(media.Caption)
		}
		if path == "" {
			continue
		}
		cache[id] = path
	}

	return mediaFileRe.ReplaceAllStringFunc(body, func(match string) string {
		sub := mediaFileRe.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		id := sub[2]
		filename := sub[3]
		replacement := cache[id]
		if replacement == "" {
			return fmt.Sprintf("/api/files/%s/%s/%s", sub[1], id, filename)
		}
		if strings.HasPrefix(replacement, "http://") || strings.HasPrefix(replacement, "https://") {
			return replacement
		}
		if strings.HasPrefix(replacement, "/") {
			return replacement
		}
		return "/" + replacement
	})
}

func serveMediaUpload(w http.ResponseWriter, r *http.Request, clean string) bool {
	media := getMediaByPath(clean)
	if media == nil || media.File == "" {
		return false
	}
	fileURL := fmt.Sprintf("%s/api/files/media/%s/%s", pbURL, media.ID, url.PathEscape(media.File))
	resp, err := httpClient.Get(fileURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	cache := resp.Header.Get("Cache-Control")
	if cache != "" {
		w.Header().Set("Cache-Control", cache)
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	return err == nil
}
