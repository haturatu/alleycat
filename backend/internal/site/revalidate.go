package site

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"alleycat-backend/internal/dag"
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

	return nil
}

func revalidatePost(root string, req revalidateRequest) error {
	current := decodePostRecord(req.Current)
	original := decodePostRecord(req.Original)
	slog.Info("revalidate post start", "action", req.Action, "current_id", valueOrEmptyPostID(current), "current_slug", valueOrEmptyPostSlug(current), "original_id", valueOrEmptyPostID(original), "original_slug", valueOrEmptyPostSlug(original))

	if original != nil && strings.TrimSpace(original.Slug) != "" {
		slog.Info("revalidate post remove original route", "route", "/posts/"+original.Slug+"/")
		if err := removeSnapshotRoute(root, "/posts/"+url.PathEscape(original.Slug)+"/"); err != nil {
			return err
		}
	}

	if !dagPostRouteRevalidationEnabled() && current != nil && current.Published && strings.TrimSpace(current.Slug) != "" {
		settings := currentRevalidationSettings()
		route := "/posts/" + current.Slug + "/"
		slog.Info("revalidate post render current route", "route", route)
		html, ok, err := renderLegacyPostRoute(func() (string, bool) {
			return renderPostFromInput(&postRenderInput{
				path:   route,
				locale: "",
				slug:   current.Slug,
				post:   current,
			}, settings)
		})
		if err != nil {
			return err
		}
		if ok {
			if err := writeSnapshotRoute(root, "/posts/"+url.PathEscape(current.Slug)+"/", html); err != nil {
				return err
			}
		}
	}
	if err := revalidateDAGAffectedPostRoutes(root, dagChangedKeysForPost(current, original), dagExtraAffectedBaseRoutes(current, original)); err != nil {
		return err
	}

	slog.Info("revalidate post family start")
	if err := revalidateSourcePostFamily(root, current, original); err != nil {
		return err
	}
	if err := revalidateAdjacentPostContext(root, current, original); err != nil {
		return err
	}
	impact := analyzePostImpact(current, original)
	slog.Info("revalidate post impact analyzed", "home", impact.home, "main_archive", impact.mainArchive, "tag_routes", len(impact.tagArchives), "category_routes", len(impact.categoryDirs))
	if impact.home || impact.mainArchive {
		settings := currentRevalidationSettings()
		slog.Info("revalidate post home/archive start")
		if err := revalidateHomeAndArchives(root, settings, current, original, impact); err != nil {
			return err
		}
	}
	slog.Info("revalidate post completed")
	return nil
}

func revalidateTranslation(root string, req revalidateRequest) error {
	current := decodeTranslationRecord(req.Current)
	original := decodeTranslationRecord(req.Original)
	slog.Info("revalidate translation start", "action", req.Action, "current_locale", valueOrEmptyTranslationLocale(current), "current_slug", valueOrEmptyTranslationSlug(current), "original_locale", valueOrEmptyTranslationLocale(original), "original_slug", valueOrEmptyTranslationSlug(original))

	if original != nil && strings.TrimSpace(original.Locale) != "" && strings.TrimSpace(original.Slug) != "" {
		slog.Info("revalidate translation remove original route", "route", "/"+normalizeLocale(original.Locale)+"/posts/"+original.Slug+"/")
		if err := removeSnapshotRoute(root, "/"+url.PathEscape(normalizeLocale(original.Locale))+"/posts/"+url.PathEscape(original.Slug)+"/"); err != nil {
			return err
		}
	}

	if !dagPostRouteRevalidationEnabled() && current != nil && current.Published && strings.TrimSpace(current.Locale) != "" && strings.TrimSpace(current.Slug) != "" {
		settings := currentRevalidationSettings()
		if isEnabledTranslationLocale(settings, current.Locale) {
			route := "/" + normalizeLocale(current.Locale) + "/posts/" + current.Slug + "/"
			slog.Info("revalidate translation render current route", "route", route)
			post := translationToPost(*current)
			html, ok, err := renderLegacyPostRoute(func() (string, bool) {
				return renderPostFromInput(&postRenderInput{
					path:        route,
					locale:      normalizeLocale(current.Locale),
					slug:        current.Slug,
					post:        &post,
					translation: current,
				}, settings)
			})
			if err != nil {
				return err
			}
			if ok {
				if err := writeSnapshotRoute(root, "/"+url.PathEscape(normalizeLocale(current.Locale))+"/posts/"+url.PathEscape(current.Slug)+"/", html); err != nil {
					return err
				}
			}
		}
	}
	if err := revalidateDAGAffectedPostRoutes(root, dagChangedKeysForTranslation(current, original), dagExtraAffectedTranslationRoutes(current, original)); err != nil {
		return err
	}

	slog.Info("revalidate translation context start")
	if err := revalidateTranslationContext(root, current, original); err != nil {
		return err
	}
	if err := revalidateAdjacentTranslationContext(root, current, original); err != nil {
		return err
	}
	return nil
}

