package main

import (
	"log/slog"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})))

	app := pocketbase.New()
	registerTranslationFeatures(app)
	registerBackupImportCommand(app)
	registerMediaChecksumBackfillCommand(app)
	registerStaticRegenHooks(app)

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return ensureCollections(e.App)
	})

	if err := app.Start(); err != nil {
		slog.Error("pocketbase start failed", "error", err)
		os.Exit(1)
	}
}
