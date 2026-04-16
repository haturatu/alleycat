package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"alleycat-backend/internal/dag"
)

func TestRevalidateTranslationContextUpdatesHomeAndArchive(t *testing.T) {
	root := t.TempDir()
	for _, route := range []string{"/", "/archive/"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", route, err)
		}
		if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", route, err)
		}
	}

	settings := defaultSettings()
	settings.WelcomeText = "Welcome"
	settings.HomePageSize = 3
	settings.ArchivePageSize = 10
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "en,ru"

	post := PostRecord{
		ID:          "source-1",
		Slug:        "new-post",
		Title:       "New Post",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-11 12:33:00.000Z",
		Tags:        "dns,go",
		Category:    "infra",
	}
	translation := PostTranslationRecord{
		SourcePost:  "source-1",
		Locale:      "ru",
		Slug:        "new-post",
		Title:       "New Post RU",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-11 12:33:00.000Z",
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 nil,
		publishedPosts:       []PostRecord{post},
		publishedPages:       nil,
		tags:                 []string{"dns", "go"},
		categories:           []string{"infra"},
		postBySlug:           map[string]PostRecord{post.Slug: post},
		postByID:             map[string]PostRecord{post.ID: post},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{post.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{"dns": []PostRecord{post}, "go": []PostRecord{post}},
		postsByCategory:      map[string][]PostRecord{"infra": []PostRecord{post}},
		archiveIndex: map[string]archiveListing{
			"/archive/":                {posts: []PostRecord{post}, pageCount: 1},
			"/archive/dns/":            {posts: []PostRecord{post}, pageCount: 1},
			"/archive/go/":             {posts: []PostRecord{post}, pageCount: 1},
			"/archive/category/infra/": {posts: []PostRecord{post}, pageCount: 1},
		},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		return revalidateTranslationContext(root, &translation, nil)
	})
	if err != nil {
		t.Fatalf("revalidateTranslationContext: %v", err)
	}

	for route, want := range map[string]string{
		"/":         "new-post",
		"/archive/": "new-post",
	} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		body, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", route, err)
		}
		text := string(body)
		if strings.Contains(text, "stale") {
			t.Fatalf("%q still stale", route)
		}
		if !strings.Contains(text, want) {
			t.Fatalf("%q missing %q: %s", route, want, text)
		}
	}
}

func TestRevalidateTranslationContextUsesDAGForHomeAndArchiveWhenEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	for _, route := range []string{"/", "/archive/"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", route, err)
		}
		if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", route, err)
		}
	}

	settings := defaultSettings()
	settings.WelcomeText = "Welcome"
	settings.HomePageSize = 3
	settings.ArchivePageSize = 10
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	post := PostRecord{
		ID:          "source-1",
		Slug:        "new-post",
		Title:       "New Post",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-11 12:33:00.000Z",
	}
	translation := PostTranslationRecord{
		SourcePost:  "source-1",
		Locale:      "ru",
		Slug:        "new-post",
		Title:       "New Post RU",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-11 12:33:00.000Z",
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 nil,
		publishedPosts:       []PostRecord{post},
		publishedPages:       nil,
		tags:                 nil,
		categories:           nil,
		postBySlug:           map[string]PostRecord{post.Slug: post},
		postByID:             map[string]PostRecord{post.ID: post},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{post.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex: map[string]archiveListing{
			"/archive/": {posts: []PostRecord{post}, pageCount: 1},
		},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		return revalidateTranslationContext(root, &translation, nil)
	})
	if err != nil {
		t.Fatalf("revalidateTranslationContext: %v", err)
	}

	for route, want := range map[string]string{
		"/":        "new-post",
		"/archive/": "new-post",
	} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		body, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", route, err)
		}
		text := string(body)
		if strings.Contains(text, "stale") {
			t.Fatalf("%q still stale", route)
		}
		if !strings.Contains(text, want) {
			t.Fatalf("%q missing %q: %s", route, want, text)
		}
	}
}

