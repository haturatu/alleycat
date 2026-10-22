package site

import (
	"fmt"
	"sync"
	"testing"
)

func TestSnapshotAdjacentPostIndex(t *testing.T) {
	t.Parallel()
	ctx := &snapshotBuildContext{
		publishedPosts: []PostRecord{{Slug: "newer"}, {Slug: "current"}, {Slug: "older"}},
		translationsByLocale: map[string][]PostTranslationRecord{
			"zh-cn": {{Slug: "translated-newer"}, {Slug: "translated-current"}, {Slug: "translated-older"}},
		},
	}
	for _, tc := range []struct{ locale, slug, newer, older string }{
		{"", " current ", "newer", "older"},
		{"", "newer", "", "current"},
		{"", "older", "current", ""},
		{"", "missing", "", ""},
		{"ZH_CN", "translated-current", "translated-newer", "translated-older"},
		{"fr", "current", "", ""},
	} {
		t.Run(tc.locale+"/"+tc.slug, func(t *testing.T) {
			newer, older := ctx.getAdjacentPostsInLocale(&PostRecord{Slug: tc.slug}, tc.locale)
			if postSlug(newer) != tc.newer || postSlug(older) != tc.older {
				t.Fatalf("neighbors = (%q, %q), want (%q, %q)", postSlug(newer), postSlug(older), tc.newer, tc.older)
			}
			if newer != nil {
				newer.Slug = "modified return value"
			}
			again, _ := ctx.getAdjacentPostsInLocale(&PostRecord{Slug: tc.slug}, tc.locale)
			if postSlug(again) != tc.newer {
				t.Fatal("mutating a returned neighbor modified the snapshot")
			}
		})
	}
}

func TestSnapshotAdjacentIndexPreservesFirstDuplicateAndInsertionOrder(t *testing.T) {
	t.Parallel()
	ctx := &snapshotBuildContext{publishedPosts: []PostRecord{
		{Slug: "a", PublishedAt: "2026-10-03T00:00:00Z"},
		{Slug: " duplicate ", PublishedAt: "2026-10-02T00:00:00Z"},
		{Slug: "duplicate", PublishedAt: "2026-10-01T00:00:00Z"},
	}}
	newer, older := ctx.getAdjacentPostsInLocale(&PostRecord{Slug: "duplicate"}, "")
	if postSlug(newer) != "a" || postSlug(older) != "duplicate" {
		t.Fatal("duplicate slugs no longer select the first match")
	}
	newer, older = ctx.getAdjacentPostsAroundReferenceInLocale(&PostRecord{
		Slug: "deleted", PublishedAt: "2026-10-02T12:00:00Z",
	}, "")
	if postSlug(newer) != "a" || postSlug(older) != " duplicate " {
		t.Fatal("deleted reference did not retain its chronological insertion position")
	}
	if newer, older := ctx.getAdjacentPostsInLocale(nil, ""); newer != nil || older != nil {
		t.Fatal("nil reference has neighbors")
	}
}

func TestSnapshotAdjacentIndexConcurrentInitialization(t *testing.T) {
	t.Parallel()
	ctx := &snapshotBuildContext{publishedPosts: []PostRecord{{Slug: "a"}, {Slug: "b"}, {Slug: "c"}}}
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			newer, older := ctx.getAdjacentPostsInLocale(&PostRecord{Slug: "b"}, "")
			if postSlug(newer) != "a" || postSlug(older) != "c" {
				t.Error("concurrent lookup returned incorrect neighbors")
			}
		})
	}
	wg.Wait()
}

func TestSnapshotAdjacentLookupDoesNotAllocateArticleList(t *testing.T) {
	ctx := &snapshotBuildContext{publishedPosts: make([]PostRecord, 10000)}
	for i := range ctx.publishedPosts {
		ctx.publishedPosts[i].Slug = fmt.Sprint(i)
	}
	ctx.preparePostIndexes()
	post := &ctx.publishedPosts[5000]
	allocations := testing.AllocsPerRun(10, func() { _, _ = ctx.getAdjacentPostsInLocale(post, "") })
	if allocations > 2 {
		t.Fatalf("lookup allocated %.0f objects, want only the two returned records", allocations)
	}
}

func postSlug(post *PostRecord) string {
	if post == nil {
		return ""
	}
	return post.Slug
}
