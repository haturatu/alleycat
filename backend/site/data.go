package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const taxonomyCacheTTL = 60 * time.Second
const mediaPathCacheTTL = 60 * time.Second

type taxonomyCacheEntry struct {
	expiresAt  time.Time
	tags       []string
	categories []string
}

type mediaPathCacheEntry struct {
	expiresAt time.Time
	found     bool
	media     MediaRecord
}

type pagedListFetcher[T any] func(params map[string]string) (PBList[T], error)

var taxonomyCache = struct {
	mu    sync.RWMutex
	entry taxonomyCacheEntry
}{}

var mediaPathCache = struct {
	mu    sync.RWMutex
	items map[string]mediaPathCacheEntry
}{
	items: map[string]mediaPathCacheEntry{},
}

var filterEscapeReplacer = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")

func getPosts(params map[string]string) (PBList[PostRecord], error) {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return ctx.queryPosts(params), nil
	}
	return fetchList[PostRecord](fmt.Sprintf("%s/api/collections/posts/records", pbURL), params)
}

func getPostTranslations(params map[string]string) (PBList[PostTranslationRecord], error) {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return ctx.queryPostTranslations(params), nil
	}
	return fetchList[PostTranslationRecord](fmt.Sprintf("%s/api/collections/post_translations/records", pbURL), params)
}

func getPages(params map[string]string) (PBList[PageRecord], error) {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		items := make([]PageRecord, 0, len(ctx.publishedPages))
		for _, item := range ctx.publishedPages {
			if item.Published {
				items = append(items, item)
			}
		}
		return paginateSnapshotPages(items, params["page"], params["perPage"]), nil
	}
	return fetchList[PageRecord](fmt.Sprintf("%s/api/collections/pages/records", pbURL), params)
}

func getPagesMenu() []PageRecord {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return append([]PageRecord(nil), ctx.menu...)
	}
	data, err := getPages(map[string]string{
		"perPage": "200",
		"filter":  "published = true && menuVisible = true",
		"sort":    "menuOrder",
	})
	if err != nil {
		return nil
	}
	return data.Items
}