func TestRevalidatePostUsesDAGToUpdateCurrentRouteWhenEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"

	post := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/posts/hello/")
	if err != nil {
		t.Fatalf("snapshotFilePath current: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll current: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile current: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{post},
		postBySlug:           map[string]PostRecord{post.Slug: post},
		postByID:             map[string]PostRecord{post.ID: post},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	req := revalidateRequest{Current: mustMarshalJSONRaw(t, post)}
	err = withSnapshotBuildContext(ctx, func() error {
		return revalidatePost(root, req)
	})
	if err != nil {
		t.Fatalf("revalidatePost: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile current: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("current route still stale")
	}
	if !strings.Contains(text, "Hello") {
		t.Fatalf("current route missing rendered post: %s", text)
	}
}

func TestRevalidatePageUsesDAGToUpdateRoutesAffectedByMenuChange(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"

	post := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	page := PageRecord{
		ID:          "page-1",
		Title:       "About",
		URL:         "/about/",
		Body:        "<p>about</p>",
		Published:   true,
		MenuVisible: true,
	}

	homeTarget, err := snapshotFilePath(root, "/")
	if err != nil {
		t.Fatalf("snapshotFilePath home: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(homeTarget), 0o755); err != nil {
		t.Fatalf("MkdirAll home: %v", err)
	}
	if err := os.WriteFile(homeTarget, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile home: %v", err)
	}

	pageTarget, err := snapshotFilePath(root, "/about/")
	if err != nil {
		t.Fatalf("snapshotFilePath page: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(pageTarget), 0o755); err != nil {
		t.Fatalf("MkdirAll page: %v", err)
	}
	if err := os.WriteFile(pageTarget, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile page: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 []PageRecord{page},
		publishedPosts:       []PostRecord{post},
		publishedPages:       []PageRecord{page},
		postBySlug:           map[string]PostRecord{post.Slug: post},
		postByID:             map[string]PostRecord{post.ID: post},
		pageByURL:            map[string]PageRecord{page.URL: page},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex: map[string]archiveListing{
			"/archive/": {posts: []PostRecord{post}, pageCount: 1},
		},
	}

	req := revalidateRequest{Current: mustMarshalJSONRaw(t, page)}
	err = withSnapshotBuildContext(ctx, func() error {
		return revalidatePage(root, req)
	})
	if err != nil {
		t.Fatalf("revalidatePage: %v", err)
	}

	for route, target := range map[string]string{
		"/":      homeTarget,
		"/about/": pageTarget,
	} {
		body, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", route, err)
		}
		text := string(body)
		if strings.Contains(text, "stale") {
			t.Fatalf("%q still stale", route)
		}
		if !strings.Contains(text, "About") {
			t.Fatalf("%q missing menu/page title: %s", route, text)
		}
	}
}

func TestRevalidateTranslationUsesDAGToUpdateCurrentRouteWhenEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	source := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	translation := PostTranslationRecord{
		ID:          "tr-1",
		SourcePost:  source.ID,
		Locale:      "ru",
		Slug:        "privet",
		Title:       "Privet",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/ru/posts/privet/")
	if err != nil {
		t.Fatalf("snapshotFilePath localized: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll localized: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile localized: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{source},
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	req := revalidateRequest{Current: mustMarshalJSONRaw(t, translation)}
	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateTranslation(root, req)
	})
	if err != nil {
		t.Fatalf("revalidateTranslation: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile localized: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("localized route still stale")
	}
	if !strings.Contains(text, "Privet") {
		t.Fatalf("localized route missing rendered translation: %s", text)
	}
}

func TestRevalidatePostRemovesOldRouteAndUpdatesNeighborWhenUnpublishedWithDAG(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"

	newer := PostRecord{
		ID:          "post-newer",
		Slug:        "newer",
		Title:       "Newer",
		Body:        "<p>newer</p>",
		Published:   false,
		PublishedAt: "2026-04-17T10:00:00Z",
	}
	original := PostRecord{
		ID:          "post-newer",
		Slug:        "newer",
		Title:       "Newer",
		Body:        "<p>newer</p>",
		Published:   true,
		PublishedAt: "2026-04-17T10:00:00Z",
	}
	current := PostRecord{
		ID:          "post-current",
		Slug:        "current",
		Title:       "Current",
		Body:        "<p>current</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	older := PostRecord{
		ID:          "post-older",
		Slug:        "older",
		Title:       "Older",
		Body:        "<p>older</p>",
		Published:   true,
		PublishedAt: "2026-04-15T10:00:00Z",
	}

	removedTarget, err := snapshotFilePath(root, "/posts/newer/")
	if err != nil {
		t.Fatalf("snapshotFilePath removed route: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(removedTarget), 0o755); err != nil {
		t.Fatalf("MkdirAll removed route: %v", err)
	}
	if err := os.WriteFile(removedTarget, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile removed route: %v", err)
	}

	neighborTarget, err := snapshotFilePath(root, "/posts/current/")
	if err != nil {
		t.Fatalf("snapshotFilePath neighbor route: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(neighborTarget), 0o755); err != nil {
		t.Fatalf("MkdirAll neighbor route: %v", err)
	}
	if err := os.WriteFile(neighborTarget, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile neighbor route: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:       settings,
		publishedPosts: []PostRecord{current, older},
		postBySlug: map[string]PostRecord{
			current.Slug: current,
			older.Slug:   older,
		},
		postByID: map[string]PostRecord{
			current.ID: current,
			older.ID:   older,
		},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	req := revalidateRequest{
		Current:  mustMarshalJSONRaw(t, newer),
		Original: mustMarshalJSONRaw(t, original),
	}
	err = withSnapshotBuildContext(ctx, func() error {
		return revalidatePost(root, req)
	})
	if err != nil {
		t.Fatalf("revalidatePost: %v", err)
	}

	if _, err := os.Stat(removedTarget); !os.IsNotExist(err) {
		t.Fatalf("removed route should be deleted, stat err = %v", err)
	}

	body, err := os.ReadFile(neighborTarget)
	if err != nil {
		t.Fatalf("ReadFile neighbor route: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("neighbor route still stale")
	}
	if strings.Contains(text, `/posts/newer/`) {
		t.Fatalf("neighbor route still references removed post: %s", text)
	}
}

func TestRevalidateTranslationRemovesOldRouteAndUpdatesNeighborWhenUnpublishedWithDAG(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	sourceNewer := PostRecord{
		ID:          "post-newer",
		Slug:        "newer",
		Title:       "Newer",
		Body:        "<p>newer</p>",
		Published:   true,
		PublishedAt: "2026-04-17T10:00:00Z",
	}
	original := PostTranslationRecord{
		ID:          "tr-newer",
		SourcePost:  sourceNewer.ID,
		Locale:      "ru",
		Slug:        "newer-ru",
		Title:       "Newer RU",
		Body:        "<p>newer ru</p>",
		Published:   true,
		PublishedAt: "2026-04-17T10:00:00Z",
	}
	newer := PostTranslationRecord{
		ID:          "tr-newer",
		SourcePost:  sourceNewer.ID,
		Locale:      "ru",
		Slug:        "newer-ru",
		Title:       "Newer RU",
		Body:        "<p>newer ru</p>",
		Published:   false,
		PublishedAt: "2026-04-17T10:00:00Z",
	}
	sourceCurrent := PostRecord{
		ID:          "post-current",
		Slug:        "current",
		Title:       "Current",
		Body:        "<p>current</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	current := PostTranslationRecord{
		ID:          "tr-current",
		SourcePost:  sourceCurrent.ID,
		Locale:      "ru",
		Slug:        "current-ru",
		Title:       "Current RU",
		Body:        "<p>current ru</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	sourceOlder := PostRecord{
		ID:          "post-older",
		Slug:        "older",
		Title:       "Older",
		Body:        "<p>older</p>",
		Published:   true,
		PublishedAt: "2026-04-15T10:00:00Z",
	}
	older := PostTranslationRecord{
		ID:          "tr-older",
		SourcePost:  sourceOlder.ID,
		Locale:      "ru",
		Slug:        "older-ru",
		Title:       "Older RU",
		Body:        "<p>older ru</p>",
		Published:   true,
		PublishedAt: "2026-04-15T10:00:00Z",
	}

	removedTarget, err := snapshotFilePath(root, "/ru/posts/newer-ru/")
	if err != nil {
		t.Fatalf("snapshotFilePath removed localized route: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(removedTarget), 0o755); err != nil {
		t.Fatalf("MkdirAll removed localized route: %v", err)
	}
	if err := os.WriteFile(removedTarget, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile removed localized route: %v", err)
	}

	neighborTarget, err := snapshotFilePath(root, "/ru/posts/current-ru/")
	if err != nil {
		t.Fatalf("snapshotFilePath neighbor localized route: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(neighborTarget), 0o755); err != nil {
		t.Fatalf("MkdirAll neighbor localized route: %v", err)
	}
	if err := os.WriteFile(neighborTarget, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile neighbor localized route: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:       settings,
		publishedPosts: []PostRecord{sourceCurrent, sourceOlder, sourceNewer},
		postBySlug: map[string]PostRecord{
			sourceCurrent.Slug: sourceCurrent,
			sourceOlder.Slug:   sourceOlder,
			sourceNewer.Slug:   sourceNewer,
		},
		postByID: map[string]PostRecord{
			sourceCurrent.ID: sourceCurrent,
			sourceOlder.ID:   sourceOlder,
			sourceNewer.ID:   sourceNewer,
		},
		pageByURL: map[string]PageRecord{},
		translationByKey: map[string]PostTranslationRecord{
			"ru|" + current.Slug: current,
			"ru|" + older.Slug:   older,
		},
		translationsBySource: map[string][]PostTranslationRecord{
			sourceCurrent.ID: []PostTranslationRecord{current},
			sourceOlder.ID:   []PostTranslationRecord{older},
		},
		translationsByLocale: map[string][]PostTranslationRecord{
			"ru": []PostTranslationRecord{current, older},
		},
		postsByTag:      map[string][]PostRecord{},
		postsByCategory: map[string][]PostRecord{},
		archiveIndex:    map[string]archiveListing{},
	}

	req := revalidateRequest{
		Current:  mustMarshalJSONRaw(t, newer),
		Original: mustMarshalJSONRaw(t, original),
	}
	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateTranslation(root, req)
	})
	if err != nil {
		t.Fatalf("revalidateTranslation: %v", err)
	}

	if _, err := os.Stat(removedTarget); !os.IsNotExist(err) {
		t.Fatalf("removed localized route should be deleted, stat err = %v", err)
	}

	body, err := os.ReadFile(neighborTarget)
	if err != nil {
		t.Fatalf("ReadFile neighbor localized route: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("neighbor localized route still stale")
	}
	if strings.Contains(text, `/ru/posts/newer-ru/`) {
		t.Fatalf("neighbor localized route still references removed translation: %s", text)
	}
}

func TestRevalidateAdjacentPostContextUpdatesNeighborNavigation(t *testing.T) {
	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "en"

	older := PostRecord{
		ID:          "post-older",
		Slug:        "weekly-fail2ban-report-public-monitoring",
		Title:       "Weekly fail2ban report public monitoring",
		Body:        "<p>older</p>",
		Published:   true,
		PublishedAt: "2026-04-10T10:00:00Z",
	}
	current := PostRecord{
		ID:          "post-current",
		Slug:        "ultimate-freebsd-pf-conf",
		Title:       "Ultimate FreeBSD pf.conf",
		Body:        "<p>current</p>",
		Published:   true,
		PublishedAt: "2026-04-11T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/posts/"+older.Slug+"/")
	if err != nil {
		t.Fatalf("snapshotFilePath older: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll older: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile older: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 nil,
		publishedPosts:       []PostRecord{current, older},
		publishedPages:       nil,
		tags:                 nil,
		categories:           nil,
		postBySlug:           map[string]PostRecord{older.Slug: older, current.Slug: current},
		postByID:             map[string]PostRecord{older.ID: older, current.ID: current},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex: map[string]archiveListing{
			"/archive/": {posts: []PostRecord{current, older}, pageCount: 1},
		},
	}

	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateAdjacentPostContext(root, &current, nil)
	})
	if err != nil {
		t.Fatalf("revalidateAdjacentPostContext: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile older: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("older route still stale")
	}
	if !strings.Contains(text, "/posts/"+current.Slug+"/") {
		t.Fatalf("older route missing updated newer link: %s", text)
	}
}

func TestRevalidateAdjacentPostContextSkipsLegacyPathWhenDAGEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"

	older := PostRecord{
		ID:          "post-older",
		Slug:        "older",
		Title:       "Older",
		Body:        "<p>older</p>",
		Published:   true,
		PublishedAt: "2026-04-10T10:00:00Z",
	}
	current := PostRecord{
		ID:          "post-current",
		Slug:        "current",
		Title:       "Current",
		Body:        "<p>current</p>",
		Published:   true,
		PublishedAt: "2026-04-11T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/posts/"+older.Slug+"/")
	if err != nil {
		t.Fatalf("snapshotFilePath older: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll older: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile older: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{current, older},
		postBySlug:           map[string]PostRecord{older.Slug: older, current.Slug: current},
		postByID:             map[string]PostRecord{older.ID: older, current.ID: current},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateAdjacentPostContext(root, &current, nil)
	})
	if err != nil {
		t.Fatalf("revalidateAdjacentPostContext: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile older: %v", err)
	}
	if string(body) != "stale" {
		t.Fatalf("legacy adjacent rebuild should be skipped when DAG is enabled: %s", string(body))
	}
}

func TestRevalidateAdjacentTranslationContextSkipsLegacyPathWhenDAGEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	sourceOlder := PostRecord{
		ID:          "post-older",
		Slug:        "older",
		Title:       "Older",
		Body:        "<p>older</p>",
		Published:   true,
		PublishedAt: "2026-04-10T10:00:00Z",
	}
	sourceCurrent := PostRecord{
		ID:          "post-current",
		Slug:        "current",
		Title:       "Current",
		Body:        "<p>current</p>",
		Published:   true,
		PublishedAt: "2026-04-11T10:00:00Z",
	}
	olderTranslation := PostTranslationRecord{
		ID:          "tr-older",
		SourcePost:  sourceOlder.ID,
		Locale:      "ru",
		Slug:        "older-ru",
		Title:       "Older RU",
		Body:        "<p>older ru</p>",
		Published:   true,
		PublishedAt: "2026-04-10T10:00:00Z",
	}
	currentTranslation := PostTranslationRecord{
		ID:          "tr-current",
		SourcePost:  sourceCurrent.ID,
		Locale:      "ru",
		Slug:        "current-ru",
		Title:       "Current RU",
		Body:        "<p>current ru</p>",
		Published:   true,
		PublishedAt: "2026-04-11T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/ru/posts/"+olderTranslation.Slug+"/")
	if err != nil {
		t.Fatalf("snapshotFilePath older translation: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll older translation: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile older translation: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:       settings,
		publishedPosts: []PostRecord{sourceCurrent, sourceOlder},
		postBySlug: map[string]PostRecord{
			sourceOlder.Slug:   sourceOlder,
			sourceCurrent.Slug: sourceCurrent,
		},
		postByID: map[string]PostRecord{
			sourceOlder.ID:   sourceOlder,
			sourceCurrent.ID: sourceCurrent,
		},
		pageByURL: map[string]PageRecord{},
		translationByKey: map[string]PostTranslationRecord{
			"ru|" + olderTranslation.Slug:   olderTranslation,
			"ru|" + currentTranslation.Slug: currentTranslation,
		},
		translationsBySource: map[string][]PostTranslationRecord{
			sourceOlder.ID:   []PostTranslationRecord{olderTranslation},
			sourceCurrent.ID: []PostTranslationRecord{currentTranslation},
		},
		translationsByLocale: map[string][]PostTranslationRecord{
			"ru": []PostTranslationRecord{currentTranslation, olderTranslation},
		},
		postsByTag:      map[string][]PostRecord{},
		postsByCategory: map[string][]PostRecord{},
		archiveIndex:    map[string]archiveListing{},
	}

	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateAdjacentTranslationContext(root, &currentTranslation, nil)
	})
	if err != nil {
		t.Fatalf("revalidateAdjacentTranslationContext: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile older translation: %v", err)
	}
	if string(body) != "stale" {
		t.Fatalf("legacy translation adjacent rebuild should be skipped when DAG is enabled: %s", string(body))
	}
}

func TestRevalidateSourcePostFamilySkipsLegacyPathWhenDAGEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	source := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>Hello</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	translation := PostTranslationRecord{
		ID:          "tr-1",
		SourcePost:  source.ID,
		Locale:      "ru",
		Slug:        "privet",
		Title:       "Privet",
		Body:        "<p>Privet</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	for _, route := range []string{"/posts/hello/", "/ru/posts/privet/"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", route, err)
		}
		if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", route, err)
		}
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{source},
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		return revalidateSourcePostFamily(root, &source, nil)
	})
	if err != nil {
		t.Fatalf("revalidateSourcePostFamily: %v", err)
	}

	for _, route := range []string{"/posts/hello/", "/ru/posts/privet/"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		body, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", route, err)
		}
		if string(body) != "stale" {
			t.Fatalf("legacy source family rebuild should be skipped when DAG is enabled for %q: %s", route, string(body))
		}
	}
}

