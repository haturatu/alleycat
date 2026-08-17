package pbapp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

type localeCase struct {
	Name       string   `json:"name"`
	Input      string   `json:"input"`
	Locales    []string `json:"locales"`
	Normalized string   `json:"normalized"`
}

func loadLocaleCases(t *testing.T) []localeCase {
	t.Helper()

	data, err := os.ReadFile("testdata/locale_cases.json")
	if err != nil {
		t.Fatalf("read locale cases: %v", err)
	}
	var cases []localeCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("unmarshal locale cases: %v", err)
	}
	return cases
}

func TestParseLocaleList(t *testing.T) {
	t.Parallel()

	for _, tc := range loadLocaleCases(t) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			got := parseLocaleList(tc.Input)
			if !reflect.DeepEqual(got, tc.Locales) {
				t.Fatalf("parseLocaleList(%q) = %#v, want %#v", tc.Input, got, tc.Locales)
			}
		})
	}
}

func TestNormalizeLocale(t *testing.T) {
	t.Parallel()

	cases := loadLocaleCases(t)
	if got := normalizeLocale(" En_US "); got != cases[0].Normalized {
		t.Fatalf("normalizeLocale returned %q, want %q", got, cases[0].Normalized)
	}
	if got := normalizeLocale(" ja_JP "); got != "ja-jp" {
		t.Fatalf("normalizeLocale returned %q, want %q", got, "ja-jp")
	}
}

func TestSplitTranslationBodyKeepsSmallBodyWhole(t *testing.T) {
	t.Parallel()

	body := "<p>Hello</p>\n<p>World</p>"
	got := splitTranslationBody(body, 100)
	want := []string{body}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitTranslationBody() = %#v, want %#v", got, want)
	}
}

func TestSplitTranslationBodySplitsOnBlockBoundaries(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		"<p>first paragraph with enough text</p>",
		"<p>second paragraph with enough text</p>",
		"<p>third paragraph with enough text</p>",
	}, "")

	got := splitTranslationBody(body, 50)
	if len(got) != 3 {
		t.Fatalf("chunk count = %d, want 3; chunks=%#v", len(got), got)
	}
	for _, chunk := range got {
		if len([]rune(chunk)) > 50 {
			t.Fatalf("chunk too large: %d runes in %q", len([]rune(chunk)), chunk)
		}
		if !strings.HasSuffix(chunk, "</p>") {
			t.Fatalf("expected paragraph boundary chunk, got %q", chunk)
		}
	}
}

func TestSplitTranslationBodySplitsOversizedSegment(t *testing.T) {
	t.Parallel()

	body := "<p>" + strings.Repeat("a", 120) + "</p>"
	got := splitTranslationBody(body, 40)
	if len(got) < 3 {
		t.Fatalf("expected oversized segment to split, got %#v", got)
	}
	for _, chunk := range got {
		if len([]rune(chunk)) > 40 {
			t.Fatalf("chunk too large: %d runes in %q", len([]rune(chunk)), chunk)
		}
	}
}

func TestExtractFirstJSONObject(t *testing.T) {
	t.Parallel()

	input := "```json\n{\"translated_body\":\"hello\"}\n```\nextra text"
	got, err := extractFirstJSONObject(input)
	if err != nil {
		t.Fatalf("extractFirstJSONObject returned error: %v", err)
	}
	if got != "{\"translated_body\":\"hello\"}" {
		t.Fatalf("extractFirstJSONObject = %q", got)
	}
}

func TestExtractFirstJSONObjectKeepsEscapedQuotes(t *testing.T) {
	t.Parallel()

	input := "{\"translated_body\":\"say \\\"hello\\\"\"} trailing"
	got, err := extractFirstJSONObject(input)
	if err != nil {
		t.Fatalf("extractFirstJSONObject returned error: %v", err)
	}
	if got != "{\"translated_body\":\"say \\\"hello\\\"\"}" {
		t.Fatalf("extractFirstJSONObject = %q", got)
	}
}

