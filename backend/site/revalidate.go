package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
)

type revalidateRequest struct {
	Collection string          `json:"collection"`
	Action     string          `json:"action"`
	Current    json.RawMessage `json:"current"`
	Original   json.RawMessage `json:"original"`
}

var snapshotMutation = struct {
	mu sync.Mutex
}{}

func handleRevalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isRevalidateAuthorized(r) {
		slog.Warn("revalidate request forbidden", "remote_addr", r.RemoteAddr)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req revalidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("revalidate request decode failed", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	slog.Info("revalidate request received", "collection", req.Collection, "action", req.Action)
	if err := applyRevalidation(req); err != nil {
		slog.Error("revalidation failed", "collection", req.Collection, "action", req.Action, "error", err)
		http.Error(w, "revalidation failed", http.StatusInternalServerError)
		return
	}
	slog.Info("revalidation completed", "collection", req.Collection, "action", req.Action)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func isRevalidateAuthorized(r *http.Request) bool {
	token := strings.TrimSpace(os.Getenv("STATIC_REGEN_TOKEN"))
	if token == "" {
		return true
	}
	return r.Header.Get("X-Regen-Token") == token
}

func applyRevalidation(req revalidateRequest) error {
	slog.Info("revalidation lock wait start", "collection", req.Collection, "action", req.Action)
	snapshotMutation.mu.Lock()
	defer snapshotMutation.mu.Unlock()
	slog.Info("revalidation lock acquired", "collection", req.Collection, "action", req.Action)

	root := getPrerenderedSnapshotDir()
	if root == "" {
		slog.Warn("revalidate requested before snapshot was ready; rebuilding whole snapshot", "collection", req.Collection, "action", req.Action)
		var err error
		root, err = buildStaticSnapshot()
		if err != nil {
			return err
		}
		setPrerenderedSnapshotDir(root)
		return nil
	}

	invalidateDerivedCaches()
	slog.Info("revalidation context build start", "collection", req.Collection, "action", req.Action, "root", root)
	ctx, err := newSnapshotBuildContext()
	if err != nil {
		slog.Error("revalidation context build failed", "collection", req.Collection, "action", req.Action, "error", err)
		return err
	}
	slog.Info("revalidation context build completed", "collection", req.Collection, "action", req.Action)

	return withSnapshotBuildContext(ctx, func() error {
		switch req.Collection {
		case "settings":
			slog.Info("revalidate mode selected", "mode", "full", "collection", req.Collection, "action", req.Action)
			return rebuildWholeSnapshot()
		case "pages":
			slog.Info("revalidate mode selected", "mode", "page", "collection", req.Collection, "action", req.Action)
			return revalidatePage(root, req)
		case "posts":
			slog.Info("revalidate mode selected", "mode", "post", "collection", req.Collection, "action", req.Action)
			return revalidatePost(root, req)
		case "post_translations":
			slog.Info("revalidate mode selected", "mode", "translation", "collection", req.Collection, "action", req.Action)
			return revalidateTranslation(root, req)
		default:
			slog.Warn("revalidate skipped for unsupported collection", "collection", req.Collection, "action", req.Action)
			return nil
		}
	})
}

func rebuildWholeSnapshot() error {
	root, err := buildStaticSnapshot()
	if err != nil {
		return err
	}
	setPrerenderedSnapshotDir(root)
	return nil
}

func revalidatePage(root string, req revalidateRequest) error {
	current := decodePageRecord(req.Current)
	original := decodePageRecord(req.Original)
	settings := getSettings()
	slog.Info("revalidate page start", "action", req.Action, "current_url", valueOrEmptyPageURL(current), "original_url", valueOrEmptyPageURL(original))

	if original != nil && strings.TrimSpace(original.URL) != "" {
		slog.Info("revalidate page remove original route", "route", original.URL)
		if err := removeSnapshotRoute(root, original.URL); err != nil {
			return err
		}
	}
	if current != nil && current.Published && strings.TrimSpace(current.URL) != "" {
		slog.Info("revalidate page render current route", "route", current.URL)
		html, ok := renderPageFromRecord(current, settings)
		if ok {
			if err := writeSnapshotRoute(root, current.URL, []byte(html)); err != nil {
				return err
			}
		}
	}

	slog.Info("revalidate page sitemap write start")
	return writeSitemapFiles(root, settings)
}

func revalidatePost(root string, req revalidateRequest) error {
	current := decodePostRecord(req.Current)
	original := decodePostRecord(req.Original)
	settings := getSettings()
	slog.Info("revalidate post start", "action", req.Action, "current_id", valueOrEmptyPostID(current), "current_slug", valueOrEmptyPostSlug(current), "original_id", valueOrEmptyPostID(original), "original_slug", valueOrEmptyPostSlug(original))

	if original != nil && strings.TrimSpace(original.Slug) != "" {
		slog.Info("revalidate post remove original route", "route", "/posts/"+original.Slug+"/")
		if err := removeSnapshotRoute(root, "/posts/"+url.PathEscape(original.Slug)+"/"); err != nil {
			return err
		}
	}

	if current != nil && current.Published && strings.TrimSpace(current.Slug) != "" {
		slog.Info("revalidate post render current route", "route", "/posts/"+current.Slug+"/")
		html, ok := renderPostFromInput(&postRenderInput{
			path:   "/posts/" + current.Slug + "/",
			locale: "",
			slug:   current.Slug,
			post:   current,
		}, settings)
		if ok {
			if err := writeSnapshotRoute(root, "/posts/"+url.PathEscape(current.Slug)+"/", []byte(html)); err != nil {
				return err
			}
		}
	}

	slog.Info("revalidate post family start")
	if err := revalidateSourcePostFamily(root, settings, current, original); err != nil {
		return err
	}
	impact := analyzePostImpact(current, original, settings)
	slog.Info("revalidate post impact analyzed", "home", impact.home, "main_archive", impact.mainArchive, "feed", impact.feed, "sitemap", impact.sitemap, "tag_routes", len(impact.tagArchives), "category_routes", len(impact.categoryDirs))
	if impact.home || impact.mainArchive {
		slog.Info("revalidate post home/archive start")
		if err := revalidateHomeAndArchives(root, settings, current, original, impact); err != nil {
			return err
		}
	}
	if impact.feed {
		slog.Info("revalidate post feed write start")
		if err := writeFeedFiles(root, settings); err != nil {
			return err
		}
	}
	if impact.sitemap {
		slog.Info("revalidate post sitemap write start")
		if err := writeSitemapFiles(root, settings); err != nil {
			return err
		}
	}
	slog.Info("revalidate post completed")
	return nil
}

func revalidateTranslation(root string, req revalidateRequest) error {
	current := decodeTranslationRecord(req.Current)
	original := decodeTranslationRecord(req.Original)
	settings := getSettings()
	slog.Info("revalidate translation start", "action", req.Action, "current_locale", valueOrEmptyTranslationLocale(current), "current_slug", valueOrEmptyTranslationSlug(current), "original_locale", valueOrEmptyTranslationLocale(original), "original_slug", valueOrEmptyTranslationSlug(original))

	if original != nil && strings.TrimSpace(original.Locale) != "" && strings.TrimSpace(original.Slug) != "" {
		slog.Info("revalidate translation remove original route", "route", "/"+normalizeLocale(original.Locale)+"/posts/"+original.Slug+"/")
		if err := removeSnapshotRoute(root, "/"+url.PathEscape(normalizeLocale(original.Locale))+"/posts/"+url.PathEscape(original.Slug)+"/"); err != nil {
			return err
		}
	}

	if current != nil && current.Published && strings.TrimSpace(current.Locale) != "" && strings.TrimSpace(current.Slug) != "" && isEnabledTranslationLocale(settings, current.Locale) {
		slog.Info("revalidate translation render current route", "route", "/"+normalizeLocale(current.Locale)+"/posts/"+current.Slug+"/")
		post := translationToPost(*current)
		html, ok := renderPostFromInput(&postRenderInput{
			path:        "/" + current.Locale + "/posts/" + current.Slug + "/",
			locale:      normalizeLocale(current.Locale),
			slug:        current.Slug,
			post:        &post,
			translation: current,
		}, settings)
		if ok {
			if err := writeSnapshotRoute(root, "/"+url.PathEscape(normalizeLocale(current.Locale))+"/posts/"+url.PathEscape(current.Slug)+"/", []byte(html)); err != nil {
				return err
			}
		}
	}

	slog.Info("revalidate translation context start")
	if err := revalidateTranslationContext(root, settings, current, original); err != nil {
		return err
	}
	slog.Info("revalidate translation sitemap write start")
	return writeLocalizedSitemapFiles(root, settings, current, original)
}

type postRevalidationImpact struct {
	home         bool
	mainArchive  bool
	feed         bool
	sitemap      bool
	tagArchives  []string
	categoryDirs []string
}

func revalidateHomeAndArchives(root string, settings SettingsRecord, current, original *PostRecord, impact postRevalidationImpact) error {
	if impact.home {
		slog.Info("revalidate home write start")
		if err := writeSnapshotRoute(root, "/", []byte(renderHome(settings))); err != nil {
			return err
		}
	}
	if impact.mainArchive {
		slog.Info("revalidate main archive export start", "route", "/archive/")
		if err := exportArchiveSeries(root, "/archive/", "published = true", settings); err != nil {
			return err
		}
	}
	for _, route := range append(append([]string(nil), impact.tagArchives...), impact.categoryDirs...) {
		slog.Info("revalidate archive route rebuild start", "route", route)
		if err := rebuildArchiveRoute(root, settings, route); err != nil {
			return err
		}
	}
	return nil
}

func analyzePostImpact(current, original *PostRecord, settings SettingsRecord) postRevalidationImpact {
	impact := postRevalidationImpact{}
	wasPublished := snapshotPublishedPost(original)
	isPublished := snapshotPublishedPost(current)
	if !wasPublished && !isPublished {
		return impact
	}
	impact.home = true
	impact.mainArchive = true
	impact.feed = true
	impact.sitemap = true
	impact.tagArchives = collectTagArchiveRoutes(current, original)
	impact.categoryDirs = collectCategoryArchiveRoutes(current, original)
	return impact
}

func snapshotPublishedPost(item *PostRecord) bool {
	return item != nil && item.Published && strings.TrimSpace(item.Slug) != ""
}

func collectTagArchiveRoutes(items ...*PostRecord) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range items {
		if item == nil {
			continue
		}
		for _, tag := range parseTags(item.Tags) {
			route := "/archive/" + url.PathEscape(tag) + "/"
			if _, ok := seen[route]; ok {
				continue
			}
			seen[route] = struct{}{}
			out = append(out, route)
		}
	}
	return out
}

func collectCategoryArchiveRoutes(items ...*PostRecord) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range items {
		if item == nil {
			continue
		}
		category := strings.TrimSpace(item.Category)
		if category == "" {
			continue
		}
		route := "/archive/category/" + url.PathEscape(category) + "/"
		if _, ok := seen[route]; ok {
			continue
		}
		seen[route] = struct{}{}
		out = append(out, route)
	}
	return out
}

