package main

func invalidateSettingsCache() {
	settingsCache.mu.Lock()
	settingsCache.entry = settingsCacheEntry{}
	settingsCache.mu.Unlock()
}

func invalidateTaxonomyCache() {
	taxonomyCache.mu.Lock()
	taxonomyCache.entry = taxonomyCacheEntry{}
	taxonomyCache.mu.Unlock()
}

func invalidateFeedCache() {
	feedItemsCache.mu.Lock()
	feedItemsCache.items = map[string]feedCacheEntry{}
	feedItemsCache.mu.Unlock()
}

func invalidateSitemapCache() {
	sitemapCache.mu.Lock()
	sitemapCache.items = map[string]sitemapCacheEntry{}
	sitemapCache.mu.Unlock()
}

func invalidateDerivedCaches() {
	invalidateSettingsCache()
	invalidateTaxonomyCache()
	invalidateFeedCache()
	invalidateSitemapCache()
}
