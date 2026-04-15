package site

import (
	"strings"
	"testing"
)

func TestSiteDAGResolvesBasePostRoute(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "en"

	post := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>Body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	older := PostRecord{
		ID:          "post-0",
		Slug:        "older",
		Title:       "Older",
		Body:        "<p>Older</p>",
		Published:   true,
		PublishedAt: "2026-04-15T10:00:00Z",
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 []PageRecord{{Title: "About", URL: "/about/", MenuVisible: true, Published: true}},
		publishedPosts:       []PostRecord{post, older},
		publishedPages:       []PageRecord{{Title: "About", URL: "/about/", MenuVisible: true, Published: true}},
		postBySlug:           map[string]PostRecord{post.Slug: post, older.Slug: older},
		postByID:             map[string]PostRecord{post.ID: post, older.ID: older},
		pageByURL:            map[string]PageRecord{"/about/": {Title: "About", URL: "/about/", MenuVisible: true, Published: true}},
		translationByKey:     map[string]PostTranslationRecord{},
		translationsBySource: map[string][]PostTranslationRecord{},
		translationsByLocale: map[string][]PostTranslationRecord{},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		engine := newSiteDAGEngine()
		resolveCtx := engine.NewContext()
		value, err := engine.Resolve(resolveCtx, routeNodeKey("/posts/hello/"))
		if err != nil {
			t.Fatalf("Resolve route: %v", err)
		}
		route, ok := value.(routeValue)
		if !ok {
			t.Fatalf("route value type = %T", value)
		}
		body := string(route.Body)
		if !strings.Contains(body, "Hello") {
			t.Fatalf("route body missing post title: %s", body)
		}
		if !strings.Contains(body, "/posts/older/") {
			t.Fatalf("route body missing older post nav: %s", body)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withSnapshotBuildContext: %v", err)
	}
}

func TestSiteDAGResolvesLocalizedPostRoute(t *testing.T) {
	t.Parallel()

	settings := defaultSettings()
	settings.SiteName = "Alleycat"
	settings.SiteLanguage = "ja"
	settings.TranslationSourceLocale = "ja"
	settings.TranslationLocales = "ru"

	source := PostRecord{
		ID:          "post-1",
		Slug:        "hello",
		Title:       "Hello",
		Body:        "<p>Body</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	translation := PostTranslationRecord{
		ID:          "tr-1",
		SourcePost:  source.ID,
		Locale:      "ru",
		Slug:        "hello",
		Title:       "Privet",
		Body:        "<p>Body RU</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 nil,
		publishedPosts:       []PostRecord{source},
		publishedPages:       nil,
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{},
		translationByKey:     map[string]PostTranslationRecord{"ru|hello": translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		engine := newSiteDAGEngine()
		resolveCtx := engine.NewContext()
		value, err := engine.Resolve(resolveCtx, routeNodeKey("/ru/posts/hello/"))
		if err != nil {
			t.Fatalf("Resolve localized route: %v", err)
		}
		route, ok := value.(routeValue)
		if !ok {
			t.Fatalf("route value type = %T", value)
		}
		body := string(route.Body)
		if !strings.Contains(body, "Privet") {
			t.Fatalf("localized route body missing title: %s", body)
		}
		if !strings.Contains(body, `href="/ru/posts/hello"`) {
			t.Fatalf("localized route body missing localized canonical path: %s", body)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withSnapshotBuildContext: %v", err)
	}
}
