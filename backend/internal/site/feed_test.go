package site

import (
	"reflect"
	"testing"
	"time"
)

func TestFeedItemTimeParsesPocketBaseTimestamp(t *testing.T) {
	t.Parallel()

	got := feedItemTime(feedItem{Date: "2026-03-31 03:23:00.000Z"})
	want := time.Date(2026, 3, 31, 3, 23, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("feedItemTime() = %v, want %v", got, want)
	}
}

func TestFeedItemsSortNewestFirst(t *testing.T) {
	t.Parallel()

	items := []feedItem{
		{URL: "https://example.com/posts/older/", Date: "2026-03-22 23:50:00.000Z"},
		{URL: "https://example.com/posts/newer/", Date: "2026-03-31 03:23:00.000Z"},
		{URL: "https://example.com/posts/oldest/", Date: "2025-09-13 15:00:00.000Z"},
	}
	sortFeedItems(items)
	got := []string{items[0].URL, items[1].URL, items[2].URL}
	want := []string{
		"https://example.com/posts/newer/",
		"https://example.com/posts/older/",
		"https://example.com/posts/oldest/",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortFeedItems() order = %#v, want %#v", got, want)
	}
}
