package site

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestRevalidateHTTPChaos(t *testing.T) {
	t.Setenv("STATIC_REGEN_TOKEN", "chaos-token")

	const requests = 1000
	var wg sync.WaitGroup
	wg.Add(requests)
	errors := make(chan string, requests)
	for index := 0; index < requests; index++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/__internal/revalidate", strings.NewReader(`{"collection":"unknown","action":"update"}`))
			req.Header.Set("X-Regen-Token", "chaos-token")
			rec := httptest.NewRecorder()
			handleRevalidate(rec, req)
			if rec.Code != http.StatusBadRequest {
				errors <- strconv.Itoa(rec.Code)
			}
		}()
	}
	wg.Wait()
	close(errors)
	for status := range errors {
		t.Errorf("unexpected status under concurrent invalid requests: %s", status)
	}
}
