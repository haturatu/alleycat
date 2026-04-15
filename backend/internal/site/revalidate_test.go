package site

import (
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
		return revalidateTranslationContext(root, settings, &translation, nil)
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
		return revalidateAdjacentPostContext(root, settings, &current, nil)
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
		return revalidateAdjacentPostContext(root, settings, &current, nil)
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
		return revalidateAdjacentTranslationContext(root, settings, &currentTranslation, nil)
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
		return revalidateSourcePostFamily(root, settings, &source, nil)
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
		return revalidateTranslationContext(root, settings, &translation, nil)
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

func TestRenderPostRouteForRevalidationUsesDAGWhenEnabled(t *testing.T) {
	t.Setenv("SITE_DAG_POST_ROUTES", "true")

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"

	post := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>Body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 nil,
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

	err := withSnapshotBuildContext(ctx, func() error {
		body, ok, err := renderPostRouteForRevalidation("/posts/hello/", func() (string, bool) {
			t.Fatal("fallback should not be used when DAG is enabled")
			return "", false
		})
		if err != nil {
			t.Fatalf("renderPostRouteForRevalidation: %v", err)
		}
		if !ok {
			t.Fatal("renderPostRouteForRevalidation ok = false")
		}
		if !strings.Contains(string(body), "Hello") {
			t.Fatalf("route body missing post title: %s", string(body))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withSnapshotBuildContext: %v", err)
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
		return revalidateDAGAffectedPostRoutes(root, []dag.NodeKey{postBySlugNodeKey("", newer.Slug)})
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
		return revalidateDAGAffectedPostRoutes(root, []dag.NodeKey{postBySlugNodeKey("ru", translation.Slug)})
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
		return revalidateDAGAffectedPostRoutes(root, []dag.NodeKey{postBySlugNodeKey("", source.Slug)})
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

	reasons := dagAffectedRouteReasons(resolveCtx, []dag.NodeKey{source}, route)
	if len(reasons) != 1 {
		t.Fatalf("dagAffectedRouteReasons length = %d, want 1 (%#v)", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], source.String()) || !strings.Contains(reasons[0], route.String()) {
		t.Fatalf("dagAffectedRouteReasons = %#v", reasons)
	}
}

type resolverFunc func(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error)

func (f resolverFunc) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	return f(ctx, key)
}
