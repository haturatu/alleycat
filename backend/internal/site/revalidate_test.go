package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