func writeFeedFiles(root string, settings SettingsRecord) error {
	if settings.EnableFeedXML {
		if err := exportRecordedRoute(root, "/feed.xml", prerenderFeedXML(settings)); err != nil {
			return err
		}
	} else if err := removeSnapshotRoute(root, "/feed.xml"); err != nil {
		return err
	}
	if settings.EnableFeedJSON {
		return exportRecordedRoute(root, "/feed.json", prerenderFeedJSON(settings))
	}
	return removeSnapshotRoute(root, "/feed.json")
}

func writeSitemapFiles(root string, settings SettingsRecord) error {
	if err := exportRecordedRoute(root, "/robots.txt", prerenderRobots(settings)); err != nil {
		return err
	}
	if err := exportRecordedRoute(root, "/sitemap.xml", prerenderSitemap(settings, "")); err != nil {
		return err
	}
	for _, locale := range parseTranslationLocales(settings.TranslationLocales) {
		if err := exportRecordedRoute(root, "/sitemap-"+locale+".xml", prerenderSitemap(settings, locale)); err != nil {
			return err
		}
	}
	return nil
}

func rebuildArchiveRoute(root string, settings SettingsRecord, basePath string) error {
	if err := clearArchiveRoute(root, basePath); err != nil {
		return err
	}
	filter, ok := archiveFilterForBasePath(basePath)
	if !ok {
		return nil
	}
	return exportArchiveSeries(root, basePath, filter, settings)
}

