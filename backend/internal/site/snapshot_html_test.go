package site

import (
	"bytes"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSnapshotResolvesMetadataDuringGeneration(t *testing.T) {
	root := t.TempDir()
	settings := defaultSettings()
	settings.SiteURL = "https://configured.example/base?ignored=1"
	input := []byte(`<html><head><link rel="canonical" href="/posts/example/">` +
		`<meta property="og:image" content="/og/example.png"></head><body>Content</body></html>`)
	if err := withSnapshotBuildContext(&snapshotBuildContext{settings: settings}, func() error {
		return writeSnapshotRoute(root, "/posts/example/", input)
	}); err != nil {
		t.Fatal(err)
	}
	target, err := snapshotFilePath(root, "/posts/example/")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(saved, []byte(absoluteSnapshotHTMLPrefix)) {
		t.Fatal("generated HTML does not record resolved metadata")
	}
	for _, want := range []string{"https://configured.example/posts/example", "https://configured.example/og/example.png"} {
		if !bytes.Contains(saved, []byte(want)) {
			t.Fatalf("generated metadata is missing %q", want)
		}
	}
	previous := getPrerenderedSnapshotDir()
	// Keep any previously selected test snapshot intact.
	prerenderedSnapshot.mu.Lock()
	prerenderedSnapshot.dir = root
	prerenderedSnapshot.mu.Unlock()
	t.Cleanup(func() {
		prerenderedSnapshot.mu.Lock()
		prerenderedSnapshot.dir = previous
		prerenderedSnapshot.mu.Unlock()
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://different-host.example/posts/example/", nil)
	if !servePrerenderedSnapshot(recorder, req, "/posts/example/") {
		t.Fatal("generated snapshot was not served")
	}
	if !bytes.Equal(recorder.Body.Bytes(), saved[len(absoluteSnapshotHTMLPrefix):]) {
		t.Fatal("serving changed the prepared HTML or leaked its internal prefix")
	}
}

func TestSnapshotWithoutSiteURLKeepsRequestHostFallback(t *testing.T) {
	body := []byte(`<link rel="canonical" href="/example/">`)
	if got := preparePrerenderedSnapshotHTML(body, SettingsRecord{}); !bytes.Equal(got, body) {
		t.Fatal("snapshot without a configured base URL must remain unprepared")
	}
}

func TestPreparedSnapshotDoesNotAllocateDOMPerRequest(t *testing.T) {
	root := t.TempDir()
	settings := SettingsRecord{SiteURL: "https://configured.example"}
	body := []byte(`<html><head><link rel="canonical" href="/example/"></head><body>` +
		strings.Repeat(`<p>Text with <a href="/archive">a link</a>.</p>`, 500) + `</body></html>`)
	if err := withSnapshotBuildContext(&snapshotBuildContext{settings: settings}, func() error {
		return writeSnapshotRoute(root, "/example/", body)
	}); err != nil {
		t.Fatal(err)
	}
	prerenderedSnapshot.mu.Lock()
	previous := prerenderedSnapshot.dir
	prerenderedSnapshot.dir = root
	prerenderedSnapshot.mu.Unlock()
	defer func() {
		prerenderedSnapshot.mu.Lock()
		prerenderedSnapshot.dir = previous
		prerenderedSnapshot.mu.Unlock()
	}()
	req := httptest.NewRequest("GET", "https://configured.example/example/", nil)
	served := true
	allocations := testing.AllocsPerRun(5, func() {
		served = servePrerenderedSnapshot(httptest.NewRecorder(), req, "/example/") && served
	})
	if !served {
		t.Fatal("snapshot not served")
	}
	// DOM construction for this fixture requires over 9,000 allocations. Leave
	// ample room for changes in HTTP and file I/O without allowing it to return.
	if allocations >= 500 {
		t.Fatalf("prepared snapshot allocated %.0f objects per request; expected fewer than 500", allocations)
	}
}
