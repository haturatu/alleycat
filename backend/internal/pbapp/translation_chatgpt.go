package pbapp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var (
	chatGPTResponsesURL = "https://api.openai.com/v1/responses"
	chatGPTModelsURL    = "https://api.openai.com/v1/models"
)

type chatGPTTranslationProvider struct {
	app        core.App
	model      string
	requestsPM int
}

func (p *chatGPTTranslationProvider) translateTitleAndBody(title, body, sourceLocale, targetLocale string) (string, string, error) {
	text, err := p.request(buildTranslationTitleAndBodyPrompt(title, body, sourceLocale, targetLocale))
	if err != nil {
		return "", "", err
	}
	return parseTranslatedPayload(text, "chatgpt")
}

func (p *chatGPTTranslationProvider) translateTitle(title, sourceLocale, targetLocale string) (string, error) {
	text, err := p.request(buildTranslationTitlePrompt(title, sourceLocale, targetLocale))
	if err != nil {
		return "", err
	}
	return parseTranslatedTitle(text, "chatgpt")
}

func (p *chatGPTTranslationProvider) translateBodyChunk(body, sourceLocale, targetLocale string, index, count int) (string, error) {
	text, err := p.request(buildTranslationBodyChunkPrompt(body, sourceLocale, targetLocale, index, count))
	if err != nil {
		return "", err
	}
	return parseTranslatedBody(text, "chatgpt")
}

func (p *chatGPTTranslationProvider) generateSlug(title string) (string, error) {
	return generateSlugWithRequest(p.request, title, "chatgpt")
}

func (p *chatGPTTranslationProvider) request(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	model := strings.TrimSpace(p.model)
	if model == "" {
		models, err := fetchChatGPTModels(p.app)
		if err != nil {
			return "", err
		}
		if len(models) == 0 {
			return "", errors.New("ChatGPT has no available models for this account")
		}
		model = models[0]
	}
	sharedTranslationRateLimiter.Wait(p.requestsPM)
	body, err := json.Marshal(map[string]any{
		"model":        model,
		"input":        []map[string]string{{"role": "user", "content": prompt}},
		"instructions": "Return only the requested JSON. Do not use tools or Markdown fences.",
		"store":        false,
		"stream":       true,
	})
	if err != nil {
		return "", err
	}
	response, err := chatGPTAuthenticatedRequest(ctx, p.app, http.MethodPost, chatGPTResponsesURL, body)
	if err != nil {
		return "", err
	}
	return parseChatGPTResponseStream(response)
}

func chatGPTAuthenticatedRequest(ctx context.Context, app core.App, method, endpoint string, body []byte) ([]byte, error) {
	rejectedToken := ""
	for attempt := 0; attempt < 2; attempt++ {
		token, err := chatGPTAccessToken(ctx, app, rejectedToken)
		if err != nil {
			return nil, &ProviderError{Provider: "chatgpt", StatusCode: http.StatusUnauthorized, Body: err.Error()}
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := chatGPTClient.Do(req)
		if err != nil {
			return nil, errors.New("ChatGPT request could not be completed")
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxTranslationResponseBytes+1))
		_ = resp.Body.Close()
		if readErr != nil || len(data) > maxTranslationResponseBytes {
			return nil, errors.New("ChatGPT response is unreadable or too large")
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			rejectedToken = token
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			message := strings.ReplaceAll(string(data), token, "[redacted]")
			if requestID := resp.Header.Get("x-request-id"); requestID != "" {
				message = "request_id=" + requestID + " " + message
			}
			return nil, &ProviderError{Provider: "chatgpt", StatusCode: resp.StatusCode, Body: message}
		}
		return data, nil
	}
	return nil, errors.New("ChatGPT authentication failed; reconnect in Settings")
}

func fetchChatGPTModels(app core.App) ([]string, error) {
	credentials, err := loadChatGPTCredentials(app)
	if err != nil {
		return nil, err
	}
	if credentials.AccessToken == "" {
		return []string{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	body, err := chatGPTAuthenticatedRequest(ctx, app, http.MethodGet, chatGPTModelsURL, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Models []struct {
			Slug       string `json:"slug"`
			Visibility string `json:"visibility"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.New("ChatGPT returned an invalid model catalog")
	}
	models := []string{}
	seen := map[string]bool{}
	for _, model := range response.Models {
		if model.Visibility == "list" && model.Slug != "" && !seen[model.Slug] {
			models = append(models, model.Slug)
			seen[model.Slug] = true
		}
	}
	return models, nil
}

func parseChatGPTResponseStream(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 4096), maxTranslationResponseBytes)
	var data, text strings.Builder
	completed := false
	consume := func() error {
		if data.Len() == 0 {
			return nil
		}
		raw := strings.TrimSpace(data.String())
		data.Reset()
		if raw == "[DONE]" {
			return nil
		}
		var event struct {
			Type     string `json:"type"`
			Delta    string `json:"delta"`
			Response struct {
				Status string `json:"status"`
				Output []struct {
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"output"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(raw), &event) != nil {
			return errors.New("ChatGPT returned an invalid stream event")
		}
		switch event.Type {
		case "response.output_text.delta":
			text.WriteString(event.Delta)
		case "response.completed":
			if event.Response.Status != "completed" {
				return errors.New("ChatGPT response did not complete")
			}
			var final strings.Builder
			for _, output := range event.Response.Output {
				for _, content := range output.Content {
					if content.Type == "output_text" {
						final.WriteString(content.Text)
					}
				}
			}
			if final.Len() > 0 {
				text.Reset()
				text.WriteString(final.String())
			}
			completed = true
		case "response.failed", "response.incomplete", "error":
			return errors.New("ChatGPT generation failed or was incomplete; check the account and usage limits")
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := consume(); err != nil {
				return "", err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			data.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if err := consume(); err != nil {
		return "", err
	}
	if !completed || strings.TrimSpace(text.String()) == "" {
		return "", fmt.Errorf("ChatGPT stream ended without a completed text response")
	}
	return extractFirstJSONObject(text.String())
}