func clearArchiveRoute(root string, basePath string) error {
	target, err := snapshotFilePath(root, basePath)
	if err != nil {
		return err
	}
	parent := filepathDir(target)
	return os.RemoveAll(parent)
}

func archiveFilterForBasePath(basePath string) (string, bool) {
	clean := cleanPath(basePath)
	parts := strings.Split(strings.Trim(clean, "/"), "/")
	if len(parts) == 1 && parts[0] == "archive" {
		return "published = true", true
	}
	if len(parts) == 2 && parts[0] == "archive" {
		tag := decodePathSegment(parts[1])
		return "published = true && tags ~ \"" + escapeFilter(tag) + "\"", true
	}
	if len(parts) == 3 && parts[0] == "archive" && parts[1] == "category" {
		category := decodePathSegment(parts[2])
		return "published = true && category = \"" + escapeFilter(category) + "\"", true
	}
	return "", false
}

func revalidateSourcePostFamily(root string, settings SettingsRecord, current, original *PostRecord) error {
	for _, item := range []*PostRecord{current, original} {
		if item == nil || strings.TrimSpace(item.ID) == "" {
			continue
		}
		slog.Info("revalidate source post family item start", "source_post_id", item.ID)
		if err := revalidatePostRecordAndTranslations(root, settings, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func revalidatePostRecordAndTranslations(root string, settings SettingsRecord, sourcePostID string) error {
	post := getPostByID(sourcePostID)
	if post != nil && post.Published && strings.TrimSpace(post.Slug) != "" {
		slog.Info("revalidate source post render start", "source_post_id", sourcePostID, "route", "/posts/"+post.Slug+"/")
		html, ok := renderPostFromInput(&postRenderInput{
			path:   "/posts/" + post.Slug + "/",
			locale: "",
			slug:   post.Slug,
			post:   post,
		}, settings)
		if ok {
			if err := writeSnapshotRoute(root, "/posts/"+url.PathEscape(post.Slug)+"/", []byte(html)); err != nil {
				return err
			}
		}
	}

	for _, item := range getPostTranslationsBySource(sourcePostID) {
		locale := normalizeLocale(item.Locale)
		if locale == "" || strings.TrimSpace(item.Slug) == "" || !isEnabledTranslationLocale(settings, locale) {
			continue
		}
		slog.Info("revalidate translation render start", "source_post_id", sourcePostID, "locale", locale, "route", "/"+locale+"/posts/"+item.Slug+"/")
		post := translationToPost(item)
		html, ok := renderPostFromInput(&postRenderInput{
			path:        "/" + locale + "/posts/" + item.Slug + "/",
			locale:      locale,
			slug:        item.Slug,
			post:        &post,
			translation: &item,
		}, settings)
		if ok {
			if err := writeSnapshotRoute(root, "/"+url.PathEscape(locale)+"/posts/"+url.PathEscape(item.Slug)+"/", []byte(html)); err != nil {
				return err
			}
		}
	}
	return nil
}

func revalidateTranslationContext(root string, settings SettingsRecord, current, original *PostTranslationRecord) error {
	seen := map[string]struct{}{}
	for _, item := range []*PostTranslationRecord{current, original} {
		if item == nil {
			continue
		}
		sourceID := strings.TrimSpace(item.SourcePost)
		if sourceID == "" {
			continue
		}
		if _, ok := seen[sourceID]; ok {
			continue
		}
		seen[sourceID] = struct{}{}
		if err := revalidatePostRecordAndTranslations(root, settings, sourceID); err != nil {
			return err
		}
	}
	return nil
}

func writeLocalizedSitemapFiles(root string, settings SettingsRecord, current, original *PostTranslationRecord) error {
	slog.Info("revalidate localized robots write start")
	if err := exportRecordedRoute(root, "/robots.txt", prerenderRobots(settings)); err != nil {
		return err
	}
	locales := map[string]struct{}{}
	for _, item := range []*PostTranslationRecord{current, original} {
		if item == nil {
			continue
		}
		locale := normalizeLocale(item.Locale)
		if locale != "" {
			locales[locale] = struct{}{}
		}
	}
	for locale := range locales {
		slog.Info("revalidate localized sitemap write start", "locale", locale)
		if err := exportRecordedRoute(root, "/sitemap-"+locale+".xml", prerenderSitemap(settings, locale)); err != nil {
			return err
		}
	}
	return nil
}

func valueOrEmptyPostID(item *PostRecord) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.ID)
}

func valueOrEmptyPostSlug(item *PostRecord) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Slug)
}

