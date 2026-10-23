package site

import (
	"strings"

	"alleycat-backend/internal/dag"
)

// Build topology from immutable snapshot metadata. Dependency discovery must not
// resolve route values: doing so renders every page before selecting any work.
func snapshotRouteDependencies(snapshot *snapshotBuildContext) map[dag.NodeKey][]dag.NodeKey {
	deps := map[dag.NodeKey][]dag.NodeKey{}
	for _, page := range snapshot.publishedPages {
		if pageURL := strings.TrimSpace(page.URL); pageURL != "" {
			deps[menuPagesNodeKey()] = append(deps[menuPagesNodeKey()], pageByURLNodeKey(pageURL))
		}
	}

	addFamily := func(sourceID string) {
		key := postFamilyNodeKey(sourceID)
		if _, exists := deps[key]; exists {
			return
		}
		deps[key] = nil
		if source, ok := snapshot.postByID[sourceID]; ok && strings.TrimSpace(source.Slug) != "" {
			deps[key] = append(deps[key], postBySlugNodeKey("", strings.TrimSpace(source.Slug)))
		}
		for _, translation := range snapshot.translationsBySource[sourceID] {
			locale := normalizeLocale(translation.Locale)
			if isEnabledTranslationLocale(snapshot.settings, locale) && strings.TrimSpace(translation.Slug) != "" {
				deps[key] = append(deps[key], postBySlugNodeKey(locale, strings.TrimSpace(translation.Slug)))
			}
		}
	}

	addPosts := func(locale string, items []PostRecord, sourceIDs []string) {
		positions := make(map[string]int, len(items))
		for i := range items {
			slug := strings.TrimSpace(items[i].Slug)
			if _, exists := positions[slug]; !exists {
				positions[slug] = i
			}
		}
		for i, post := range items {
			slug := strings.TrimSpace(post.Slug)
			if slug == "" {
				continue
			}
			postKey := postBySlugNodeKey(locale, slug)
			adjacentKey := adjacentPostsNodeKey(locale, slug)
			deps[adjacentKey] = []dag.NodeKey{postKey}
			position := positions[slug]
			for _, neighbor := range []int{position - 1, position + 1} {
				if neighbor >= 0 && neighbor < len(items) && strings.TrimSpace(items[neighbor].Slug) != "" {
					deps[adjacentKey] = append(deps[adjacentKey], postBySlugNodeKey(locale, items[neighbor].Slug))
				}
			}
			inputKey := postRenderInputNodeKey(locale, slug)
			deps[inputKey] = []dag.NodeKey{postKey, settingsNodeKey(), menuPagesNodeKey(), adjacentKey}
			sourceID := strings.TrimSpace(sourceIDs[i])
			if sourceID != "" {
				addFamily(sourceID)
				deps[inputKey] = append(deps[inputKey], postFamilyNodeKey(sourceID))
			}
			if snapshot.settings.ShowRelatedPosts {
				relatedKey := relatedPostsNodeKey(locale, slug)
				deps[relatedKey] = []dag.NodeKey{postKey}
				deps[inputKey] = append(deps[inputKey], relatedKey)
			}
		}
	}
	sourceIDs := make([]string, len(snapshot.publishedPosts))
	for i := range snapshot.publishedPosts {
		sourceIDs[i] = snapshot.publishedPosts[i].ID
	}
	addPosts("", snapshot.publishedPosts, sourceIDs)
	for locale, translations := range snapshot.translationsByLocale {
		if normalizeLocale(locale) == "" {
			continue
		}
		items := make([]PostRecord, len(translations))
		sourceIDs := make([]string, len(translations))
		for i := range translations {
			items[i] = translationToPost(translations[i])
			sourceIDs[i] = translations[i].SourcePost
		}
		addPosts(locale, items, sourceIDs)
	}

	for _, route := range snapshot.routeKeys() {
		switch {
		case route.ID == "/":
			input := homeRenderInputNodeKey()
			deps[route] = []dag.NodeKey{input}
			deps[input] = []dag.NodeKey{settingsNodeKey(), menuPagesNodeKey(), homeListingNodeKey()}
		case strings.HasPrefix(route.ID, "/archive"):
			input := archiveRenderInputNodeKey(route.ID)
			deps[route] = []dag.NodeKey{input}
			deps[input] = []dag.NodeKey{settingsNodeKey(), menuPagesNodeKey(), archiveListingNodeKey(route.ID)}
		default:
			if locale, slug, ok := resolvePostPath(route.ID); ok {
				deps[route] = []dag.NodeKey{postRenderInputNodeKey(locale, slug)}
			} else {
				input := pageRenderInputNodeKey(route.ID)
				deps[route] = []dag.NodeKey{input}
				deps[input] = []dag.NodeKey{pageByURLNodeKey(route.ID), settingsNodeKey(), menuPagesNodeKey()}
			}
		}
	}
	return deps
}
