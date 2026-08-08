package site

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApplyRevalidationSettingsRebuildsOutsideSnapshotContext(t *testing.T) {
	previousRoot := getPrerenderedSnapshotDir()
	prerenderedSnapshot.mu.Lock()
	prerenderedSnapshot.dir = t.TempDir()
	prerenderedSnapshot.mu.Unlock()
	t.Cleanup(func() {
		prerenderedSnapshot.mu.Lock()
		prerenderedSnapshot.dir = previousRoot
		prerenderedSnapshot.mu.Unlock()
	})

	previousRebuild := rebuildWholeSnapshotForRevalidation
	t.Cleanup(func() { rebuildWholeSnapshotForRevalidation = previousRebuild })
	rebuilt := false
	rebuildWholeSnapshotForRevalidation = func(context.Context) error {
		rebuilt = true
		if currentSnapshotBuildContext() != nil {
			t.Fatal("settings rebuild must not run inside a snapshot build context")
		}
		return nil
	}

	err := applyRevalidation(context.Background(), revalidateRequest{Collection: "settings", Action: "update"})
	if err != nil {
		t.Fatalf("applyRevalidation: %v", err)
	}
	if !rebuilt {
		t.Fatal("settings rebuild was not called")
	}
}

func TestApplyRevalidationTimesOutWhileWaitingForMutationSlot(t *testing.T) {
	<-snapshotMutation
	t.Cleanup(func() { snapshotMutation <- struct{}{} })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := applyRevalidation(ctx, revalidateRequest{Collection: "posts", Action: "update"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}

func TestHandleRevalidateRejectsOversizedBody(t *testing.T) {
	t.Setenv("STATIC_REGEN_TOKEN", "test-token")
	body := `{"collection":"posts","action":"update","current":"` + strings.Repeat("x", maxRevalidateBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/__internal/revalidate", strings.NewReader(body))
	req.Header.Set("X-Regen-Token", "test-token")
	rec := httptest.NewRecorder()

	handleRevalidate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRevalidateRejectsUnsupportedCollection(t *testing.T) {
	t.Setenv("STATIC_REGEN_TOKEN", "test-token")
	req := httptest.NewRequest(http.MethodPost, "/__internal/revalidate", strings.NewReader(`{"collection":"unknown","action":"update"}`))
	req.Header.Set("X-Regen-Token", "test-token")
	rec := httptest.NewRecorder()

	handleRevalidate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
