package site

import "testing"

func TestPostBySlugNodeKey(t *testing.T) {
	t.Parallel()

	base := postBySlugNodeKey("", "hello")
	if base.Kind != nodePostBySlug || base.Scope != nodeScopeBase || base.ID != "hello" {
		t.Fatalf("base post key = %#v", base)
	}

	localized := postBySlugNodeKey("RU", "hello")
	if localized.Kind != nodeTranslationBySlug || localized.Scope != "ru" || localized.ID != "hello" {
		t.Fatalf("localized post key = %#v", localized)
	}
}

func TestRouteNodeKeyCleansPath(t *testing.T) {
	t.Parallel()

	key := routeNodeKey("/posts/hello//")
	if key.Kind != nodeRoute || key.ID != "/posts/hello" {
		t.Fatalf("route key = %#v", key)
	}
}
