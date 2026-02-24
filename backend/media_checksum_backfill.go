package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

type backfillStats struct {
	Scanned    int
	Updated    int
	Duplicates int
	Skipped    int
	Failed     int
}

type canonicalMedia struct {
	ID   string
	Path string
}

func registerMediaChecksumBackfillCommand(app *pocketbase.PocketBase) {
	cmd := &cobra.Command{
		Use:   "backfill-media-checksum",
		Short: "Backfill SHA-256 checksums for media records and align duplicate paths",
		RunE: func(command *cobra.Command, _ []string) error {
			if err := app.Bootstrap(); err != nil {
				return err
			}

			stats, err := backfillMediaChecksums(app)
			_, _ = fmt.Fprintf(
				command.OutOrStdout(),
				"Backfill result: scanned=%d updated=%d duplicates=%d skipped=%d failed=%d\n",
				stats.Scanned,
				stats.Updated,
				stats.Duplicates,
				stats.Skipped,
				stats.Failed,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Long = "Compute SHA-256 checksums for existing media records.\n" +
		"If duplicate content is found, duplicate records are pointed to the canonical path."

	app.RootCmd.AddCommand(cmd)
}

func backfillMediaChecksums(app core.App) (backfillStats, error) {
	stats := backfillStats{}

	mediaCollection, err := app.FindCollectionByNameOrId("media")
	if err != nil {
		return stats, err
	}

	records := make([]*core.Record, 0)
	if err := app.RecordQuery(mediaCollection).
		OrderBy("uploaded_at asc", "created asc", "id asc").
		All(&records); err != nil {
		return stats, err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return stats, err
	}
	defer fsys.Close()

	canonicalByChecksum := make(map[string]canonicalMedia, len(records))

	for _, record := range records {
		stats.Scanned++

		filename := strings.TrimSpace(record.GetString("file"))
		if filename == "" {
			stats.Skipped++
			continue
		}
		existingPath := strings.TrimSpace(record.GetString("path"))

		fileKey := record.BaseFilesPath() + "/" + filename
		reader, err := fsys.GetReader(fileKey)
		if err != nil {
			stats.Failed++
			continue
		}

		sumHex, hashErr := hashReaderSHA256Hex(reader)
		closeErr := reader.Close()
		if hashErr != nil || closeErr != nil {
			stats.Failed++
			continue
		}

		if canonical, ok := canonicalByChecksum[sumHex]; ok && canonical.ID != record.Id {
			stats.Duplicates++
			changed := false

			if canonical.Path != "" && existingPath != canonical.Path {
				record.Set("path", canonical.Path)
				changed = true
			}
			if strings.TrimSpace(record.GetString("checksum")) != "" {
				record.Set("checksum", "")
				changed = true
			}

			if changed {
				if err := app.Save(record); err != nil {
					stats.Failed++
					continue
				}
				stats.Updated++
			}
			continue
		}

		candidatePath := buildUploadPath(filename, sumHex)

		canonicalByChecksum[sumHex] = canonicalMedia{
			ID:   record.Id,
			Path: candidatePath,
		}

		changed := false
		if strings.TrimSpace(record.GetString("checksum")) != sumHex {
			record.Set("checksum", sumHex)
			changed = true
		}
		if candidatePath != "" && existingPath != candidatePath {
			record.Set("path", candidatePath)
			changed = true
		}

		if !changed {
			continue
		}

		if err := app.Save(record); err != nil {
			stats.Failed++
			continue
		}
		stats.Updated++
	}

	if stats.Failed > 0 {
		return stats, fmt.Errorf("backfill completed with %d failed records", stats.Failed)
	}

	return stats, nil
}

func hashReaderSHA256Hex(reader io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
