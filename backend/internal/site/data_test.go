package site

import (
	"testing"
	"time"
)

func TestPostPublishedTimeParsesPocketBaseTimestamp(t *testing.T) {
	t.Parallel()

	post := PostRecord{PublishedAt: "2026-04-18 09:31:00.000Z"}
	got := postPublishedTime(post)
	want := time.Date(2026, 4, 18, 9, 31, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("postPublishedTime() = %v, want %v", got, want)
	}
}
