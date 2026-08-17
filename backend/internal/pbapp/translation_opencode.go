package pbapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultOpenCodeGoChatCompletionsURL  = "https://opencode.ai/zen/go/v1/chat/completions"
	defaultOpenCodeGoModelsURL           = "https://opencode.ai/zen/go/v1/models"
	defaultOpenCodeZenResponsesURL       = "https://opencode.ai/zen/v1/responses"
	defaultOpenCodeZenChatCompletionsURL = "https://opencode.ai/zen/v1/chat/completions"
	defaultOpenCodeZenMessagesURL        = "https://opencode.ai/zen/v1/messages"
	defaultOpenCodeZenGoogleModelsURL    = "https://opencode.ai/zen/v1/models"
	defaultOpenCodeZenModelsURL          = "https://opencode.ai/zen/v1/models"
)

var (
	openCodeGoChatCompletionsURL  = defaultOpenCodeGoChatCompletionsURL
	openCodeGoModelsURL           = defaultOpenCodeGoModelsURL
	openCodeZenResponsesURL       = defaultOpenCodeZenResponsesURL
	openCodeZenChatCompletionsURL = defaultOpenCodeZenChatCompletionsURL
	openCodeZenMessagesURL        = defaultOpenCodeZenMessagesURL
	openCodeZenGoogleModelsURL    = defaultOpenCodeZenGoogleModelsURL
	openCodeZenModelsURL          = defaultOpenCodeZenModelsURL
)

type openCodeGoTranslationProvider struct {
	model      string
	apiKey     string
	requestsPM int
}

type openCodeZenTranslationProvider struct {
	model      string
	apiKey     string
	requestsPM int
}

type openCodeGoChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type openCodeZenResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

type openCodeZenMessagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type openCodeZenGoogleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (p *openCodeGoTranslationProvider) translateTitleAndBody(title, body, sourceLocale, targetLocale string) (string, string, error) {
	text, err := requestOpenCodeGoJSON(
		buildTranslationTitleAndBodyPrompt(title, body, sourceLocale, targetLocale),
		p.model,
		p.apiKey,
		p.requestsPM,
	)
	if err != nil {
		return "", "", err
	}
	return parseTranslatedPayload(text, "opencode-go")
}

func (p *openCodeGoTranslationProvider) translateTitle(title, sourceLocale, targetLocale string) (string, error) {
	text, err := requestOpenCodeGoJSON(
		buildTranslationTitlePrompt(title, sourceLocale, targetLocale),
		p.model,
		p.apiKey,
		p.requestsPM,
	)
	if err != nil {
		return "", err
	}
	return parseTranslatedTitle(text, "opencode-go")
}

func (p *openCodeGoTranslationProvider) translateBodyChunk(body, sourceLocale, targetLocale string, chunkIndex, chunkCount int) (string, error) {
	text, err := requestOpenCodeGoJSON(
		buildTranslationBodyChunkPrompt(body, sourceLocale, targetLocale, chunkIndex, chunkCount),
		p.model,
		p.apiKey,
		p.requestsPM,
	)
	if err != nil {
		return "", err
	}
	return parseTranslatedBody(text, "opencode-go")
}

func (p *openCodeGoTranslationProvider) generateSlug(title string) (string, error) {
	return generateSlugWithOpenCodeRequest(
		func(prompt string) (string, error) {
			return requestOpenCodeGoJSON(prompt, p.model, p.apiKey, p.requestsPM)
		},
		title,
		"opencode-go",
	)
}

func (p *openCodeZenTranslationProvider) translateTitleAndBody(title, body, sourceLocale, targetLocale string) (string, string, error) {
	text, err := requestOpenCodeZen(
		buildTranslationTitleAndBodyPrompt(title, body, sourceLocale, targetLocale),
		p.model,
		p.apiKey,
		p.requestsPM,
	)
	if err != nil {
		return "", "", err
	}
	return parseTranslatedPayload(text, "opencode-zen")
}

