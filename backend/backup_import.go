package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/archive"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/spf13/cobra"
)

// Keep this aligned with PocketBase restore exclusions.
var backupImportExcludeEntries = []string{
	core.LocalBackupsDirName,
	core.LocalTempDirName,
	core.LocalAutocertCacheDirName,
	"lost+found",
}

func registerBackupImportCommand(app *pocketbase.PocketBase) {
	cmd := &cobra.Command{
		Use:   "import-backup <backup-zip>",
		Short: "Import a PocketBase backup zip file into the current data directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			zipPath := args[0]
			fmt.Fprintf(command.OutOrStdout(), "Importing backup zip: %s\n", zipPath)
			fmt.Fprintf(command.OutOrStdout(), "Target data dir: %s\n", app.DataDir())
			if err := importBackupZip(app, zipPath); err != nil {
				return err
			}
			fmt.Fprintln(command.OutOrStdout(), "Backup import completed.")
			return nil
		},
	}
	cmd.Long = "Import a PocketBase backup zip into pb_data.\n" +
		"Run this while the PocketBase server process is stopped."

	app.RootCmd.AddCommand(cmd)
}

func importBackupZip(app core.App, zipPath string) error {
	cleanZipPath := filepath.Clean(zipPath)
	zipInfo, err := os.Stat(cleanZipPath)
	if err != nil {
		return fmt.Errorf("failed to access backup zip %q: %w", cleanZipPath, err)
	}
	if zipInfo.IsDir() {
		return fmt.Errorf("backup path must be a file, got directory: %q", cleanZipPath)
	}
	if filepath.Ext(zipInfo.Name()) != ".zip" {
		return fmt.Errorf("backup file must be a .zip: %q", cleanZipPath)
	}

	dataDir := app.DataDir()
	if dataDir == "" {
		return errors.New("data dir is empty")
	}
	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to ensure data dir: %w", err)
	}

	localTempDir := filepath.Join(dataDir, core.LocalTempDirName)
	if err := os.MkdirAll(localTempDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	extractedDir := filepath.Join(localTempDir, "pb_import_"+security.PseudorandomString(8))
	oldTempDataDir := filepath.Join(localTempDir, "old_pb_data_"+security.PseudorandomString(8))
	defer os.RemoveAll(extractedDir)

	if err := archive.Extract(cleanZipPath, extractedDir); err != nil {
		return fmt.Errorf("failed to extract backup zip: %w", err)
	}

	extractedDB := filepath.Join(extractedDir, "data.db")
	if _, err := os.Stat(extractedDB); err != nil {
		return fmt.Errorf("invalid backup zip: data.db is missing (%w)", err)
	}

	if err := osutils.MoveDirContent(dataDir, oldTempDataDir, backupImportExcludeEntries...); err != nil {
		return fmt.Errorf("failed to move current pb_data to temp dir: %w", err)
	}

	restoreErr := osutils.MoveDirContent(extractedDir, dataDir, backupImportExcludeEntries...)
	if restoreErr != nil {
		_ = osutils.MoveDirContent(dataDir, extractedDir, backupImportExcludeEntries...)
		_ = osutils.MoveDirContent(oldTempDataDir, dataDir, backupImportExcludeEntries...)
		return fmt.Errorf("failed to move extracted backup into pb_data: %w", restoreErr)
	}

	if err := os.RemoveAll(oldTempDataDir); err != nil {
		return fmt.Errorf("backup imported, but cleanup failed for %q: %w", oldTempDataDir, err)
	}

	return nil
}
