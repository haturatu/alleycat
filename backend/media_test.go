package main

import (
	"encoding/json"
	"os"
	"testing"
)

type mediaCase struct {
	Name       string `json:"name"`
	Filename   string `json:"filename"`
	Checksum   string `json:"checksum"`
	Normalized string `json:"normalized"`
	UploadPath string `json:"upload_path"`
}

func loadMediaCases(t *testing.T) []mediaCase {
	t.Helper()

	data, err := os.ReadFile("testdata/media_cases.json")
	if err != nil {
		t.Fatalf("read media cases: %v", err)
	}
	var cases []mediaCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("unmarshal media cases: %v", err)
	}
	return cases
}

func TestNormalizeUploadFilename(t *testing.T) {
	t.Parallel()

	for _, tc := range loadMediaCases(t) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeUploadFilename(tc.Filename); got != tc.Normalized {
				t.Fatalf("normalizeUploadFilename(%q) = %q, want %q", tc.Filename, got, tc.Normalized)
			}
		})
	}

	t.Run("short suffix kept", func(t *testing.T) {
		t.Parallel()
		if got := normalizeUploadFilename("photo_ab.jpg"); got != "photo_ab.jpg" {
			t.Fatalf("normalizeUploadFilename returned %q", got)
		}
	})
}

func TestBuildUploadPath(t *testing.T) {
	t.Parallel()

	for _, tc := range loadMediaCases(t) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if got := buildUploadPath(tc.Filename, tc.Checksum); got != tc.UploadPath {
				t.Fatalf("buildUploadPath(%q, %q) = %q, want %q", tc.Filename, tc.Checksum, got, tc.UploadPath)
			}
		})
	}

	t.Run("checksum path without extension", func(t *testing.T) {
		t.Parallel()
		if got := buildUploadPath("photo", "abc123"); got != "/uploads/abc123" {
			t.Fatalf("buildUploadPath returned %q", got)
		}
	})
}
