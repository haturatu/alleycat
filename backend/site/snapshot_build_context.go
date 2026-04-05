package main

import (
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var activeSnapshotBuild = struct {
	mu  sync.RWMutex
	ctx *snapshotBuildContext
}{}

func withSnapshotBuildContext(ctx *snapshotBuildContext, fn func() error) error {
	activeSnapshotBuild.mu.Lock()
	prev := activeSnapshotBuild.ctx
	activeSnapshotBuild.ctx = ctx
	activeSnapshotBuild.mu.Unlock()
	defer func() {
		activeSnapshotBuild.mu.Lock()
		activeSnapshotBuild.ctx = prev
		activeSnapshotBuild.mu.Unlock()
	}()
	return fn()
}

func currentSnapshotBuildContext() *snapshotBuildContext {
	activeSnapshotBuild.mu.RLock()
	ctx := activeSnapshotBuild.ctx
	activeSnapshotBuild.mu.RUnlock()
	return ctx
}

func newSnapshotBuildContext() (*snapshotBuildContext, error) {
	settings, err := fetchSettingsStrict()
	if err != nil {
		return nil, err
	}
	pages, err := listPublishedPagesStrict()
	if err != nil {
		return nil, err
	}
	posts, err := listPublishedPostsStrict()
	if err != nil {
		return nil, err
	}
	tags, categories := collectTaxonomiesStrict(posts)

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 buildMenuPages(pages),
		publishedPosts:       append([]PostRecord(nil), posts...),
		publishedPages:       append([]PageRecord(nil), pages...),
		tags:                 append([]string(nil), tags...),
		categories:           append([]string(nil), categories...),
		postBySlug:           map[string]PostRecord{},
		postByID:             map[string]PostRecord{},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	for _, post := range ctx.publishedPosts {
		if slug := strings.TrimSpace(post.Slug); slug != "" {
			ctx.postBySlug[slug] = post
		}
		if id := strings.TrimSpace(post.ID); id != "" {
			ctx.postByID[id] = post
		}
		for _, tag := range parseTags(post.Tags) {
			ctx.postsByTag[tag] = append(ctx.postsByTag[tag], post)
		}
		if category := strings.TrimSpace(post.Category); category != "" {
			ctx.postsByCategory[category] = append(ctx.postsByCategory[category], post)
		}
	}

	for _, page := range ctx.publishedPages {
		if pageURL := strings.TrimSpace(page.URL); pageURL != "" {
			ctx.pageByURL[pageURL] = page
		}
	}

	for _, locale := range parseTranslationLocales(settings.TranslationLocales) {
		items, err := listPublishedTranslationsByLocaleStrict(locale)
		if err != nil {
			return nil, err
		}
		sort.SliceStable(items, func(i, j int) bool {
			a := postPublishedTime(translationToPost(items[i]))
			b := postPublishedTime(translationToPost(items[j]))
			if !a.Equal(b) {
				return a.After(b)
			}
			return items[i].Slug < items[j].Slug
		})
		ctx.translationsByLocale[locale] = append([]PostTranslationRecord(nil), items...)
		for _, item := range items {
			ctx.translationByKey[locale+"|"+strings.TrimSpace(item.Slug)] = item
			sourceID := strings.TrimSpace(item.SourcePost)
			if sourceID != "" {
				ctx.translationsBySource[sourceID] = append(ctx.translationsBySource[sourceID], item)
			}
		}
	}

	for sourceID, items := range ctx.translationsBySource {
		sort.SliceStable(items, func(i, j int) bool {
			return normalizeLocale(items[i].Locale) < normalizeLocale(items[j].Locale)
		})
		ctx.translationsBySource[sourceID] = items
	}

	ctx.archiveIndex["/archive/"] = ctx.buildArchiveListing(ctx.publishedPosts)
	for tag, items := range ctx.postsByTag {
		ctx.archiveIndex["/archive/"+url.PathEscape(tag)+"/"] = ctx.buildArchiveListing(items)
	}
	for category, items := range ctx.postsByCategory {
		ctx.archiveIndex["/archive/category/"+url.PathEscape(category)+"/"] = ctx.buildArchiveListing(items)
	}

	return ctx, nil
}

func buildMenuPages(pages []PageRecord) []PageRecord {
	menu := make([]PageRecord, 0, len(pages))
	for _, page := range pages {
		if page.Published && page.MenuVisible {
			menu = append(menu, page)
		}
	}
	sort.SliceStable(menu, func(i, j int) bool {
		if menu[i].MenuOrder != menu[j].MenuOrder {
			return menu[i].MenuOrder < menu[j].MenuOrder
		}
		return menu[i].URL < menu[j].URL
	})
	return menu
}

func (ctx *snapshotBuildContext) buildArchiveListing(items []PostRecord) archiveListing {
	copied := append([]PostRecord(nil), items...)
	perPage := ctx.settings.ArchivePageSize
	if perPage <= 0 {
		perPage = 10
	}
	pageCount := snapshotTotalPages(len(copied), perPage)
	return archiveListing{posts: copied, pageCount: pageCount}
}

func (ctx *snapshotBuildContext) postsForLocale(locale string) []PostRecord {
	locale = normalizeLocale(locale)
	if locale == "" {
		return append([]PostRecord(nil), ctx.publishedPosts...)
	}
	items := ctx.translationsByLocale[locale]
	out := make([]PostRecord, 0, len(items))
	for _, item := range items {
		out = append(out, translationToPost(item))
	}
	return out
}

func (ctx *snapshotBuildContext) getAdjacentPostsInLocale(post *PostRecord, locale string) (newer *PostRecord, older *PostRecord) {
	items := ctx.postsForLocale(locale)
	index := -1
	for i := range items {
		if strings.TrimSpace(items[i].Slug) == strings.TrimSpace(post.Slug) {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, nil
	}
	if index > 0 {
		copy := items[index-1]
		newer = &copy
	}
	if index+1 < len(items) {
		copy := items[index+1]
		older = &copy
	}
	return newer, older
}

func (ctx *snapshotBuildContext) getRelatedPostsInLocale(post *PostRecord, locale string, limit int) []PostRecord {
	return scoreRelatedPosts(post, ctx.postsForLocale(locale), limit)
}

func scoreRelatedPosts(post *PostRecord, candidates []PostRecord, limit int) []PostRecord {
	if post == nil {
		return nil
	}
	if limit <= 0 {
		limit = 4
	}

	targetTags := map[string]struct{}{}
	for _, tag := range parseTags(post.Tags) {
		targetTags[strings.ToLower(tag)] = struct{}{}
	}
	targetCategory := strings.ToLower(strings.TrimSpace(post.Category))

	type scoredPost struct {
		post  PostRecord
		score int
		date  time.Time
	}
	scored := make([]scoredPost, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Slug) == "" || candidate.Slug == post.Slug {
			continue
		}
		score := 0
		if targetCategory != "" && strings.ToLower(strings.TrimSpace(candidate.Category)) == targetCategory {
			score += 2
		}
		for _, tag := range parseTags(candidate.Tags) {
			if _, ok := targetTags[strings.ToLower(tag)]; ok {
				score++
			}
		}
		if score == 0 {
			continue
		}
		scored = append(scored, scoredPost{post: candidate, score: score, date: postPublishedTime(candidate)})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].date.Equal(scored[j].date) {
			return scored[i].date.After(scored[j].date)
		}
		return scored[i].post.Slug < scored[j].post.Slug
	})
	if len(scored) > limit {
		scored = scored[:limit]
	}
	out := make([]PostRecord, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.post)
	}
	return out
}
