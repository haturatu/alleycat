package site

import (
	"bytes"
	"testing"
)

func FuzzDecodeRevalidateRequest(f *testing.F) {
	f.Add([]byte(`{"collection":"posts","action":"update"}`))
	f.Add([]byte(`{"collection":"settings","action":"update","current":null,"original":null}`))
	f.Add([]byte("not-json"))

	f.Fuzz(func(t *testing.T, body []byte) {
		request, err := decodeRevalidateRequest(bytes.NewReader(body))
		if err == nil && len(request.Collection) > len(body) {
			t.Fatalf("decoded collection length %d exceeds body length %d", len(request.Collection), len(body))
		}
	})
}
