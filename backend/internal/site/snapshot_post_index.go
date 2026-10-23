package site

import "strings"

type snapshotPostIndex struct {
	posts          []PostRecord
	positionBySlug map[string]int
}

// Snapshot records are immutable after construction. Indexes belong to that
// snapshot, so publication changes naturally receive a new ordering and index.
func (ctx *snapshotBuildContext) preparePostIndexes() {
	ctx.postIndexesOnce.Do(func() {
		indexPosts := func(posts []PostRecord) snapshotPostIndex {
			index := snapshotPostIndex{posts: posts, positionBySlug: make(map[string]int, len(posts))}
			for i := range posts {
				slug := strings.TrimSpace(posts[i].Slug)
				if _, exists := index.positionBySlug[slug]; !exists {
					index.positionBySlug[slug] = i
				}
			}
			return index
		}
		ctx.postIndexes = make(map[string]snapshotPostIndex, 1+len(ctx.translationsByLocale))
		ctx.postIndexes[""] = indexPosts(ctx.publishedPosts)
		for locale, translations := range ctx.translationsByLocale {
			posts := make([]PostRecord, len(translations))
			for i := range translations {
				posts[i] = translationToPost(translations[i])
			}
			ctx.postIndexes[locale] = indexPosts(posts)
		}
	})
}

func (ctx *snapshotBuildContext) postIndexForLocale(locale string) snapshotPostIndex {
	ctx.preparePostIndexes()
	return ctx.postIndexes[normalizeLocale(locale)]
}

func adjacentPostsAt(items []PostRecord, index int) (newer, older *PostRecord) {
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