func getPageByURL(path string) *PageRecord {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		item, ok := ctx.pageByURL[path]
		if !ok {
			return nil
		}
		copy := item
		return &copy
	}
	data, err := getPages(map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("url = \"%s\" && published = true", escapeFilter(path)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func getPostBySlugInLocale(slug string, locale string) *PostRecord {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		if normalizeLocale(locale) == "" {
			item, ok := ctx.postBySlug[slug]
			if !ok {
				return nil
			}
			copy := item
			return &copy
		}
		item, ok := ctx.translationByKey[normalizeLocale(locale)+"|"+slug]
		if !ok {
			return nil
		}
		post := translationToPost(item)
		return &post
	}
	if locale == "" {
		data, err := fetchList[PostRecord](fmt.Sprintf("%s/api/collections/posts/records", pbURL), map[string]string{
			"perPage": "1",
			"filter":  fmt.Sprintf("slug = \"%s\" && published = true", escapeFilter(slug)),
		})
		if err != nil || len(data.Items) == 0 {
			return nil
		}
		return &data.Items[0]
	}

	data, err := getPostTranslations(map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("slug = \"%s\" && locale = \"%s\" && published = true", escapeFilter(slug), escapeFilter(locale)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	post := translationToPost(data.Items[0])
	return &post
}

func getPostByID(id string) *PostRecord {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		item, ok := ctx.postByID[id]
		if !ok {
			return nil
		}
		copy := item
		return &copy
	}
	post, err := fetchRecord[PostRecord](fmt.Sprintf("%s/api/collections/posts/records/%s", pbURL, url.PathEscape(id)))
	if err != nil {
		return nil
	}
	return &post
}

func getPostTranslationBySlugLocale(slug string, locale string) *PostTranslationRecord {
	if strings.TrimSpace(slug) == "" || strings.TrimSpace(locale) == "" {
		return nil
	}
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		item, ok := ctx.translationByKey[normalizeLocale(locale)+"|"+slug]
		if !ok {
			return nil
		}
		copy := item
		return &copy
	}
	data, err := getPostTranslations(map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("slug = \"%s\" && locale = \"%s\" && published = true", escapeFilter(slug), escapeFilter(locale)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func getPostTranslationsBySource(sourcePostID string) []PostTranslationRecord {
	if strings.TrimSpace(sourcePostID) == "" {
		return nil
	}
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return append([]PostTranslationRecord(nil), ctx.translationsBySource[sourcePostID]...)
	}
	data, err := getPostTranslations(map[string]string{
		"page":    "1",
		"perPage": "200",
		"filter":  fmt.Sprintf("source_post = \"%s\" && published = true", escapeFilter(sourcePostID)),
		"sort":    "locale",
	})
	if err != nil {
		return nil
	}
	return data.Items
}

func filterTranslationsByEnabledLocales(items []PostTranslationRecord, settings SettingsRecord) []PostTranslationRecord {
	if len(items) == 0 {
		return nil
	}
	filtered := make([]PostTranslationRecord, 0, len(items))
	for _, item := range items {
		if !isEnabledTranslationLocale(settings, item.Locale) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func getAdjacentPostsInLocale(post *PostRecord, locale string) (newer *PostRecord, older *PostRecord) {
	if post == nil {
		return nil, nil
	}
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return ctx.getAdjacentPostsInLocale(post, locale)
	}
	field := ""
	value := ""
	if strings.TrimSpace(post.PublishedAt) != "" {
		field = "published_at"
		value = post.PublishedAt
	} else if strings.TrimSpace(post.Date) != "" {
		field = "date"
		value = post.Date
	}
	if field == "" || value == "" {
		return nil, nil
	}

	if locale == "" {
		fetchNearest := func(op, sort string) *PostRecord {
			data, err := getPosts(map[string]string{
				"page":    "1",
				"perPage": "1",
				"filter":  fmt.Sprintf("published = true && %s %s \"%s\"", field, op, escapeFilter(value)),
				"sort":    sort,
			})
			if err != nil || len(data.Items) == 0 {
				return nil
			}
			return &data.Items[0]
		}
		newer = fetchNearest(">", field)
		older = fetchNearest("<", "-"+field)
		return newer, older
	}

	fetchNearestTranslated := func(op, sort string) *PostRecord {
		data, err := getPostTranslations(map[string]string{
			"page":    "1",
			"perPage": "1",
			"filter":  fmt.Sprintf("published = true && locale = \"%s\" && %s %s \"%s\"", escapeFilter(locale), field, op, escapeFilter(value)),
			"sort":    sort,
		})
		if err != nil || len(data.Items) == 0 {
			return nil
		}
		item := translationToPost(data.Items[0])
		return &item
	}
	newer = fetchNearestTranslated(">", field)
	older = fetchNearestTranslated("<", "-"+field)
	return newer, older
}

func getRelatedPostsInLocale(post *PostRecord, locale string, limit int) []PostRecord {
	if post == nil {
		return nil
	}
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return ctx.getRelatedPostsInLocale(post, locale, limit)
	}
	if limit <= 0 {
		limit = 4
	}

	var candidates []PostRecord
	if locale == "" {
		items, err := getPosts(map[string]string{
			"page":    "1",
			"perPage": "120",
			"filter":  fmt.Sprintf("published = true && id != \"%s\"", escapeFilter(post.ID)),
			"sort":    "-published_at",
		})
		if err != nil {
			items, err = getPosts(map[string]string{
				"page":    "1",
				"perPage": "120",
				"filter":  fmt.Sprintf("published = true && id != \"%s\"", escapeFilter(post.ID)),
				"sort":    "-date",
			})
			if err != nil {
				return nil
			}
		}
		candidates = items.Items
	} else {
		items, err := getPostTranslations(map[string]string{
			"page":    "1",
			"perPage": "120",
			"filter":  fmt.Sprintf("published = true && locale = \"%s\" && id != \"%s\"", escapeFilter(locale), escapeFilter(post.ID)),
			"sort":    "-published_at",
		})
		if err != nil {
			return nil
		}
		candidates = make([]PostRecord, 0, len(items.Items))
		for _, item := range items.Items {
			candidates = append(candidates, translationToPost(item))
		}
	}

	return scoreRelatedPosts(post, candidates, limit)
}

func postPublishedTime(post PostRecord) time.Time {
	value := strings.TrimSpace(post.PublishedAt)
	if value == "" {
		value = strings.TrimSpace(post.Date)
	}
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed
	}
	return time.Time{}
}

func getMediaByIDs(ids []string) map[string]MediaRecord {
	result := make(map[string]MediaRecord, len(ids))
	if len(ids) == 0 {
		return result
	}

	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	if len(unique) == 0 {
		return result
	}

	const chunkSize = 40
	for start := 0; start < len(unique); start += chunkSize {
		end := start + chunkSize
		if end > len(unique) {
			end = len(unique)
		}
		chunk := unique[start:end]
		parts := make([]string, 0, len(chunk))
		for _, id := range chunk {
			parts = append(parts, fmt.Sprintf("id = \"%s\"", escapeFilter(id)))
		}
		filter := strings.Join(parts, " || ")
		data, err := fetchList[MediaRecord](fmt.Sprintf("%s/api/collections/media/records", pbURL), map[string]string{
			"page":    "1",
			"perPage": strconv.Itoa(len(chunk)),
			"filter":  filter,
		})
		if err != nil {
			continue
		}
		for _, item := range data.Items {
			result[item.ID] = item
		}
	}

	return result
}

func getMediaByPath(path string) *MediaRecord {
	now := time.Now()
	mediaPathCache.mu.RLock()
	cached, ok := mediaPathCache.items[path]
	mediaPathCache.mu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		if !cached.found {
			return nil
		}
		item := cached.media
		return &item
	}

	data, err := fetchList[MediaRecord](fmt.Sprintf("%s/api/collections/media/records", pbURL), map[string]string{
		"page":    "1",
		"perPage": "1",
		"filter":  fmt.Sprintf("path = \"%s\"", escapeFilter(path)),
	})
	if err != nil || len(data.Items) == 0 {
		mediaPathCache.mu.Lock()
		mediaPathCache.items[path] = mediaPathCacheEntry{
			expiresAt: now.Add(mediaPathCacheTTL),
			found:     false,
		}
		mediaPathCache.mu.Unlock()
		return nil
	}
	item := data.Items[0]
	mediaPathCache.mu.Lock()
	mediaPathCache.items[path] = mediaPathCacheEntry{
		expiresAt: now.Add(mediaPathCacheTTL),
		found:     true,
		media:     item,
	}
	mediaPathCache.mu.Unlock()
	return &item
}

func collectTags() []string {
	tags, _ := collectTaxonomies()
	return tags
}

func collectCategories() []string {
	_, categories := collectTaxonomies()
	return categories
}

func collectTaxonomies() ([]string, []string) {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return append([]string(nil), ctx.tags...), append([]string(nil), ctx.categories...)
	}
	now := time.Now()
	taxonomyCache.mu.RLock()
	cached := taxonomyCache.entry
	taxonomyCache.mu.RUnlock()
	if now.Before(cached.expiresAt) {
		return append([]string(nil), cached.tags...), append([]string(nil), cached.categories...)
	}

	posts := listPublishedPosts()

	tags, categories := collectTaxonomiesStrict(posts)

	taxonomyCache.mu.Lock()
	taxonomyCache.entry = taxonomyCacheEntry{
		expiresAt:  now.Add(taxonomyCacheTTL),
		tags:       append([]string(nil), tags...),
		categories: append([]string(nil), categories...),
	}
	taxonomyCache.mu.Unlock()

	return tags, categories
}

