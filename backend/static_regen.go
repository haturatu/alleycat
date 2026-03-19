package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type regenRequest struct {
	Collection string          `json:"collection"`
	Action     string          `json:"action"`
	Current    json.RawMessage `json:"current"`
	Original   json.RawMessage `json:"original"`
}

func registerStaticRegenHooks(app *pocketbase.PocketBase) {
	bindRegenHooks(app, "posts")
	bindRegenHooks(app, "pages")
	bindRegenHooks(app, "post_translations")
	bindRegenHooks(app, "settings")
}

func bindRegenHooks(app *pocketbase.PocketBase, collection string) {
	app.OnRecordAfterCreateSuccess(collection).BindFunc(func(e *core.RecordEvent) error {
		triggerStaticRegen(collection, "create", e.Record, nil)
		return e.Next()
	})
	app.OnRecordAfterUpdateSuccess(collection).BindFunc(func(e *core.RecordEvent) error {
		triggerStaticRegen(collection, "update", e.Record, e.Record.Original())
		return e.Next()
	})
	app.OnRecordAfterDeleteSuccess(collection).BindFunc(func(e *core.RecordEvent) error {
		triggerStaticRegen(collection, "delete", nil, e.Record.Original())
		return e.Next()
	})
}

func triggerStaticRegen(collection, action string, current, original *core.Record) {
	target := strings.TrimSpace(os.Getenv("SSR_REGEN_URL"))
	if target == "" {
		slog.Debug("static regen skipped because SSR_REGEN_URL is empty", "collection", collection, "action", action)
		return
	}

	payload := regenRequest{
		Collection: collection,
		Action:     action,
		Current:    marshalRecordJSON(current),
		Original:   marshalRecordJSON(original),
	}

	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			slog.Error("static regen payload marshal failed", "collection", collection, "action", action, "error", err)
			return
		}

		req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
		if err != nil {
			slog.Error("static regen request init failed", "collection", collection, "action", action, "target", target, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if token := strings.TrimSpace(os.Getenv("STATIC_REGEN_TOKEN")); token != "" {
			req.Header.Set("X-Regen-Token", token)
		}

		client := &http.Client{Timeout: 5 * time.Second}
		slog.Info("static regen notify start", "collection", collection, "action", action, "target", target)
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("static regen notify failed", "collection", collection, "action", action, "target", target, "error", err)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			slog.Warn("static regen notify returned non-success status", "collection", collection, "action", action, "status", resp.StatusCode, "target", target)
			return
		}
		slog.Info("static regen notify completed", "collection", collection, "action", action, "status", resp.StatusCode, "target", target)
	}()
}

func marshalRecordJSON(record *core.Record) json.RawMessage {
	if record == nil {
		return nil
	}
	data, err := record.MarshalJSON()
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}
