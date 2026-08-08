package site

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchJSONLimitsUpstreamBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"value":"%s"}`, strings.Repeat("x", maxUpstreamJSONBytes))
	}))
	defer server.Close()

	_, err := fetchJSON[struct {
		Value string `json:"value"`
	}](server.URL)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("fetchJSON error = %v, want bounded response error", err)
	}
}

func TestFetchJSONTruncatesUpstreamErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", maxUpstreamErrorBytes+1024)))
	}))
	defer server.Close()

	_, err := fetchJSON[struct{}](server.URL)
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("fetchJSON error = %v, want truncated upstream error", err)
	}
}