func listPublishedPosts() []PostRecord {
	items, _ := listPublishedPostsStrict()
	return items
}

func listPublishedPostsStrict() ([]PostRecord, error) {
	return listPublishedRecords(getPosts, "published = true", 200, true, "-published_at", "-date")
}

func listPublishedPages() []PageRecord {
	items, _ := listPublishedPagesStrict()
	return items
}

func listPublishedPagesStrict() ([]PageRecord, error) {
	return listPublishedRecords(getPages, "published = true", 200, true, "-published_at", "-date")
}

func listPublishedTranslationsByLocale(locale string) []PostTranslationRecord {
	items, _ := listPublishedTranslationsByLocaleStrict(locale)
	return items
}

func listPublishedTranslationsByLocaleStrict(locale string) ([]PostTranslationRecord, error) {
	return listPublishedRecords(
		getPostTranslations,
		fmt.Sprintf("published = true && locale = \"%s\"", escapeFilter(locale)),
		200,
		true,
		"-published_at",
	)
}

func listPublishedRecords[T any](fetch pagedListFetcher[T], filter string, perPage int, strict bool, sorts ...string) ([]T, error) {
	if perPage <= 0 {
		perPage = 200
	}
	if len(sorts) == 0 {
		sorts = []string{"-published_at"}
	}

	items := make([]T, 0, perPage)
	page := 1
	for {
		data, err := fetchPublishedPage(fetch, filter, page, perPage, sorts...)
		if err != nil {
			if strict {
				return nil, err
			}
			break
		}
		items = append(items, data.Items...)
		if len(data.Items) < perPage {
			break
		}
		page++
	}
	return items, nil
}

