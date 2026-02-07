package main

import (
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

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