func valueOrEmptyPageURL(item *PageRecord) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.URL)
}

func valueOrEmptyTranslationLocale(item *PostTranslationRecord) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Locale)
}

func valueOrEmptyTranslationSlug(item *PostTranslationRecord) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Slug)
}

func prerenderFeedXML(settings SettingsRecord) []byte {
	rec := httptest.NewRecorder()
	writeRSSFeed(rec, nil, settings)
	return rec.Body.Bytes()
}

func prerenderFeedJSON(settings SettingsRecord) []byte {
	rec := httptest.NewRecorder()
	writeJSONFeed(rec, nil, settings)
	return rec.Body.Bytes()
}

func decodePostRecord(data json.RawMessage) *PostRecord {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var out PostRecord
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return &out
}

func decodePageRecord(data json.RawMessage) *PageRecord {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var out PageRecord
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return &out
}

func decodeTranslationRecord(data json.RawMessage) *PostTranslationRecord {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var out PostTranslationRecord
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return &out
}

func removeSnapshotRoute(root, route string) error {
	target, err := snapshotFilePath(root, route)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func filepathDir(path string) string {
	if strings.HasSuffix(path, "/index.html") {
		return strings.TrimSuffix(path, "/index.html")
	}
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		return "."
	}
	return path[:lastSlash]
}
