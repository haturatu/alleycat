package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

func fetchJSON[T any](target string) (T, error) {
	var zero T
	resp, err := httpClient.Get(target)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	var out T
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return zero, err
	}
	return out, nil
}

func fetchList[T any](base string, params map[string]string) (PBList[T], error) {
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
	return fetchJSON[PBList[T]](u.String())
}

func fetchRecord[T any](target string) (T, error) {
	return fetchJSON[T](target)
}

func proxyFile(url string) (*http.Response, []byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	copyResp := *resp
	copyResp.Body = io.NopCloser(bytes.NewReader(body))
	return &copyResp, body, nil
}
