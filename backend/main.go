package main

import (
	"database/sql"
	"errors"
	"log"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
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

	_, err = ensureCollection(app, core.CollectionTypeBase, "posts", func(c *core.Collection) error {
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

		addIndexIfMissing(c, "CREATE UNIQUE INDEX `idx_posts_slug` ON `posts` (slug)")

		return nil
	})
	if err != nil {
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

		return nil
	})
	if err != nil {
		return err
	}

	return app.ReloadCachedCollections()
}

func migrateMediaPaths(app core.App) error {
	media, err := app.FindCollectionByNameOrId("media")
	if err != nil {
		return err
	}
	records := []*core.Record{}
	if err := app.RecordQuery(media).All(&records); err != nil {
		return err
	}
	for _, record := range records {
		pathValue := record.GetString("path")
		if pathValue != "" {
			continue
		}
		path := strings.TrimSpace(record.GetString("caption"))
		if path != "" && strings.HasPrefix(path, "/uploads/") {
			record.Set("path", path)
			if err := app.Save(record); err != nil {
				return err
			}
			continue
		}
		filename := strings.TrimSpace(record.GetString("file"))
		if filename == "" {
			continue
		}
		normalized := normalizeUploadFilename(filename)
		if normalized == "" {
			continue
		}
		record.Set("path", "/uploads/"+normalized)
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func normalizeUploadFilename(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return filename
	}
	base := strings.TrimSuffix(filename, ext)
	underscore := strings.LastIndex(base, "_")
	if underscore == -1 {
		return filename
	}
	suffix := base[underscore+1:]
	if len(suffix) < 6 || len(suffix) > 16 {
		return filename
	}
	for _, r := range suffix {
		if (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return filename
		}
	}
	return base[:underscore] + ext
}

func ensureCollection(app core.App, typ, name string, configure func(c *core.Collection) error) (*core.Collection, error) {
	existing, err := app.FindCollectionByNameOrId(name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var collection *core.Collection
	if existing == nil {
		collection = core.NewCollection(typ, name)
	} else {
		collection = existing
	}

	if err := configure(collection); err != nil {
		return nil, err
	}

	if err := app.Save(collection); err != nil {
		return nil, err
	}

	return collection, nil
}

func addFieldIfMissing(collection *core.Collection, field core.Field) {
	if collection.Fields.GetByName(field.GetName()) != nil {
		return
	}
	collection.Fields.Add(field)
}

func addIndexIfMissing(collection *core.Collection, index string) {
	for _, existing := range collection.Indexes {
		if existing == index {
			return
		}
	}
	collection.Indexes = append(collection.Indexes, index)
}

func setRuleIfNil(ptr **string, rule string) {
	if *ptr != nil {
		return
	}
	value := rule
	*ptr = &value
}

func main() {
	app := pocketbase.New()

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return ensureCollections(e.App)
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