func (p *openCodeZenTranslationProvider) translateTitle(title, sourceLocale, targetLocale string) (string, error) {
	text, err := requestOpenCodeZen(
		buildTranslationTitlePrompt(title, sourceLocale, targetLocale),
		p.model,
		p.apiKey,
		p.requestsPM,
	)
	if err != nil {
		return "", err
	}
	return parseTranslatedTitle(text, "opencode-zen")
}

func (p *openCodeZenTranslationProvider) translateBodyChunk(body, sourceLocale, targetLocale string, chunkIndex, chunkCount int) (string, error) {
	text, err := requestOpenCodeZen(
		buildTranslationBodyChunkPrompt(body, sourceLocale, targetLocale, chunkIndex, chunkCount),
		p.model,
		p.apiKey,
		p.requestsPM,
	)
	if err != nil {
		return "", err
	}
	return parseTranslatedBody(text, "opencode-zen")
}

func (p *openCodeZenTranslationProvider) generateSlug(title string) (string, error) {
	return generateSlugWithOpenCodeRequest(
		func(prompt string) (string, error) {
			return requestOpenCodeZen(prompt, p.model, p.apiKey, p.requestsPM)
		},
		title,
		"opencode-zen",
	)
}

func buildTranslationTitleAndBodyPrompt(title, body, sourceLocale, targetLocale string) string {
	input := map[string]string{
		"source_locale": sourceLocale,
		"target_locale": targetLocale,
		"title":         title,
		"body":          body,
	}
	return buildTranslationPrompt(
		"Translate title and HTML body faithfully from source_locale to target_locale. Preserve HTML tags, links, entities, and code blocks in body.",
		input,
		`{"translated_title":"...","translated_body":"..."}`,
	)
}

func buildTranslationTitlePrompt(title, sourceLocale, targetLocale string) string {
	input := map[string]string{
		"source_locale": sourceLocale,
		"target_locale": targetLocale,
		"title":         title,
	}
	return buildTranslationPrompt(
		"Translate title faithfully from source_locale to target_locale.",
		input,
		`{"translated_title":"..."}`,
	)
}

func buildTranslationBodyChunkPrompt(body, sourceLocale, targetLocale string, chunkIndex, chunkCount int) string {
	input := map[string]any{
		"source_locale": sourceLocale,
		"target_locale": targetLocale,
		"chunk_index":   chunkIndex,
		"chunk_count":   chunkCount,
		"body":          body,
	}
	return buildTranslationPrompt(
		"Translate this HTML body fragment faithfully from source_locale to target_locale. Preserve HTML tags, links, entities, and code blocks. The fragment is one chunk of a longer document, so keep boundaries natural and do not add introductions or conclusions.",
		input,
		`{"translated_body":"..."}`,
	)
}

func buildTranslationPrompt(instruction string, input any, keys string) string {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return instruction + " Return ONLY valid JSON with exactly these keys: " + keys + ". Do not use Markdown fences."
	}
	return "You are a translation engine for blog content. " + instruction + " Return ONLY valid JSON with exactly these keys: " + keys + ". Do not use Markdown fences.\n" + string(inputJSON)
}

func generateSlugWithOpenCodeRequest(request func(string) (string, error), title, provider string) (string, error) {
	input := map[string]string{"title": title}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	prompt := "You generate concise English URL slugs for blog posts and pages from titles written in any language. Translate or transliterate the title into natural English keywords when needed. Return ONLY valid JSON with exactly this key: {\"slug\":\"...\"}. The slug must contain only lowercase ASCII letters, numbers, and single hyphens. Do not use Markdown fences.\n" + string(inputJSON)
	text, err := request(prompt)
	if err != nil {
		return "", err
	}
	var payload slugGenerationResponse
	if err := unmarshalTranslationJSON(text, &payload); err != nil {
		return "", err
	}
	slug := normalizeGeneratedSlug(payload.Slug)
	if slug == "" {
		return "", fmt.Errorf("%s returned an empty slug", provider)
	}
	return slug, nil
}

func parseTranslatedPayload(text, provider string) (string, string, error) {
	var payload translatedPayload
	if err := unmarshalTranslationJSON(text, &payload); err != nil {
		return "", "", err
	}
	payload.TranslatedTitle = strings.TrimSpace(payload.TranslatedTitle)
	payload.TranslatedBody = strings.TrimSpace(payload.TranslatedBody)
	if payload.TranslatedTitle == "" || payload.TranslatedBody == "" {
		return "", "", fmt.Errorf("%s returned empty title/body", provider)
	}
	return payload.TranslatedTitle, payload.TranslatedBody, nil
}

