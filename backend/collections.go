package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func ensureCollections(app core.App) error {
	cmsUsers, err := ensureCollection(app, core.CollectionTypeAuth, "cms_users", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `@request.auth.id != ""`)
		setRuleIfNil(&c.ViewRule, `@request.auth.id != ""`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && @request.auth.role = "admin"`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && @request.auth.role = "admin"`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && @request.auth.role = "admin"`)

		addFieldIfMissing(c, &core.TextField{
			Name:     "name",
			Required: true,
			Max:      120,
		})
		addFieldIfMissing(c, &core.SelectField{
			Name:      "role",
			Required:  true,
			Values:    []string{"admin", "editor", "viewer"},
			MaxSelect: 1,
		})

		return nil
	})
	if err != nil {
		return err
	}

	_, err = ensureCollection(app, core.CollectionTypeAuth, "users", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `id = @request.auth.id`)
		setRuleIfNil(&c.ViewRule, `id = @request.auth.id`)
		setRuleIfNil(&c.CreateRule, ``)
		setRuleIfNil(&c.UpdateRule, `id = @request.auth.id`)
		setRuleIfNil(&c.DeleteRule, `id = @request.auth.id`)

		addFieldIfMissing(c, &core.TextField{
			Name: "name",
			Max:  255,
		})
		addFieldIfMissing(c, &core.FileField{
			Name:        "avatar",
			MaxSelect:   1,
			MimeTypes:   []string{"image/jpeg", "image/png", "image/svg+xml", "image/gif", "image/webp"},
			Protected:   false,
			Required:    false,
			MaxSize:     0,
			Thumbs:      nil,
			Presentable: false,
		})

		return nil
	})
	if err != nil {
		return err
	}

	_, err = ensureCollection(app, core.CollectionTypeBase, "media", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `@request.auth.id != "" || public = true`)
		setRuleIfNil(&c.ViewRule, `@request.auth.id != "" || public = true`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)

		addFieldIfMissing(c, &core.FileField{
			Name:      "file",
			MaxSelect: 1,
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "path",
			Max:  200,
		})
		addFieldIfMissing(c, &core.AutodateField{
			Name:     "uploaded_at",
			OnCreate: true,
			OnUpdate: false,
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "alt",
			Max:  200,
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "caption",
		})
		addFieldIfMissing(c, &core.BoolField{
			Name: "public",
		})

		return nil
	})
	if err != nil {
		return err
	}

	if err := migrateMediaPaths(app); err != nil {
		return err
	}

	_, err = ensureCollection(app, core.CollectionTypeBase, "pages", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `@request.auth.id != "" || (published = true && published_at <= @now)`)
		setRuleIfNil(&c.ViewRule, `@request.auth.id != "" || (published = true && published_at <= @now)`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)

		addFieldIfMissing(c, &core.TextField{
			Name:     "title",
			Required: true,
		})
		addFieldIfMissing(c, &core.TextField{
			Name:     "slug",
			Required: true,
		})
		addFieldIfMissing(c, &core.TextField{
			Name:     "url",
			Required: true,
		})
		addFieldIfMissing(c, &core.EditorField{
			Name:        "body",
			Required:    true,
			ConvertURLs: false,
		})
		addFieldIfMissing(c, &core.BoolField{
			Name: "menuVisible",
		})
		addFieldIfMissing(c, &core.NumberField{
			Name: "menuOrder",
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "menuTitle",
		})
		addFieldIfMissing(c, &core.DateField{
			Name: "published_at",
		})
		addFieldIfMissing(c, &core.BoolField{
			Name: "published",
		})

		addIndexIfMissing(c, "CREATE UNIQUE INDEX `idx_pages_slug` ON `pages` (slug)")
		addIndexIfMissing(c, "CREATE UNIQUE INDEX `idx_pages_url` ON `pages` (url)")

		return nil
	})
	if err != nil {
		return err
	}

	postsCollection, err := ensureCollection(app, core.CollectionTypeBase, "posts", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `@request.auth.id != "" || (published = true && published_at <= @now)`)
		setRuleIfNil(&c.ViewRule, `@request.auth.id != "" || (published = true && published_at <= @now)`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)

		addFieldIfMissing(c, &core.TextField{
			Name:     "title",
			Required: true,
		})
		addFieldIfMissing(c, &core.TextField{
			Name:     "slug",
			Required: true,
		})
		addFieldIfMissing(c, &core.EditorField{
			Name:        "body",
			Required:    true,
			ConvertURLs: false,
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "excerpt",
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "tags",
		})
		addFieldIfMissing(c, &core.DateField{
			Name: "published_at",
		})
		addFieldIfMissing(c, &core.BoolField{
			Name: "published",
		})
		addFieldIfMissing(c, &core.TextField{
			Name: "category",
		})
		addFieldIfMissing(c, &core.RelationField{
			Name:         "author",
			CollectionId: cmsUsers.Id,
			MaxSelect:    1,
			MinSelect:    0,
		})
		addFieldIfMissing(c, &core.FileField{
			Name:      "featured_image",
			MaxSelect: 1,
		})
		addFieldIfMissing(c, &core.FileField{
			Name:      "attachments",
			MaxSelect: 10,
		})

		removeIndexesByName(c, "idx_posts_slug_locale")
		addIndexIfMissing(c, "CREATE UNIQUE INDEX `idx_posts_slug` ON `posts` (slug)")

		return nil
	})
	if err != nil {
		return err
	}

	_, err = ensureCollection(app, core.CollectionTypeBase, "post_translations", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `@request.auth.id != "" || (published = true && published_at <= @now)`)
		setRuleIfNil(&c.ViewRule, `@request.auth.id != "" || (published = true && published_at <= @now)`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)

		addFieldIfMissing(c, &core.RelationField{
			Name:         "source_post",
			CollectionId: postsCollection.Id,
			MaxSelect:    1,
			MinSelect:    1,
		})
		addFieldIfMissing(c, &core.TextField{
			Name:     "locale",
			Required: true,
			Max:      20,
		})
		addFieldIfMissing(c, &core.TextField{
			Name:     "title",
			Required: true,
		})
		addFieldIfMissing(c, &core.TextField{
			Name:     "slug",
			Required: true,
		})
		addFieldIfMissing(c, &core.EditorField{
			Name:        "body",
			Required:    true,
			ConvertURLs: false,
		})
		addFieldIfMissing(c, &core.TextField{Name: "excerpt"})
		addFieldIfMissing(c, &core.TextField{Name: "tags"})
		addFieldIfMissing(c, &core.TextField{Name: "category"})
		addFieldIfMissing(c, &core.RelationField{
			Name:         "author",
			CollectionId: cmsUsers.Id,
			MaxSelect:    1,
			MinSelect:    0,
		})
		addFieldIfMissing(c, &core.DateField{Name: "published_at"})
		addFieldIfMissing(c, &core.BoolField{Name: "published"})
		addFieldIfMissing(c, &core.BoolField{Name: "translation_done"})
		addFieldIfMissing(c, &core.FileField{Name: "featured_image", MaxSelect: 1})
		addFieldIfMissing(c, &core.FileField{Name: "attachments", MaxSelect: 10})

		addIndexIfMissing(c, "CREATE UNIQUE INDEX `idx_post_translations_source_locale` ON `post_translations` (source_post, locale)")
		addIndexIfMissing(c, "CREATE INDEX `idx_post_translations_slug_locale` ON `post_translations` (slug, locale)")
		return nil
	})
	if err != nil {
		return err
	}

	if err := migrateLegacyPostTranslations(app, postsCollection); err != nil {
		return err
	}

	_, err = ensureCollection(app, core.CollectionTypeBase, "settings", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `id != ""`)
		setRuleIfNil(&c.ViewRule, `id != ""`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && (@request.auth.role = "admin" || @request.auth.role = "editor")`)

		addFieldIfMissing(c, &core.TextField{Name: "site_name"})
		addFieldIfMissing(c, &core.TextField{Name: "description"})
		addFieldIfMissing(c, &core.TextField{Name: "welcome_text"})
		addFieldIfMissing(c, &core.BoolField{Name: "enable_analytics"})
		addFieldIfMissing(c, &core.TextField{Name: "analytics_url"})
		addFieldIfMissing(c, &core.TextField{Name: "analytics_site_id"})
		addFieldIfMissing(c, &core.BoolField{Name: "enable_ads"})
		addFieldIfMissing(c, &core.TextField{Name: "ads_client"})
		addFieldIfMissing(c, &core.BoolField{Name: "enable_code_highlight"})
		addFieldIfMissing(c, &core.TextField{Name: "highlight_theme"})
		addFieldIfMissing(c, &core.NumberField{Name: "archive_page_size"})
		addFieldIfMissing(c, &core.NumberField{Name: "excerpt_length"})
		addFieldIfMissing(c, &core.NumberField{Name: "home_page_size"})
		addFieldIfMissing(c, &core.BoolField{Name: "show_toc"})
		addFieldIfMissing(c, &core.BoolField{Name: "show_archive_tags"})
		addFieldIfMissing(c, &core.BoolField{Name: "show_tags"})
		addFieldIfMissing(c, &core.BoolField{Name: "show_categories"})
		addFieldIfMissing(c, &core.BoolField{Name: "show_archive_search"})
		addFieldIfMissing(c, &core.TextField{Name: "home_top_image"})
		addFieldIfMissing(c, &core.TextField{Name: "home_top_image_alt"})
		addFieldIfMissing(c, &core.TextField{Name: "footer_html"})
		addFieldIfMissing(c, &core.TextField{Name: "theme", Max: 40})
		addFieldIfMissing(c, &core.TextField{Name: "site_url"})
		addFieldIfMissing(c, &core.TextField{Name: "site_language"})
		addFieldIfMissing(c, &core.BoolField{Name: "enable_feed_xml"})
		addFieldIfMissing(c, &core.BoolField{Name: "enable_feed_json"})
		addFieldIfMissing(c, &core.NumberField{Name: "feed_items_limit"})
		addFieldIfMissing(c, &core.BoolField{Name: "enable_post_translation"})
		addFieldIfMissing(c, &core.TextField{Name: "translation_source_locale"})
		addFieldIfMissing(c, &core.TextField{Name: "translation_locales"})
		addFieldIfMissing(c, &core.TextField{Name: "translation_model"})
		addFieldIfMissing(c, &core.NumberField{Name: "translation_requests_per_minute"})
		existingGeminiKey := c.Fields.GetByName("gemini_api_key")
		if existingGeminiKey == nil {
			c.Fields.Add(&core.TextField{Name: "gemini_api_key", Hidden: true})
		} else {
			textField, ok := existingGeminiKey.(*core.TextField)
			if !ok {
				return fmt.Errorf("settings.gemini_api_key field must be a text field")
			}
			textField.Hidden = true
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = ensureCollection(app, core.CollectionTypeBase, "app_secrets", func(c *core.Collection) error {
		setRuleIfNil(&c.ListRule, `@request.auth.id != "" && @request.auth.role = "admin"`)
		setRuleIfNil(&c.ViewRule, `@request.auth.id != "" && @request.auth.role = "admin"`)
		setRuleIfNil(&c.CreateRule, `@request.auth.id != "" && @request.auth.role = "admin"`)
		setRuleIfNil(&c.UpdateRule, `@request.auth.id != "" && @request.auth.role = "admin"`)
		setRuleIfNil(&c.DeleteRule, `@request.auth.id != "" && @request.auth.role = "admin"`)

		addFieldIfMissing(c, &core.TextField{Name: "gemini_api_key"})
		return nil
	})
	if err != nil {
		return err
	}

	if err := migrateGeminiAPIKeyToSecrets(app); err != nil {
		return err
	}
	if err := ensureSettingsRecord(app); err != nil {
		return err
	}

	return app.ReloadCachedCollections()
}

func migrateLegacyPostTranslations(app core.App, postsCollection *core.Collection) error {
	sourceField := postsCollection.Fields.GetByName("source_post")
	localeField := postsCollection.Fields.GetByName("locale")
	if sourceField == nil || localeField == nil {
		return nil
	}

	legacyRecords, err := app.FindRecordsByFilter(
		postsCollection,
		"source_post != ''",
		"",
		0,
		0,
	)
	if err != nil {
		return err
	}
	if len(legacyRecords) == 0 {
		return nil
	}

	translationsCollection, err := app.FindCollectionByNameOrId("post_translations")
	if err != nil {
		return err
	}

	for _, legacy := range legacyRecords {
		sourcePostID := strings.TrimSpace(legacy.GetString("source_post"))
		locale := normalizeLocale(legacy.GetString("locale"))
		if sourcePostID == "" || locale == "" {
			continue
		}

		translated, findErr := app.FindFirstRecordByFilter(
			translationsCollection,
			"source_post = {:source} && locale = {:locale}",
			dbx.Params{"source": sourcePostID, "locale": locale},
		)
		if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
			return findErr
		}
		if translated == nil {
			translated = core.NewRecord(translationsCollection)
		}

		translated.Set("source_post", sourcePostID)
		translated.Set("locale", locale)
		translated.Set("title", legacy.GetString("title"))
		translated.Set("slug", legacy.GetString("slug"))
		translated.Set("body", legacy.GetString("body"))
		translated.Set("excerpt", legacy.GetString("excerpt"))
		translated.Set("tags", legacy.GetString("tags"))
		translated.Set("category", legacy.GetString("category"))
		translated.Set("author", legacy.GetString("author"))
		translated.Set("published", legacy.GetBool("published"))
		translated.Set("published_at", legacy.GetString("published_at"))
		translated.Set("translation_done", legacy.GetBool("translation_done"))
		translated.Set("featured_image", legacy.GetStringSlice("featured_image"))
		translated.Set("attachments", legacy.GetStringSlice("attachments"))

		if saveErr := app.Save(translated); saveErr != nil {
			return saveErr
		}
		if deleteErr := app.Delete(legacy); deleteErr != nil {
			return deleteErr
		}
	}

	return nil
}