func TestParseGeminiTranslationTextRepairsInvalidStringEscapes(t *testing.T) {
	t.Parallel()

	input := `{"translated_title":"title","translated_body":"Use \uÐ... and C:\Users\docs"}`
	got, err := parseGeminiTranslationText(input)
	if err != nil {
		t.Fatalf("parseGeminiTranslationText returned error: %v", err)
	}
	wantBody := `Use \uÐ... and C:\Users\docs`
	if got.TranslatedBody != wantBody {
		t.Fatalf("translated body = %q, want %q", got.TranslatedBody, wantBody)
	}
}

func TestParseGeminiTranslationTextKeepsValidEscapes(t *testing.T) {
	t.Parallel()

	input := `{"translated_title":"line\n\\path","translated_body":"こんにちは\\u4f60"}`
	got, err := parseGeminiTranslationText(input)
	if err != nil {
		t.Fatalf("parseGeminiTranslationText returned error: %v", err)
	}
	if got.TranslatedTitle != "line\n\\path" {
		t.Fatalf("translated title = %q, want valid JSON escapes to be decoded", got.TranslatedTitle)
	}
	if got.TranslatedBody != "こんにちは\\u4f60" {
		t.Fatalf("translated body = %q, want literal escaped unicode sequence", got.TranslatedBody)
	}
}

func TestTranslationResponseSchema(t *testing.T) {
	t.Parallel()

	got := geminiResponseSchema("translated_title", "translated_body")
	want := map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"translated_title": map[string]any{"type": "STRING"},
			"translated_body":  map[string]any{"type": "STRING"},
		},
		"required": []string{"translated_title", "translated_body"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("geminiResponseSchema = %#v, want %#v", got, want)
	}
}

func TestNewTranslationProviderSupportsConfiguredProviders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		provider string
		wantType any
	}{
		{provider: "gemini", wantType: &geminiTranslationProvider{}},
		{provider: "opencode-go", wantType: &openCodeGoTranslationProvider{}},
		{provider: "opencode-zen", wantType: &openCodeZenTranslationProvider{}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.provider, func(t *testing.T) {
			provider, err := newTranslationProvider(translationSettings{Provider: tc.provider})
			if err != nil {
				t.Fatalf("newTranslationProvider returned error: %v", err)
			}
			switch tc.wantType.(type) {
			case *geminiTranslationProvider:
				if _, ok := provider.(*geminiTranslationProvider); !ok {
					t.Fatalf("provider = %T, want Gemini provider", provider)
				}
			case *openCodeGoTranslationProvider:
				if _, ok := provider.(*openCodeGoTranslationProvider); !ok {
					t.Fatalf("provider = %T, want OpenCode Go provider", provider)
				}
			case *openCodeZenTranslationProvider:
				if _, ok := provider.(*openCodeZenTranslationProvider); !ok {
					t.Fatalf("provider = %T, want OpenCode Zen provider", provider)
				}
			}
		})
	}

	if _, err := newTranslationProvider(translationSettings{Provider: "unsupported"}); err == nil {
		t.Fatal("newTranslationProvider accepted an unsupported provider")
	}
}

func TestRequestOpenCodeGoJSON(t *testing.T) {
	originalURL := openCodeGoChatCompletionsURL
	t.Cleanup(func() { openCodeGoChatCompletionsURL = originalURL })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["model"] != "kimi-k2.6" {
			t.Errorf("model = %v, want kimi-k2.6", payload["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"translated_title\\\":\\\"Hello\\\",\\\"translated_body\\\":\\\"<p>World</p>\\\"}\\n```\"}}]}"))
	}))
	defer server.Close()
	openCodeGoChatCompletionsURL = server.URL

	text, err := requestOpenCodeGoJSON("translate", "kimi-k2.6", "test-key", 0)
	if err != nil {
		t.Fatalf("requestOpenCodeGoJSON returned error: %v", err)
	}
	if text != `{"translated_title":"Hello","translated_body":"<p>World</p>"}` {
		t.Fatalf("response = %q", text)
	}
}

func TestOpenCodeZenProtocolForModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model    string
		protocol openCodeZenProtocol
	}{
		{model: "gpt-5.4-mini", protocol: openCodeZenResponsesProtocol},
		{model: "claude-sonnet-4-6", protocol: openCodeZenMessagesProtocol},
		{model: "gemini-3.5-flash", protocol: openCodeZenGoogleProtocol},
		{model: "deepseek-v4-flash-free", protocol: openCodeZenChatProtocol},
		{model: "opencode/kimi-k2.6", protocol: openCodeZenChatProtocol},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.model, func(t *testing.T) {
			t.Parallel()
			if got := openCodeZenProtocolForModel(tc.model); got != tc.protocol {
				t.Fatalf("openCodeZenProtocolForModel(%q) = %q, want %q", tc.model, got, tc.protocol)
			}
		})
	}
}

func TestRequestOpenCodeZenChatCompletions(t *testing.T) {
	originalURL := openCodeZenChatCompletionsURL
	t.Cleanup(func() { openCodeZenChatCompletionsURL = originalURL })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["model"] != "deepseek-v4-flash-free" {
			t.Errorf("model = %v, want deepseek-v4-flash-free", payload["model"])
		}
		thinking, ok := payload["thinking"].(map[string]any)
		if !ok || thinking["type"] != "disabled" {
			t.Errorf("thinking = %#v, want disabled", payload["thinking"])
		}
		if payload["max_tokens"] != float64(maxOpenCodeOutputTokens) {
			t.Errorf("max_tokens = %v, want %d", payload["max_tokens"], maxOpenCodeOutputTokens)
		}
		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) != 1 {
			t.Errorf("messages = %#v, want one message", payload["messages"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"{\\\"slug\\\":\\\"hello-world\\\"}\"}}]}"))
	}))
	defer server.Close()
	openCodeZenChatCompletionsURL = server.URL

	text, err := requestOpenCodeZen("translate", "deepseek-v4-flash-free", "test-key", 0)
	if err != nil {
		t.Fatalf("requestOpenCodeZen returned error: %v", err)
	}
	if text != `{"slug":"hello-world"}` {
		t.Fatalf("response = %q", text)
	}
}

func TestParseOpenCodeZenResponseText(t *testing.T) {
	t.Parallel()

	response := []byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"slug\":\"hello-world\"}"}]}]}`)
	got, err := parseOpenCodeZenResponseText(response)
	if err != nil {
		t.Fatalf("parseOpenCodeZenResponseText returned error: %v", err)
	}
	if got != `{"slug":"hello-world"}` {
		t.Fatalf("response = %q", got)
	}
}

func TestParseTranslationModels(t *testing.T) {
	t.Parallel()

	gemini, err := parseGeminiModels([]byte(`{"models":[{"name":"models/gemini-2.5-flash","supportedGenerationMethods":["generateContent"]},{"name":"models/embedding-001","supportedGenerationMethods":["embedContent"]}]}`))
	if err != nil {
		t.Fatalf("parseGeminiModels returned error: %v", err)
	}
	if !reflect.DeepEqual(gemini, []string{"gemini-2.5-flash"}) {
		t.Fatalf("Gemini models = %#v", gemini)
	}

	openCode, err := parseOpenCodeModels([]byte(`{"data":[{"id":"kimi-k2.6"},{"id":"glm-5.2"},{"id":"kimi-k2.6"}]}`))
	if err != nil {
		t.Fatalf("parseOpenCodeModels returned error: %v", err)
	}
	if !reflect.DeepEqual(openCode, []string{"glm-5.2", "kimi-k2.6"}) {
		t.Fatalf("OpenCode models = %#v", openCode)
	}
}