func TestRevalidateTranslationContextSkipsLegacyFamilyPathWhenDAGEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.WelcomeText = "Welcome"
	settings.HomePageSize = 3
	settings.ArchivePageSize = 10
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	source := PostRecord{
		ID:          "source-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-11 12:33:00.000Z",
	}
	translation := PostTranslationRecord{
		SourcePost:  "source-1",
		Locale:      "ru",
		Slug:        "privet",
		Title:       "Privet",
		Body:        "<p>body</p>",
		Published:   true,
		PublishedAt: "2026-04-11 12:33:00.000Z",
	}

	for _, route := range []string{"/posts/hello/", "/ru/posts/privet/", "/", "/archive/"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", route, err)
		}
		if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", route, err)
		}
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{source},
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex: map[string]archiveListing{
			"/archive/": {posts: []PostRecord{source}, pageCount: 1},
		},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		return revalidateTranslationContext(root, &translation, nil)
	})
	if err != nil {
		t.Fatalf("revalidateTranslationContext: %v", err)
	}

	for _, route := range []string{"/posts/hello/", "/ru/posts/privet/"} {
		target, err := snapshotFilePath(root, route)
		if err != nil {
			t.Fatalf("snapshotFilePath(%q): %v", route, err)
		}
		body, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", route, err)
		}
		if string(body) != "stale" {
			t.Fatalf("legacy translation family rebuild should be skipped when DAG is enabled for %q: %s", route, string(body))
		}
	}
}

