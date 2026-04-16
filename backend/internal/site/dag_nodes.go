package site

import "alleycat-backend/internal/dag"

const (
	nodeScopeBase = "base"

	nodeSettings          dag.NodeKind = "site.settings"
	nodeMenuPages         dag.NodeKind = "site.menu_pages"
	nodePageByURL         dag.NodeKind = "site.page_by_url"
	nodePostBySlug        dag.NodeKind = "site.post_by_slug"
	nodeTranslationBySlug dag.NodeKind = "site.translation_by_slug"
	nodePostFamily        dag.NodeKind = "site.post_family"
	nodeAdjacentPosts     dag.NodeKind = "site.adjacent_posts"
	nodeRelatedPosts      dag.NodeKind = "site.related_posts"
	nodeHomeListing       dag.NodeKind = "site.home_listing"
	nodeArchiveListing    dag.NodeKind = "site.archive_listing"
	nodeHomeRenderInput   dag.NodeKind = "site.home_render_input"
	nodeArchiveRenderInput dag.NodeKind = "site.archive_render_input"
	nodePageRenderInput   dag.NodeKind = "site.page_render_input"
	nodePostRenderInput   dag.NodeKind = "site.post_render_input"
	nodeRoute             dag.NodeKind = "site.route"
)

func baseLocaleScope(locale string) string {
	if normalizeLocale(locale) == "" {
		return nodeScopeBase
	}
	return normalizeLocale(locale)
}

func settingsNodeKey() dag.NodeKey {
	return dag.NodeKey{Kind: nodeSettings, ID: "default"}
}

func menuPagesNodeKey() dag.NodeKey {
	return dag.NodeKey{Kind: nodeMenuPages, ID: "default"}
}

func postBySlugNodeKey(locale, slug string) dag.NodeKey {
	scope := baseLocaleScope(locale)
	kind := nodePostBySlug
	if scope != nodeScopeBase {
		kind = nodeTranslationBySlug
	}
	return dag.NodeKey{
		Kind:  kind,
		Scope: scope,
		ID:    slug,
	}
}

func pageByURLNodeKey(path string) dag.NodeKey {
	return dag.NodeKey{
		Kind: nodePageByURL,
		ID:   cleanPath(path),
	}
}

func postFamilyNodeKey(sourcePostID string) dag.NodeKey {
	return dag.NodeKey{
		Kind: nodePostFamily,
		ID:   sourcePostID,
	}
}

func adjacentPostsNodeKey(locale, slug string) dag.NodeKey {
	return dag.NodeKey{
		Kind:  nodeAdjacentPosts,
		Scope: baseLocaleScope(locale),
		ID:    slug,
	}
}

func relatedPostsNodeKey(locale, slug string) dag.NodeKey {
	return dag.NodeKey{
		Kind:  nodeRelatedPosts,
		Scope: baseLocaleScope(locale),
		ID:    slug,
	}
}

func postRenderInputNodeKey(locale, slug string) dag.NodeKey {
	return dag.NodeKey{
		Kind:  nodePostRenderInput,
		Scope: baseLocaleScope(locale),
		ID:    slug,
	}
}

func pageRenderInputNodeKey(path string) dag.NodeKey {
	return dag.NodeKey{
		Kind: nodePageRenderInput,
		ID:   cleanPath(path),
	}
}

func homeRenderInputNodeKey() dag.NodeKey {
	return dag.NodeKey{
		Kind: nodeHomeRenderInput,
		ID:   "/",
	}
}

func homeListingNodeKey() dag.NodeKey {
	return dag.NodeKey{
		Kind: nodeHomeListing,
		ID:   "/",
	}
}

func archiveRenderInputNodeKey(path string) dag.NodeKey {
	return dag.NodeKey{
		Kind: nodeArchiveRenderInput,
		ID:   cleanPath(path),
	}
}

func archiveListingNodeKey(path string) dag.NodeKey {
	return dag.NodeKey{
		Kind: nodeArchiveListing,
		ID:   cleanPath(path),
	}
}

func routeNodeKey(path string) dag.NodeKey {
	return dag.NodeKey{
		Kind: nodeRoute,
		ID:   cleanPath(path),
	}
}

type adjacentPostsValue struct {
	Newer *PostRecord
	Older *PostRecord
}

type postFamilyValue struct {
	Source       *PostRecord
	Translations []PostTranslationRecord
}

type postRenderInputValue struct {
	Input       *postRenderInput
	Settings    SettingsRecord
	Menu        []PageRecord
	Family      postFamilyValue
	Adjacent    adjacentPostsValue
	Related     []PostRecord
	CurrentPost *PostRecord
}

type pageRenderInputValue struct {
	Page     *PageRecord
	Settings SettingsRecord
	Menu     []PageRecord
}

type homeRenderInputValue struct {
	Settings SettingsRecord
	Menu     []PageRecord
	Listing  []PostRecord
}

type archiveRenderInputValue struct {
	Path     string
	Settings SettingsRecord
	Menu     []PageRecord
	Listing  archiveListing
}

type routeValue struct {
	Path string
	Body []byte
}
