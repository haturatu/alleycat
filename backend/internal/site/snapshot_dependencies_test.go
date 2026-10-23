package site

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"alleycat-backend/internal/dag"
)

func dependencyTestSnapshot() *snapshotBuildContext {
	settings := defaultSettings()
	settings.SiteURL = "https://example.test"
	settings.TranslationLocales = "en"
	settings.ShowRelatedPosts = true
	posts := []PostRecord{
		{ID: "newer-id", Slug: "newer", Title: "Newer", Published: true, PublishedAt: "2026-10-03T00:00:00Z"},
		{ID: "current-id", Slug: "current", Title: "Current", Published: true, PublishedAt: "2026-10-02T00:00:00Z"},
		{ID: "older-id", Slug: "older", Title: "Older", Published: true, PublishedAt: "2026-10-01T00:00:00Z"},
	}
	translations := []PostTranslationRecord{
		{SourcePost: "newer-id", Locale: "en", Slug: "newer-en", Title: "Newer EN", Published: true},
		{SourcePost: "current-id", Locale: "en", Slug: "current-en", Title: "Current EN", Published: true},
	}
	page := PageRecord{URL: "/about/", Title: "About", Published: true, MenuVisible: true}
	snapshot := &snapshotBuildContext{
		settings: settings, publishedPosts: posts, publishedPages: []PageRecord{page}, menu: []PageRecord{page},
		postBySlug: map[string]PostRecord{}, postByID: map[string]PostRecord{},
		pageByURL:            map[string]PageRecord{page.URL: page},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{"en": translations},
		archiveIndex:         map[string]archiveListing{"/archive/": {posts: posts, pageCount: 1}},
	}
	for _, post := range posts {
		snapshot.postBySlug[post.Slug] = post
		snapshot.postByID[post.ID] = post
	}
	for _, translation := range translations {
		snapshot.translationByKey[translation.Locale+"|"+translation.Slug] = translation
		snapshot.translationsBySource[translation.SourcePost] = append(snapshot.translationsBySource[translation.SourcePost], translation)
	}
	return snapshot
}

func dependencyKeyNames(keys []dag.NodeKey) []string {
	names := make([]string, len(keys))
	for i := range keys {
		names[i] = keys[i].String()
	}
	sort.Strings(names)
	return names
}