func migrateGeminiAPIKeyToSecrets(app core.App) error {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) || settings == nil {
		return nil
	}

	legacyKey := strings.TrimSpace(settings.GetString("gemini_api_key"))
	secret, secretErr := app.FindFirstRecordByFilter("app_secrets", "id != ''")
	if secretErr != nil && !errors.Is(secretErr, sql.ErrNoRows) {
		return secretErr
	}

	if errors.Is(secretErr, sql.ErrNoRows) || secret == nil {
		if legacyKey == "" {
			return nil
		}
		collection, findErr := app.FindCollectionByNameOrId("app_secrets")
		if findErr != nil {
			return findErr
		}
		secret = core.NewRecord(collection)
	}

	currentSecret := strings.TrimSpace(secret.GetString("gemini_api_key"))
	if currentSecret == "" && legacyKey != "" {
		secret.Set("gemini_api_key", legacyKey)
		if saveErr := app.Save(secret); saveErr != nil {
			return saveErr
		}
	}

	if legacyKey != "" {
		settings.Set("gemini_api_key", "")
		if saveErr := app.Save(settings); saveErr != nil {
			return saveErr
		}
	}

	return nil
}

func ensureSettingsRecord(app core.App) error {
	record, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if record != nil {
		return nil
	}
	collection, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		return err
	}
	record = core.NewRecord(collection)
	return app.Save(record)
}