func TestRenderLegacyPostRouteUsesFallback(t *testing.T) {
	body, ok, err := renderLegacyPostRoute(func() (string, bool) {
		return "<html>Hello</html>", true
	})
	if err != nil {
		t.Fatalf("renderLegacyPostRoute: %v", err)
	}
	if !ok {
		t.Fatal("renderLegacyPostRoute ok = false")
	}
	if string(body) != "<html>Hello</html>" {
		t.Fatalf("renderLegacyPostRoute body = %q", string(body))
	}
}

func TestDAGAffectedRouteKeysFromChangedFiltersRoutes(t *testing.T) {
	t.Parallel()

	engine := dag.NewEngine()
	resolveCtx := engine.NewContext()

	source := dag.NodeKey{Kind: "source", ID: "a"}
	input := dag.NodeKey{Kind: nodePostRenderInput, Scope: nodeScopeBase, ID: "hello"}
	route := routeNodeKey("/posts/hello/")

	engine.Register(source.Kind, resolverFunc(func(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
		return dag.ResolveResult{Value: "source"}, nil
	}))
	engine.Register(input.Kind, resolverFunc(func(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
		return dag.ResolveResult{Value: "input", Deps: []dag.NodeKey{source}}, nil
	}))
	engine.Register(route.Kind, resolverFunc(func(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
		return dag.ResolveResult{Value: "route", Deps: []dag.NodeKey{input}}, nil
	}))

	if _, err := engine.Resolve(resolveCtx, route); err != nil {
		t.Fatalf("Resolve(route): %v", err)
	}

	affected := dagAffectedRouteKeysFromChanged(resolveCtx, []dag.NodeKey{source})
	if len(affected) != 1 || affected[0] != route {
		t.Fatalf("dagAffectedRouteKeysFromChanged = %#v", affected)
	}
}