func parseTranslatedTitle(text, provider string) (string, error) {
	var payload translatedTitlePayload
	if err := unmarshalTranslationJSON(text, &payload); err != nil {
		return "", err
	}
	payload.TranslatedTitle = strings.TrimSpace(payload.TranslatedTitle)
	if payload.TranslatedTitle == "" {
		return "", fmt.Errorf("%s returned empty title", provider)
	}
	return payload.TranslatedTitle, nil
}

func parseTranslatedBody(text, provider string) (string, error) {
	var payload translatedBodyPayload
	if err := unmarshalTranslationJSON(text, &payload); err != nil {
		return "", err
	}
	payload.TranslatedBody = strings.TrimSpace(payload.TranslatedBody)
	if payload.TranslatedBody == "" {
		return "", fmt.Errorf("%s returned empty body", provider)
	}
	return payload.TranslatedBody, nil
}

func requestOpenCodeGoJSON(prompt, model, apiKey string, requestsPerMinute int) (string, error) {
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{{
			"role":    "user",
			"content": prompt,
		}},
		"temperature": 0.2,
		"stream":      false,
	}
	return requestOpenCodeJSON("opencode-go", openCodeGoChatCompletionsURL, payload, apiKey, requestsPerMinute, parseOpenCodeGoResponseText)
}

func requestOpenCodeZen(prompt, model, apiKey string, requestsPerMinute int) (string, error) {
	modelID := normalizeOpenCodeModelID(model)
	switch openCodeZenProtocolForModel(modelID) {
	case openCodeZenResponsesProtocol:
		payload := map[string]any{
			"model": modelID,
			"input": prompt,
		}
		return requestOpenCodeJSON("opencode-zen", openCodeZenResponsesURL, payload, apiKey, requestsPerMinute, parseOpenCodeZenResponseText)
	case openCodeZenMessagesProtocol:
		payload := map[string]any{
			"model":      modelID,
			"max_tokens": maxOpenCodeOutputTokens,
			"messages": []map[string]string{{
				"role":    "user",
				"content": prompt,
			}},
		}
		return requestOpenCodeJSONWithHeaders(
			"opencode-zen",
			openCodeZenMessagesURL,
			payload,
			apiKey,
			requestsPerMinute,
			parseOpenCodeZenMessagesResponseText,
			func(req *http.Request, key string) {
				req.Header.Set("x-api-key", key)
				req.Header.Set("anthropic-version", "2023-06-01")
			},
		)
	case openCodeZenGoogleProtocol:
		payload := map[string]any{
			"contents": []map[string]any{{
				"role": "user",
				"parts": []map[string]string{{
					"text": prompt,
				}},
			}},
		}
		endpoint := strings.TrimRight(openCodeZenGoogleModelsURL, "/") + "/" + url.PathEscape(modelID) + ":generateContent"
		return requestOpenCodeJSONWithHeaders(
			"opencode-zen",
			endpoint,
			payload,
			apiKey,
			requestsPerMinute,
			parseOpenCodeZenGoogleResponseText,
			func(req *http.Request, key string) {
				req.Header.Set("x-goog-api-key", key)
			},
		)
	default:
		payload := map[string]any{
			"model": modelID,
			"messages": []map[string]string{{
				"role":    "user",
				"content": prompt,
			}},
			"temperature": 0.2,
			"stream":      false,
		}
		return requestOpenCodeJSON("opencode-zen", openCodeZenChatCompletionsURL, payload, apiKey, requestsPerMinute, parseOpenCodeGoResponseText)
	}
}

func requestOpenCodeJSON(provider, endpoint string, payload any, apiKey string, requestsPerMinute int, parse func([]byte) (string, error)) (string, error) {
	return requestOpenCodeJSONWithHeaders(provider, endpoint, payload, apiKey, requestsPerMinute, parse, func(req *http.Request, key string) {
		req.Header.Set("Authorization", "Bearer "+key)
	})
}