type postRevalidationImpact struct {
	home         bool
	mainArchive  bool
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

func analyzePostImpact(current, original *PostRecord) postRevalidationImpact {
	impact := postRevalidationImpact{}
	wasPublished := snapshotPublishedPost(original)
	isPublished := snapshotPublishedPost(current)
	if !wasPublished && !isPublished {
		return impact
	}
	impact.home = true
	impact.mainArchive = true
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
	route := parseArchiveRoute(basePath)
	if route.basePath != cleanPath(basePath) {
		return "", false
	}
	return route.filter, true
}

func revalidateSourcePostFamily(root string, current, original *PostRecord) error {
	if dagPostRouteRevalidationEnabled() {
		return nil
	}
	for _, item := range []*PostRecord{current, original} {
		if item == nil || strings.TrimSpace(item.ID) == "" {
			continue
		}
		slog.Info("revalidate source post family item start", "source_post_id", item.ID)
		if err := revalidatePostRecordAndTranslations(root, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func revalidatePostRecordAndTranslations(root string, sourcePostID string) error {
	settings := getSettings()
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		settings = snapshot.settings
	}
	post := getPostByID(sourcePostID)
	if post != nil && post.Published && strings.TrimSpace(post.Slug) != "" {
		route := "/posts/" + post.Slug + "/"
		slog.Info("revalidate source post render start", "source_post_id", sourcePostID, "route", route)
		html, ok, err := renderLegacyPostRoute(func() (string, bool) {
			return renderPostFromInput(&postRenderInput{
				path:   route,
				locale: "",
				slug:   post.Slug,
				post:   post,
			}, settings)
		})
		if err != nil {
			return err
		}
		if ok {
			if err := writeSnapshotRoute(root, "/posts/"+url.PathEscape(post.Slug)+"/", html); err != nil {
				return err
			}
		}
	}

	for _, item := range getEnabledTranslationsBySource(sourcePostID, settings) {
		locale := normalizeLocale(item.Locale)
		if locale == "" || strings.TrimSpace(item.Slug) == "" {
			continue
		}
		route := "/" + locale + "/posts/" + item.Slug + "/"
		slog.Info("revalidate translation render start", "source_post_id", sourcePostID, "locale", locale, "route", route)
		post := translationToPost(item)
		html, ok, err := renderLegacyPostRoute(func() (string, bool) {
			return renderPostFromInput(&postRenderInput{
				path:        route,
				locale:      locale,
				slug:        item.Slug,
				post:        &post,
				translation: &item,
			}, settings)
		})
		if err != nil {
			return err
		}
		if ok {
			if err := writeSnapshotRoute(root, "/"+url.PathEscape(locale)+"/posts/"+url.PathEscape(item.Slug)+"/", html); err != nil {
				return err
			}
		}
	}
	return nil
}

func dagPostRouteRevalidationEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SITE_DAG_POST_ROUTES"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func renderLegacyPostRoute(fallback func() (string, bool)) ([]byte, bool, error) {
	html, ok := fallback()
	return []byte(html), ok, nil
}

func currentRevalidationSettings() SettingsRecord {
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		return snapshot.settings
	}
	return getSettings()
}

func dagAffectedRouteKeysFromChanged(ctx *dag.ResolveContext, changed []dag.NodeKey) []dag.NodeKey {
	if ctx == nil {
		return nil
	}
	affected := ctx.AffectedFrom(changed)
	out := make([]dag.NodeKey, 0, len(affected))
	for _, key := range affected {
		if key.Kind != nodeRoute {
			continue
		}
		out = append(out, key)
	}
	return out
}

func dagChangedKeysForPost(current, original *PostRecord) []dag.NodeKey {
	keys := make([]dag.NodeKey, 0, 2)
	for _, item := range []*PostRecord{current, original} {
		if item == nil || strings.TrimSpace(item.Slug) == "" {
			continue
		}
		keys = appendUniqueDAGNodeKey(keys, postBySlugNodeKey("", strings.TrimSpace(item.Slug)))
	}
	return keys
}

func dagChangedKeysForTranslation(current, original *PostTranslationRecord) []dag.NodeKey {
	keys := make([]dag.NodeKey, 0, 2)
	for _, item := range []*PostTranslationRecord{current, original} {
		if item == nil || strings.TrimSpace(item.Slug) == "" || normalizeLocale(item.Locale) == "" {
			continue
		}
		keys = appendUniqueDAGNodeKey(keys, postBySlugNodeKey(normalizeLocale(item.Locale), strings.TrimSpace(item.Slug)))
	}
	return keys
}

func appendUniqueDAGNodeKey(items []dag.NodeKey, candidate dag.NodeKey) []dag.NodeKey {
	for _, item := range items {
		if item == candidate {
			return items
		}
	}
	return append(items, candidate)
}

type dagExtraRoute struct {
	Key     dag.NodeKey
	Reasons []string
}

func revalidateDAGAffectedPostRoutes(root string, changed []dag.NodeKey, extraRoutes []dagExtraRoute) error {
	if !dagPostRouteRevalidationEnabled() || (len(changed) == 0 && len(extraRoutes) == 0) {
		return nil
	}
	snapshot := currentSnapshotBuildContext()
	if snapshot == nil {
		return nil
	}

	engine := newSiteDAGEngine()
	resolveCtx := engine.NewContext()
	for _, route := range snapshot.postRouteKeys() {
		if _, err := engine.Resolve(resolveCtx, route); err != nil {
			return err
		}
	}

	routes := dagAffectedRouteKeysFromChanged(resolveCtx, changed)
	extraReasons := map[dag.NodeKey][]string{}
	for _, route := range extraRoutes {
		routes = appendUniqueDAGNodeKey(routes, route.Key)
		extraReasons[route.Key] = append([]string(nil), route.Reasons...)
	}

	for _, route := range routes {
		slog.Info("revalidate DAG affected route", "route", route.ID, "reasons", dagAffectedRouteReasons(resolveCtx, changed, route, extraReasons[route]))
		value, err := engine.Resolve(resolveCtx, route)
		if err != nil {
			return err
		}
		rendered, ok := value.(routeValue)
		if !ok {
			continue
		}
		if err := writeSnapshotRoute(root, rendered.Path, append([]byte(nil), rendered.Body...)); err != nil {
			return err
		}
	}
	return nil
}

func dagExtraAffectedBaseRoutes(current, original *PostRecord) []dagExtraRoute {
	snapshot := currentSnapshotBuildContext()
	if snapshot == nil {
		return nil
	}
	return snapshot.adjacentBaseRouteKeysForPosts(current, original)
}

func dagExtraAffectedTranslationRoutes(current, original *PostTranslationRecord) []dagExtraRoute {
	snapshot := currentSnapshotBuildContext()
	if snapshot == nil {
		return nil
	}
	return snapshot.adjacentLocalizedRouteKeysForTranslations(current, original)
}

func dagAffectedRouteReasons(ctx *dag.ResolveContext, changed []dag.NodeKey, route dag.NodeKey, extras []string) []string {
	if ctx == nil {
		return append([]string(nil), extras...)
	}
	out := append([]string(nil), extras...)
	for _, key := range changed {
		path := ctx.ExplainPath(key, route)
		if len(path) == 0 {
			continue
		}
		parts := make([]string, 0, len(path))
		for _, item := range path {
			parts = append(parts, item.String())
		}
		out = append(out, strings.Join(parts, " -> "))
	}
	return out
}

func revalidateTranslationContext(root string, current, original *PostTranslationRecord) error {
	settings := getSettings()
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		settings = snapshot.settings
	}
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
		if !dagPostRouteRevalidationEnabled() {
			if err := revalidatePostRecordAndTranslations(root, sourceID); err != nil {
				return err
			}
		}
		sourcePost := getPostByID(sourceID)
		impact := analyzePostImpact(sourcePost, sourcePost)
		if !impact.home && !impact.mainArchive && len(impact.tagArchives) == 0 && len(impact.categoryDirs) == 0 {
			continue
		}
		slog.Info("revalidate translation archive impact analyzed", "source_post_id", sourceID, "home", impact.home, "main_archive", impact.mainArchive, "tag_routes", len(impact.tagArchives), "category_routes", len(impact.categoryDirs))
		if err := revalidateHomeAndArchives(root, settings, sourcePost, sourcePost, impact); err != nil {
			return err
		}
	}
	return nil
}