func TestRevalidateDAGAffectedPostRoutesUpdatesNeighborRoute(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"

	newer := PostRecord{
		ID:          "post-newer",
		Slug:        "newer",
		Title:       "Newer",
		Body:        "<p>Newer</p>",
		Published:   true,
		PublishedAt: "2026-04-17T10:00:00Z",
	}
	current := PostRecord{
		ID:          "post-current",
		Slug:        "current",
		Title:       "Current",
		Body:        "<p>Current</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	older := PostRecord{
		ID:          "post-older",
		Slug:        "older",
		Title:       "Older",
		Body:        "<p>Older</p>",
		Published:   true,
		PublishedAt: "2026-04-15T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/posts/current/")
	if err != nil {
		t.Fatalf("snapshotFilePath current: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll current: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile current: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{newer, current, older},
		postBySlug:           map[string]PostRecord{newer.Slug: newer, current.Slug: current, older.Slug: older},
		postByID:             map[string]PostRecord{newer.ID: newer, current.ID: current, older.ID: older},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateDAGAffectedPostRoutes(root, []dag.NodeKey{postBySlugNodeKey("", newer.Slug)}, nil)
	})
	if err != nil {
		t.Fatalf("revalidateDAGAffectedPostRoutes: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile current: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("current route still stale")
	}
	if !strings.Contains(text, "/posts/newer/") {
		t.Fatalf("current route missing updated newer link: %s", text)
	}
}

func TestRevalidateDAGAffectedPostRoutesUpdatesBaseRouteForTranslationChange(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	source := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>Hello</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	translation := PostTranslationRecord{
		ID:          "tr-1",
		SourcePost:  source.ID,
		Locale:      "ru",
		Slug:        "privet",
		Title:       "Privet",
		Body:        "<p>Privet</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/posts/hello/")
	if err != nil {
		t.Fatalf("snapshotFilePath base: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll base: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile base: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{source},
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateDAGAffectedPostRoutes(root, []dag.NodeKey{postBySlugNodeKey("ru", translation.Slug)}, nil)
	})
	if err != nil {
		t.Fatalf("revalidateDAGAffectedPostRoutes: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile base: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("base route still stale")
	}
	if !strings.Contains(text, `/ru/posts/privet/`) {
		t.Fatalf("base route missing updated translation link: %s", text)
	}
}

func TestRevalidateDAGAffectedPostRoutesUpdatesLocalizedRouteForSourceChange(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	source := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>Hello</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	translation := PostTranslationRecord{
		ID:          "tr-1",
		SourcePost:  source.ID,
		Locale:      "ru",
		Slug:        "privet",
		Title:       "Privet",
		Body:        "<p>Privet</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	target, err := snapshotFilePath(root, "/ru/posts/privet/")
	if err != nil {
		t.Fatalf("snapshotFilePath localized: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll localized: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile localized: %v", err)
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		publishedPosts:       []PostRecord{source},
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err = withSnapshotBuildContext(ctx, func() error {
		return revalidateDAGAffectedPostRoutes(root, []dag.NodeKey{postBySlugNodeKey("", source.Slug)}, nil)
	})
	if err != nil {
		t.Fatalf("revalidateDAGAffectedPostRoutes: %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile localized: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "stale") {
		t.Fatalf("localized route still stale")
	}
	if !strings.Contains(text, `href="/posts/hello/"`) {
		t.Fatalf("localized route missing updated source language link: %s", text)
	}
}

func TestDAGAffectedRouteReasonsIncludesDependencyPath(t *testing.T) {
	t.Parallel()

	engine := dag.NewEngine()
	resolveCtx := engine.NewContext()

	source := dag.NodeKey{Kind: "source", ID: "newer"}
	input := dag.NodeKey{Kind: nodePostRenderInput, Scope: nodeScopeBase, ID: "current"}
	route := routeNodeKey("/posts/current/")

	engine.Register(source.Kind, resolverFunc(func(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
		return dag.ResolveResult{Value: "source"}, nil
	}))
	engine.Register(input.Kind, resolverFunc(func(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
		return dag.ResolveResult{Value: "input", Deps: []dag.NodeKey{source}}, nil
	}))
	engine.Register(route.Kind, resolverFunc(func(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
		return dag.ResolveResult{Value: "route", Deps: []dag.NodeKey{input}}, nil
	}))

	if _, err := engine.Resolve(resolveCtx, route); err != nil {
		t.Fatalf("Resolve(route): %v", err)
	}

	reasons := dagAffectedRouteReasons(resolveCtx, []dag.NodeKey{source}, route, nil)
	if len(reasons) != 1 {
		t.Fatalf("dagAffectedRouteReasons length = %d, want 1 (%#v)", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], source.String()) || !strings.Contains(reasons[0], route.String()) {
		t.Fatalf("dagAffectedRouteReasons = %#v", reasons)
	}
}

func TestDAGAffectedRouteReasonsIncludesExtraReasons(t *testing.T) {
	t.Parallel()

	route := routeNodeKey("/posts/current/")
	reasons := dagAffectedRouteReasons(nil, nil, route, []string{"adjacent bridge from site.post_by_slug|base|newer|"})
	if len(reasons) != 1 {
		t.Fatalf("dagAffectedRouteReasons length = %d, want 1 (%#v)", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], "adjacent bridge") {
		t.Fatalf("dagAffectedRouteReasons = %#v", reasons)
	}
}

func TestSnapshotBuildContextPostRouteKeysIncludesBaseAndLocalizedRoutes(t *testing.T) {
	t.Parallel()

	ctx := &snapshotBuildContext{
		publishedPosts: []PostRecord{
			{Slug: "hello"},
		},
		translationsByLocale: map[string][]PostTranslationRecord{
			"ru": []PostTranslationRecord{{Slug: "privet"}},
		},
		translationByKey: map[string]PostTranslationRecord{
			"ru|privet": {Slug: "privet", Locale: "ru"},
		},
	}

	keys := ctx.postRouteKeys()
	if len(keys) != 2 {
		t.Fatalf("postRouteKeys length = %d, want 2 (%#v)", len(keys), keys)
	}
	if keys[0] != routeNodeKey("/posts/hello/") && keys[1] != routeNodeKey("/posts/hello/") {
		t.Fatalf("postRouteKeys missing base route: %#v", keys)
	}
	if keys[0] != routeNodeKey("/ru/posts/privet/") && keys[1] != routeNodeKey("/ru/posts/privet/") {
		t.Fatalf("postRouteKeys missing localized route: %#v", keys)
	}
}

type resolverFunc func(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error)

func (f resolverFunc) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	return f(ctx, key)
}

func mustMarshalJSONRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return json.RawMessage(data)
}