func requestOpenCodeJSONWithHeaders(provider, endpoint string, payload any, apiKey string, requestsPerMinute int, parse func([]byte) (string, error), setHeaders func(*http.Request, string)) (string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 1; attempt <= maxTranslateRetries; attempt++ {
		sharedTranslationRateLimiter.Wait(requestsPerMinute)

		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		setHeaders(req, apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxTranslationResponseBytes+1))
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if len(respBody) > maxTranslationResponseBytes {
				lastErr = fmt.Errorf("%s response exceeds %d bytes", provider, maxTranslationResponseBytes)
			} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				if len(respBody) > maxTranslationErrorBytes {
					respBody = append(respBody[:maxTranslationErrorBytes], []byte("…(truncated)")...)
				}
				lastErr = &ProviderError{Provider: provider, StatusCode: resp.StatusCode, Body: string(respBody)}
			} else {
				text, parseErr := parse(respBody)
				if parseErr == nil {
					return text, nil
				}
				lastErr = parseErr
			}
		}
		if attempt < maxTranslateRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return "", lastErr
}

func parseOpenCodeGoResponseText(responseBody []byte) (string, error) {
	var result openCodeGoChatResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", errors.New("opencode-go returned no choices")
	}
	text := strings.TrimSpace(result.Choices[0].Message.Content)
	if text == "" {
		return "", errors.New("opencode-go returned empty content")
	}
	return extractFirstJSONObject(text)
}

func parseOpenCodeZenResponseText(responseBody []byte) (string, error) {
	var result openCodeZenResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}
	text := strings.TrimSpace(result.OutputText)
	if text == "" {
		for _, output := range result.Output {
			for _, content := range output.Content {
				if content.Type == "output_text" || content.Type == "text" || content.Type == "" {
					if candidate := strings.TrimSpace(content.Text); candidate != "" {
						text = candidate
						break
					}
				}
			}
			if text != "" {
				break
			}
		}
	}
	if text == "" {
		return "", errors.New("opencode-zen returned empty content")
	}
	return extractFirstJSONObject(text)
}

func parseOpenCodeZenMessagesResponseText(responseBody []byte) (string, error) {
	var result openCodeZenMessagesResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}
	for _, content := range result.Content {
		if text := strings.TrimSpace(content.Text); text != "" {
			return extractFirstJSONObject(text)
		}
	}
	return "", errors.New("opencode-zen returned empty content")
}

func parseOpenCodeZenGoogleResponseText(responseBody []byte) (string, error) {
	var result openCodeZenGoogleResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}
	for _, candidate := range result.Candidates {
		for _, part := range candidate.Content.Parts {
			if text := strings.TrimSpace(part.Text); text != "" {
				return extractFirstJSONObject(text)
			}
		}
	}
	return "", errors.New("opencode-zen returned empty content")
}

const (
	maxOpenCodeOutputTokens                          = 8192
	openCodeZenResponsesProtocol openCodeZenProtocol = "responses"
	openCodeZenMessagesProtocol  openCodeZenProtocol = "messages"
	openCodeZenGoogleProtocol    openCodeZenProtocol = "google"
	openCodeZenChatProtocol      openCodeZenProtocol = "chat-completions"
)

type openCodeZenProtocol string

func normalizeOpenCodeModelID(model string) string {
	return strings.TrimPrefix(strings.TrimSpace(model), "opencode/")
}

func openCodeZenProtocolForModel(model string) openCodeZenProtocol {
	modelID := strings.ToLower(normalizeOpenCodeModelID(model))
	switch {
	case strings.HasPrefix(modelID, "gpt-"), strings.HasPrefix(modelID, "grok-"), strings.HasPrefix(modelID, "muse-"):
		return openCodeZenResponsesProtocol
	case strings.HasPrefix(modelID, "claude-"), strings.HasPrefix(modelID, "qwen"):
		return openCodeZenMessagesProtocol
	case strings.HasPrefix(modelID, "gemini-"):
		return openCodeZenGoogleProtocol
	default:
		return openCodeZenChatProtocol
	}
}