func TestSnapshotMetadataGraphMatchesResolvedDependencies(t *testing.T) {
	snapshot := dependencyTestSnapshot()
	if err := withSnapshotBuildContext(snapshot, func() error {
		engine := newSiteDAGEngine()
		resolved := engine.NewContext()
		for _, route := range snapshot.routeKeys() {
			if _, err := engine.Resolve(resolved, route); err != nil {
				return err
			}
		}
		metadata := snapshotRouteDependencies(snapshot)
		graph := engine.NewContextWithDependencies(metadata)
		for key, dependencies := range metadata {
			if got, want := dependencyKeyNames(dependencies), dependencyKeyNames(resolved.Dependencies(key)); !reflect.DeepEqual(got, want) {
				t.Errorf("dependencies of %s = %v, resolved graph has %v", key, got, want)
			}
		}
		for _, key := range []dag.NodeKey{
			settingsNodeKey(), menuPagesNodeKey(), pageByURLNodeKey("/about/"),
			postBySlugNodeKey("", "current"), postBySlugNodeKey("en", "current-en"),
		} {
			got := dependencyKeyNames(dagAffectedRouteKeysFromChanged(graph, []dag.NodeKey{key}))
			want := dependencyKeyNames(dagAffectedRouteKeysFromChanged(resolved, []dag.NodeKey{key}))
			if !reflect.DeepEqual(got, want) {
				t.Errorf("affected routes from %s = %v, want %v", key, got, want)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDAGRevalidationOnlyRendersAffectedRoutes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		changed []dag.NodeKey
		want    []dag.NodeKey
	}{
		{name: "no affected routes", changed: []dag.NodeKey{postBySlugNodeKey("", "absent")}},
		{name: "post and its dependents", changed: []dag.NodeKey{postBySlugNodeKey("", "current")}, want: []dag.NodeKey{
			routeNodeKey("/posts/newer"), routeNodeKey("/posts/current"), routeNodeKey("/posts/older"), routeNodeKey("/en/posts/current-en"),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := dependencyTestSnapshot()
			assertOnlyDependencyRoutesRendered(t, snapshot, tc.changed, nil, tc.want)
		})
	}
}

func assertOnlyDependencyRoutesRendered(t *testing.T, snapshot *snapshotBuildContext, changed []dag.NodeKey, extra []dagExtraRoute, want []dag.NodeKey) {
	t.Helper()
	root := t.TempDir()
	if err := writeSnapshotFile(root, "/unrelated/", []byte("unchanged")); err != nil {
		t.Fatal(err)
	}
	engine := newSiteDAGEngine()
	var rendered []dag.NodeKey
	engine.Register(nodeRoute, resolverFunc(func(_ *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
		rendered = append(rendered, key)
		return dag.ResolveResult{Value: routeValue{Path: key.ID, Body: []byte("rendered " + key.ID)}}, nil
	}))
	if err := revalidateDAGAffectedRoutesWithEngine(root, snapshot, engine, changed, extra); err != nil {
		t.Fatal(err)
	}
	if got, expected := dependencyKeyNames(rendered), dependencyKeyNames(want); !reflect.DeepEqual(got, expected) {
		t.Fatalf("rendered %v, want %v", got, expected)
	}
	for _, route := range want {
		target, err := snapshotFilePath(root, route.ID)
		if err != nil {
			t.Fatal(err)
		}
		if body, err := os.ReadFile(target); err != nil || string(body) != "rendered "+route.ID {
			t.Fatalf("affected route %s was not written: %s, %v", route, body, err)
		}
	}
	target, err := snapshotFilePath(root, "/unrelated/")
	if err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(target); err != nil || string(body) != "unchanged" {
		t.Fatalf("unrelated snapshot was modified: %s, %v", body, err)
	}
}

func TestDAGRevalidationKeepsDeletedRecordDependencies(t *testing.T) {
	t.Run("deleted page refreshes menus", func(t *testing.T) {
		snapshot := dependencyTestSnapshot()
		page := snapshot.publishedPages[0]
		snapshot.publishedPages, snapshot.menu, snapshot.pageByURL = nil, nil, nil
		assertOnlyDependencyRoutesRendered(t, snapshot, dagChangedKeysForPage(nil, &page), nil, snapshot.routeKeys())
	})
	t.Run("deleted translation refreshes source language links", func(t *testing.T) {
		snapshot := dependencyTestSnapshot()
		translation := snapshot.translationsByLocale["en"][1]
		snapshot.translationsByLocale["en"] = snapshot.translationsByLocale["en"][:1]
		delete(snapshot.translationsBySource, translation.SourcePost)
		delete(snapshot.translationByKey, "en|"+translation.Slug)
		assertOnlyDependencyRoutesRendered(t, snapshot, dagChangedKeysForTranslation(nil, &translation), nil,
			[]dag.NodeKey{routeNodeKey("/posts/current")})
	})
	t.Run("deleted source refreshes translation and neighbors", func(t *testing.T) {
		snapshot := dependencyTestSnapshot()
		post := snapshot.publishedPosts[1]
		snapshot.publishedPosts = []PostRecord{snapshot.publishedPosts[0], snapshot.publishedPosts[2]}
		delete(snapshot.postBySlug, post.Slug)
		delete(snapshot.postByID, post.ID)
		extra := snapshot.adjacentBaseRouteKeysForPosts(nil, &post)
		assertOnlyDependencyRoutesRendered(t, snapshot, dagChangedKeysForPost(nil, &post), extra,
			[]dag.NodeKey{routeNodeKey("/posts/newer"), routeNodeKey("/posts/older"), routeNodeKey("/en/posts/current-en")})
	})
}
