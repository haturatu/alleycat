package pbapp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
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

const staticRegenQueueCapacity = 32

type staticRegenJob struct {
	target  string
	token   string
	payload regenRequest
}

var staticRegenQueue = struct {
	once  sync.Once
	queue chan staticRegenJob
}{}

var staticRegenHTTPClient = &http.Client{Timeout: 20 * time.Second}

func registerStaticRegenHooks(app *pocketbase.PocketBase) {
	startStaticRegenWorker()
	bindRegenHooks(app, "posts")
	bindRegenHooks(app, "pages")
	bindRegenHooks(app, "post_translations")
	bindRegenHooks(app, "settings")
}

func startStaticRegenWorker() {
	staticRegenQueue.once.Do(func() {
		staticRegenQueue.queue = make(chan staticRegenJob, staticRegenQueueCapacity)
		go func() {
			for job := range staticRegenQueue.queue {
				notifyStaticRegen(job)
			}
		}()
	})
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
	startStaticRegenWorker()
	job := staticRegenJob{
		target:  target,
		token:   strings.TrimSpace(os.Getenv("STATIC_REGEN_TOKEN")),
		payload: payload,
	}
	select {
	case staticRegenQueue.queue <- job:
		slog.Debug("static regen queued", "collection", collection, "action", action, "queue_capacity", staticRegenQueueCapacity)
	default:
		slog.Warn("static regen dropped because queue is full", "collection", collection, "action", action, "queue_capacity", staticRegenQueueCapacity)
	}
}

func notifyStaticRegen(job staticRegenJob) {
	body, err := json.Marshal(job.payload)
	if err != nil {
		slog.Error("static regen payload marshal failed", "collection", job.payload.Collection, "action", job.payload.Action, "error", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, job.target, bytes.NewReader(body))
	if err != nil {
		slog.Error("static regen request init failed", "collection", job.payload.Collection, "action", job.payload.Action, "target", job.target, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if job.token != "" {
		req.Header.Set("X-Regen-Token", job.token)
	}

	slog.Info("static regen notify start", "collection", job.payload.Collection, "action", job.payload.Action, "target", job.target)
	resp, err := staticRegenHTTPClient.Do(req)
	if err != nil {
		slog.Error("static regen notify failed", "collection", job.payload.Collection, "action", job.payload.Action, "target", job.target, "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("static regen notify returned non-success status", "collection", job.payload.Collection, "action", job.payload.Action, "status", resp.StatusCode, "target", job.target)
		return
	}
	slog.Info("static regen notify completed", "collection", job.payload.Collection, "action", job.payload.Action, "status", resp.StatusCode, "target", job.target)
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
