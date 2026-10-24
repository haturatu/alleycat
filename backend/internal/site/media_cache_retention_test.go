package site

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMediaPathCacheBoundsMissesAndReclaimsExpiry(t *testing.T) {
	mediaPathCache.mu.Lock()
	original := mediaPathCache.items
	mediaPathCache.items = map[string]mediaPathCacheEntry{}
	mediaPathCache.mu.Unlock()
	t.Cleanup(func() { mediaPathCache.mu.Lock(); mediaPathCache.items = original; mediaPathCache.mu.Unlock() })
	now := time.Now()
	for i := 0; i < mediaPathCacheMaxEntries*3; i++ {
		cacheMediaPath(fmt.Sprint(i), mediaPathCacheEntry{expiresAt: now.Add(time.Minute)}, now)
	}
	if len(mediaPathCache.items) != mediaPathCacheMaxEntries {
		t.Fatal(len(mediaPathCache.items))
	}
	cacheMediaPath("new", mediaPathCacheEntry{expiresAt: now.Add(time.Hour), found: true}, now.Add(2*time.Minute))
	if len(mediaPathCache.items) != 1 || !mediaPathCache.items["new"].found {
		t.Fatal("expired misses survived capacity sweep")
	}
	cacheMediaPath(strings.Repeat("x", mediaPathCacheMaxKeyBytes+1), mediaPathCacheEntry{}, now)
	if len(mediaPathCache.items) != 1 {
		t.Fatal("oversized key was cached")
	}
}
