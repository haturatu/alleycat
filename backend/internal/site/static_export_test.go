package site

import (
	"strings"
	"testing"
)

func TestBuildSnapshotRenderTasksUsesDAGForPostRoutes(t *testing.T) {
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
		Slug:        "privet",
		Title:       "Privet",
		Body:        "<p>Body RU</p>",
		Published:   true,
		PublishedAt: "2026-04-16T10:00:00Z",
	}
	page := PageRecord{
		Title:      "About",
		URL:        "/about/",
		Published:  true,
		MenuVisible: true,
	}

	ctx := &snapshotBuildContext{
		settings:             settings,
		menu:                 []PageRecord{page},
		publishedPosts:       []PostRecord{source},
		publishedPages:       []PageRecord{page},
		postBySlug:           map[string]PostRecord{source.Slug: source},
		postByID:             map[string]PostRecord{source.ID: source},
		pageByURL:            map[string]PageRecord{page.URL: page},
		translationByKey:     map[string]PostTranslationRecord{"ru|" + translation.Slug: translation},
		translationsBySource: map[string][]PostTranslationRecord{source.ID: []PostTranslationRecord{translation}},
		translationsByLocale: map[string][]PostTranslationRecord{"ru": []PostTranslationRecord{translation}},
		postsByTag:           map[string][]PostRecord{},
		postsByCategory:      map[string][]PostRecord{},
		archiveIndex:         map[string]archiveListing{},
	}

	err := withSnapshotBuildContext(ctx, func() error {
		tasks := buildSnapshotRenderTasks(ctx, settings)
		if len(tasks) != 3 {
			t.Fatalf("buildSnapshotRenderTasks length = %d, want 3", len(tasks))
		}

		seen := map[string]bool{}
		for _, task := range tasks {
			body, ok := task.render()
			if !ok {
				t.Fatalf("task %q render ok = false", task.route)
			}
			seen[task.route] = true
			switch task.route {
			case "/posts/hello":
				if !strings.Contains(string(body), "Hello") {
					t.Fatalf("base post body missing title: %s", string(body))
				}
			case "/ru/posts/privet":
				if !strings.Contains(string(body), "Privet") {
					t.Fatalf("localized post body missing title: %s", string(body))
				}
			case "/about/":
				if !strings.Contains(string(body), "About") {
					t.Fatalf("page body missing title: %s", string(body))
				}
			}
		}
		for _, route := range []string{"/posts/hello", "/ru/posts/privet", "/about/"} {
			if !seen[route] {
				t.Fatalf("missing route %q in tasks", route)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withSnapshotBuildContext: %v", err)
	}
}
