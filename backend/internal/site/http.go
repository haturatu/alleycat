package site

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

const (
	maxUpstreamJSONBytes  = 8 * 1024 * 1024
	maxUpstreamErrorBytes = 64 * 1024
)

func fetchJSON[T any](target string) (T, error) {
	return fetchJSONContext[T](context.Background(), target)
}

func fetchJSONContext[T any](ctx context.Context, target string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return zero, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamErrorBytes+1))
		if len(body) > maxUpstreamErrorBytes {
			body = append(body[:maxUpstreamErrorBytes], []byte("…(truncated)")...)
		}
		return zero, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamJSONBytes+1))
	if err != nil {
		return zero, err
	}
	if len(body) > maxUpstreamJSONBytes {
		return zero, fmt.Errorf("upstream response exceeds %d bytes", maxUpstreamJSONBytes)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func fetchList[T any](base string, params map[string]string) (PBList[T], error) {
	return fetchListContext[T](context.Background(), base, params)
}

func fetchListContext[T any](ctx context.Context, base string, params map[string]string) (PBList[T], error) {
	u, err := url.Parse(base)
	if err != nil {
		return PBList[T]{}, err
	}
	q := u.Query()
	for key, value := range params {
		if value == "" {
			continue
		}
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return fetchJSONContext[PBList[T]](ctx, u.String())
}

func fetchRecord[T any](target string) (T, error) {
	return fetchJSON[T](target)
}