func revalidateAdjacentPostContext(root string, current, original *PostRecord) error {
	if dagPostRouteRevalidationEnabled() {
		return nil
	}
	ctx := currentSnapshotBuildContext()
	if ctx == nil {
		return nil
	}

	seen := map[string]struct{}{}
	references := []*PostRecord{current, original}
	for _, ref := range references {
		if ref == nil {
			continue
		}
		newer, older := ctx.getAdjacentPostsAroundReferenceInLocale(ref, "")
		for _, candidate := range []*PostRecord{newer, older} {
			if candidate == nil || strings.TrimSpace(candidate.Slug) == "" {
				continue
			}
			slug := strings.TrimSpace(candidate.Slug)
			if _, ok := seen[slug]; ok {
				continue
			}
			seen[slug] = struct{}{}
			if err := revalidatePostRecordAndTranslations(root, strings.TrimSpace(candidate.ID)); err != nil {
				return err
			}
		}
	}
	return nil
}

func revalidateAdjacentTranslationContext(root string, current, original *PostTranslationRecord) error {
	if dagPostRouteRevalidationEnabled() {
		return nil
	}
	ctx := currentSnapshotBuildContext()
	if ctx == nil {
		return nil
	}

	seen := map[string]struct{}{}
	references := []*PostTranslationRecord{current, original}
	for _, ref := range references {
		if ref == nil {
			continue
		}
		locale := normalizeLocale(ref.Locale)
		if locale == "" {
			continue
		}
		post := translationToPost(*ref)
		newer, older := ctx.getAdjacentPostsAroundReferenceInLocale(&post, locale)
		for _, candidate := range []*PostRecord{newer, older} {
			if candidate == nil || strings.TrimSpace(candidate.Slug) == "" {
				continue
			}
			key := locale + "|" + strings.TrimSpace(candidate.Slug)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			translation := getPostTranslationBySlugLocale(candidate.Slug, locale)
			if translation == nil {
				continue
			}
			if err := revalidateTranslationContext(root, translation, translation); err != nil {
				return err
			}
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
