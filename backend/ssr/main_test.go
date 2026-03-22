package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLocalizedPostPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "/ja/posts/hello/", want: true},
		{path: "/en-us/posts/hello/", want: true},
		{path: "/posts/hello/", want: false},
		{path: "/ja/archive/hello/", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := isLocalizedPostPath(tt.path); got != tt.want {
				t.Fatalf("isLocalizedPostPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRouteHandlerRejectsAdminPath(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	rec := httptest.NewRecorder()

	routeHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouteHandlerRevalidateRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/__internal/revalidate", nil)
	rec := httptest.NewRecorder()

	routeHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
