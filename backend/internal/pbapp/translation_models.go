package pbapp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const maxTranslationModelsResponseBytes = 2 * 1024 * 1024

type translationModelsResponse struct {
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
}

type geminiModelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`
}

type openCodeModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Models []struct {
		ID string `json:"id"`
	} `json:"models"`
}

func registerTranslationModelsAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/ai/translation/models", func(e *core.RequestEvent) error {
			if err := requireEditorOrAdminAuth(e); err != nil {
				return err
			}

			provider := strings.ToLower(strings.TrimSpace(e.Request.URL.Query().Get("provider")))
			if provider == "" {
				settings, err := loadTranslationSettings(e.App)
				if err != nil {
					return err
				}
				provider = settings.Provider
			}
			if !isSupportedTranslationProvider(provider) {
				return fmt.Errorf("unsupported translation provider: %s", provider)
			}

			settingsRecord, err := e.App.FindFirstRecordByFilter("settings", "id != ''")
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			apiKey, err := loadTranslationAPIKey(e.App, settingsRecord, provider)
			if err != nil {
				return err
			}
			models := []string{}
			if provider != "gemini" || strings.TrimSpace(apiKey) != "" {
				models, err = fetchTranslationModels(provider, apiKey)
				if err != nil {
					return err
				}
			}

			return e.JSON(http.StatusOK, translationModelsResponse{Provider: provider, Models: models})
		}).Bind()

		return se.Next()
	})
}

func fetchTranslationModels(provider, apiKey string) ([]string, error) {
	endpoint, err := translationModelsEndpoint(provider, apiKey)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if provider != "gemini" && strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTranslationModelsResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxTranslationModelsResponseBytes {
		return nil, fmt.Errorf("%s models response exceeds %d bytes", provider, maxTranslationModelsResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProviderError{Provider: provider, StatusCode: resp.StatusCode, Body: string(body)}
	}

	if provider == "gemini" {
		return parseGeminiModels(body)
	}
	return parseOpenCodeModels(body)
}

func translationModelsEndpoint(provider, apiKey string) (string, error) {
	switch provider {
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/models?key=" + url.QueryEscape(apiKey), nil
	case "opencode-go":
		return openCodeGoModelsURL, nil
	case "opencode-zen":
		return openCodeZenModelsURL, nil
	default:
		return "", fmt.Errorf("unsupported translation provider: %s", provider)
	}
}

func parseGeminiModels(body []byte) ([]string, error) {
	var response geminiModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(response.Models))
	for _, model := range response.Models {
		for _, method := range model.SupportedGenerationMethods {
			if method == "generateContent" {
				models = append(models, strings.TrimPrefix(model.Name, "models/"))
				break
			}
		}
	}
	return uniqueSortedModels(models), nil
}

func parseOpenCodeModels(body []byte) ([]string, error) {
	var response openCodeModelsResponse
	if err := json.Unmarshal(body, &response); err == nil {
		models := make([]string, 0, len(response.Data)+len(response.Models))
		for _, model := range response.Data {
			models = append(models, model.ID)
		}
		for _, model := range response.Models {
			models = append(models, model.ID)
		}
		if len(models) > 0 {
			return uniqueSortedModels(models), nil
		}
	}

	var models []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(models))
	for _, model := range models {
		result = append(result, model.ID)
	}
	return uniqueSortedModels(result), nil
}

func uniqueSortedModels(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	result := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	sort.Strings(result)
	return result
}