func fetchPublishedPage[T any](fetch pagedListFetcher[T], filter string, page int, perPage int, sorts ...string) (PBList[T], error) {
	var lastErr error
	for _, sortValue := range sorts {
		data, err := fetch(map[string]string{
			"page":    strconv.Itoa(page),
			"perPage": strconv.Itoa(perPage),
			"filter":  filter,
			"sort":    sortValue,
		})
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return PBList[T]{}, lastErr
}

func collectTaxonomiesStrict(posts []PostRecord) ([]string, []string) {
	tagValues := map[string]struct{}{}
	categoryValues := map[string]struct{}{}
	for _, post := range posts {
		for _, tag := range parseTags(post.Tags) {
			tagValues[tag] = struct{}{}
		}
		category := strings.TrimSpace(post.Category)
		if category != "" {
			categoryValues[category] = struct{}{}
		}
	}

	tags := make([]string, 0, len(tagValues))
	for value := range tagValues {
		tags = append(tags, value)
	}
	sort.Strings(tags)

	categories := make([]string, 0, len(categoryValues))
	for value := range categoryValues {
		categories = append(categories, value)
	}
	sort.Strings(categories)

	return tags, categories
}

func translationToPost(item PostTranslationRecord) PostRecord {
	return PostRecord{
		ID:          item.ID,
		Title:       item.Title,
		Slug:        item.Slug,
		Body:        item.Body,
		Excerpt:     item.Excerpt,
		Tags:        item.Tags,
		Category:    item.Category,
		Published:   item.Published,
		PublishedAt: item.PublishedAt,
		Date:        item.PublishedAt,
	}
}

func escapeFilter(value string) string {
	return filterEscapeReplacer.Replace(value)
}

func decodePathSegment(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

func (ctx *snapshotBuildContext) queryPosts(params map[string]string) PBList[PostRecord] {
	items := append([]PostRecord(nil), ctx.publishedPosts...)
	filter := strings.TrimSpace(params["filter"])
	if category := extractFilterValue(filter, `category = "`); category != "" {
		items = append([]PostRecord(nil), ctx.postsByCategory[category]...)
	} else if tag := extractFilterValue(filter, `tags ~ "`); tag != "" {
		items = append([]PostRecord(nil), ctx.postsByTag[tag]...)
	}
	if query := extractSearchQuery(filter); query != "" {
		filtered := make([]PostRecord, 0, len(items))
		for _, item := range items {
			if snapshotPostMatchesQuery(item, query) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if excludeID := extractFilterValue(filter, `id != "`); excludeID != "" {
		filtered := make([]PostRecord, 0, len(items))
		for _, item := range items {
			if item.ID != excludeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if slug := extractFilterValue(filter, `slug = "`); slug != "" {
		filtered := make([]PostRecord, 0, len(items))
		for _, item := range items {
			if strings.TrimSpace(item.Slug) == slug {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return paginateSnapshotPosts(items, params["page"], params["perPage"])
}

func (ctx *snapshotBuildContext) queryPostTranslations(params map[string]string) PBList[PostTranslationRecord] {
	filter := strings.TrimSpace(params["filter"])
	locale := normalizeLocale(extractFilterValue(filter, `locale = "`))
	sourceID := extractFilterValue(filter, `source_post = "`)
	slug := extractFilterValue(filter, `slug = "`)
	excludeID := extractFilterValue(filter, `id != "`)

	var items []PostTranslationRecord
	if sourceID != "" {
		items = append([]PostTranslationRecord(nil), ctx.translationsBySource[sourceID]...)
	} else if locale != "" {
		items = append([]PostTranslationRecord(nil), ctx.translationsByLocale[locale]...)
	} else {
		for _, list := range ctx.translationsByLocale {
			items = append(items, list...)
		}
	}

	filtered := make([]PostTranslationRecord, 0, len(items))
	for _, item := range items {
		if locale != "" && normalizeLocale(item.Locale) != locale {
			continue
		}
		if slug != "" && strings.TrimSpace(item.Slug) != slug {
			continue
		}
		if excludeID != "" && item.ID == excludeID {
			continue
		}
		filtered = append(filtered, item)
	}
	return paginateSnapshotTranslations(filtered, params["page"], params["perPage"])
}

func paginateSnapshotPosts(items []PostRecord, pageValue, perPageValue string) PBList[PostRecord] {
	page, perPage := snapshotPageParams(pageValue, perPageValue)
	totalItems := len(items)
	totalPages := snapshotTotalPages(totalItems, perPage)
	start, end := snapshotPageBounds(page, perPage, totalItems)
	out := PBList[PostRecord]{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
	if start < end {
		out.Items = append([]PostRecord(nil), items[start:end]...)
	}
	return out
}

func paginateSnapshotPages(items []PageRecord, pageValue, perPageValue string) PBList[PageRecord] {
	page, perPage := snapshotPageParams(pageValue, perPageValue)
	totalItems := len(items)
	totalPages := snapshotTotalPages(totalItems, perPage)
	start, end := snapshotPageBounds(page, perPage, totalItems)
	out := PBList[PageRecord]{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
	if start < end {
		out.Items = append([]PageRecord(nil), items[start:end]...)
	}
	return out
}

func paginateSnapshotTranslations(items []PostTranslationRecord, pageValue, perPageValue string) PBList[PostTranslationRecord] {
	page, perPage := snapshotPageParams(pageValue, perPageValue)
	totalItems := len(items)
	totalPages := snapshotTotalPages(totalItems, perPage)
	start, end := snapshotPageBounds(page, perPage, totalItems)
	out := PBList[PostTranslationRecord]{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
	if start < end {
		out.Items = append([]PostTranslationRecord(nil), items[start:end]...)
	}
	return out
}

func snapshotPageParams(pageValue, perPageValue string) (int, int) {
	page, _ := strconv.Atoi(pageValue)
	if page <= 0 {
		page = 1
	}
	perPage, _ := strconv.Atoi(perPageValue)
	if perPage <= 0 {
		perPage = 30
	}
	return page, perPage
}

func snapshotTotalPages(totalItems, perPage int) int {
	if totalItems == 0 {
		return 1
	}
	return (totalItems + perPage - 1) / perPage
}

func snapshotPageBounds(page, perPage, totalItems int) (int, int) {
	start := (page - 1) * perPage
	if start > totalItems {
		start = totalItems
	}
	end := start + perPage
	if end > totalItems {
		end = totalItems
	}
	return start, end
}

func extractFilterValue(filter, prefix string) string {
	index := strings.Index(filter, prefix)
	if index < 0 {
		return ""
	}
	rest := filter[index+len(prefix):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func extractSearchQuery(filter string) string {
	const marker = `title ~ "`
	index := strings.Index(filter, marker)
	if index < 0 {
		return ""
	}
	rest := filter[index+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func snapshotPostMatchesQuery(item PostRecord, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	body := item.Body
	if body == "" {
		body = item.Content
	}
	excerpt := item.Excerpt
	if strings.TrimSpace(excerpt) == "" {
		excerpt = buildExcerpt(body, 160)
	}
	fields := []string{item.Title, item.Slug, item.Tags, excerpt, stripHTML(body)}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}
