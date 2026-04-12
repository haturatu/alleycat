package pbapp

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type slugGenerationRequest struct {
	Title string `json:"title"`
}

type slugGenerationResponse struct {
	Slug string `json:"slug"`
}

var (
	invalidSlugCharRe = regexp.MustCompile(`[^a-z0-9\s-]`)
	repeatedHyphenRe  = regexp.MustCompile(`-+`)
	repeatedSpaceRe   = regexp.MustCompile(`\s+`)
)

func registerSlugGenerationAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/ai/slug/status", func(e *core.RequestEvent) error {
			if err := requireEditorOrAdminAuth(e); err != nil {
				return err
			}

			apiKey, err := loadGeminiAPIKey(e.App, nil)
			if err != nil {
				return err
			}

			return e.JSON(http.StatusOK, map[string]any{
				"enabled": strings.TrimSpace(apiKey) != "",
			})
		}).Bind(apis.RequireAuth())

		se.Router.POST("/api/ai/slug", func(e *core.RequestEvent) error {
			if err := requireEditorOrAdminAuth(e); err != nil {
				return err
			}

			var req slugGenerationRequest
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("Invalid request body.", err)
			}

			title := strings.TrimSpace(req.Title)
			if title == "" {
				return apis.NewBadRequestError("Title is required.", nil)
			}

			settings, err := loadTranslationSettings(e.App)
			if err != nil {
				return err
			}
			if strings.TrimSpace(settings.APIKey) == "" {
				return apis.NewBadRequestError("Gemini API key is not configured.", nil)
			}

			slug, err := generateEnglishSlugWithGemini(title, settings.Model, settings.APIKey, settings.RequestsPM)
			if err != nil {
				return err
			}

			return e.JSON(http.StatusOK, slugGenerationResponse{Slug: slug})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

func requireEditorOrAdminAuth(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("Authentication required.", nil)
	}

	role := strings.TrimSpace(e.Auth.GetString("role"))
	if role != "admin" && role != "editor" {
		return apis.NewForbiddenError("Editor or admin role required.", nil)
	}

	return nil
}

func generateEnglishSlugWithGemini(title, model, apiKey string, requestsPerMinute int) (string, error) {
	input := map[string]string{
		"title": title,
	}
	inputJSON, _ := json.Marshal(input)

	prompt := "You generate concise English URL slugs for blog posts and pages from titles written in any language. " +
		"Translate or transliterate the title into natural English keywords when needed. " +
		"Return only JSON with key slug. " +
		"The slug must contain only lowercase ASCII letters, numbers, and single hyphens.\n" +
		string(inputJSON)

	text, err := requestGeminiJSON(prompt, model, apiKey, requestsPerMinute)
	if err != nil {
		return "", err
	}

	var payload slugGenerationResponse
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return "", err
	}

	slug := normalizeGeneratedSlug(payload.Slug)
	if slug == "" {
		return "", apis.NewBadRequestError("Gemini returned an empty slug.", nil)
	}

	return slug, nil
}

func normalizeGeneratedSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = repeatedSpaceRe.ReplaceAllString(value, "-")
	value = invalidSlugCharRe.ReplaceAllString(value, "")
	value = repeatedHyphenRe.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
