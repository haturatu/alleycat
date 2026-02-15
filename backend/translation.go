package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

const (
	defaultTranslationModel = "gemini-1.5-flash"
	maxTranslateRetries     = 3
)

type translationSettings struct {
	Enabled      bool
	SourceLocale string
	Locales      []string
	Model        string
	APIKey       string
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type translatedPayload struct {
	TranslatedTitle string `json:"translated_title"`
	TranslatedBody  string `json:"translated_body"`
}

type geminiError struct {
	Status int
	Body   string
}

func (e *geminiError) Error() string {
	return fmt.Sprintf("gemini request failed: status=%d", e.Status)
}

func registerTranslationFeatures(app *pocketbase.PocketBase) {
	registerTranslateCommand(app)

	app.OnRecordAfterCreateSuccess("posts").BindFunc(func(e *core.RecordEvent) error {
		triggerPostTranslation(e.App, e.Record)
		return e.Next()
	})
	app.OnRecordAfterUpdateSuccess("posts").BindFunc(func(e *core.RecordEvent) error {
		triggerPostTranslation(e.App, e.Record)
		return e.Next()
	})
}

func registerTranslateCommand(app *pocketbase.PocketBase) {
	cmd := &cobra.Command{
		Use:   "translate-posts",
		Short: "Translate existing source posts using Gemini",
		RunE: func(command *cobra.Command, args []string) error {
			if err := app.Bootstrap(); err != nil {
				return err
			}
			return translateAllSourcePosts(app)
		},
	}
	app.RootCmd.AddCommand(cmd)
}

func triggerPostTranslation(app core.App, source *core.Record) {
	if source == nil {
		return
	}
	settings, err := loadTranslationSettings(app)
	if err != nil {
		log.Printf("translation settings load failed: %v", err)
		return
	}
	if !settings.Enabled || strings.TrimSpace(settings.APIKey) == "" || len(settings.Locales) == 0 {
		return
	}
	if err := translateSourcePost(app, source, settings); err != nil {
		log.Printf("translation failed for source post=%s: %v", source.Id, err)
	}
}

func translateAllSourcePosts(app core.App) error {
	settings, err := loadTranslationSettings(app)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return errors.New("post translation is disabled in settings")
	}
	if strings.TrimSpace(settings.APIKey) == "" {
		return errors.New("gemini_api_key is empty in settings")
	}
	if len(settings.Locales) == 0 {
		return errors.New("translation_locales is empty in settings")
	}

	records, err := app.FindRecordsByFilter(
		"posts",
		"id != ''",
		"-published_at",
		0,
		0,
	)
	if err != nil {
		return err
	}

	success := 0
	failed := 0
	for _, source := range records {
		if err := translateSourcePostForMigration(app, source, settings); err != nil {
			failed++
			log.Printf("translate-posts: failed source=%s slug=%s err=%v", source.Id, source.GetString("slug"), err)
			continue
		}
		success++
	}

	log.Printf("translate-posts finished: success=%d failed=%d total=%d", success, failed, len(records))
	if failed > 0 {
		return fmt.Errorf("translation failed for %d posts", failed)
	}
	return nil
}

func translateSourcePost(app core.App, source *core.Record, settings translationSettings) error {
	title := strings.TrimSpace(source.GetString("title"))
	body := strings.TrimSpace(source.GetString("body"))
	if body == "" {
		body = strings.TrimSpace(source.GetString("content"))
	}
	if title == "" || body == "" {
		return nil
	}

	var firstErr error
	for _, locale := range settings.Locales {
		if err := upsertTranslatedPost(app, source, locale, settings, title, body, true); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			log.Printf("translation locale failed source=%s locale=%s err=%v", source.Id, locale, err)
		}
	}
	return firstErr
}

func translateSourcePostForMigration(app core.App, source *core.Record, settings translationSettings) error {
	title := strings.TrimSpace(source.GetString("title"))
	body := strings.TrimSpace(source.GetString("body"))
	if body == "" {
		body = strings.TrimSpace(source.GetString("content"))
	}
	if title == "" || body == "" {
		return nil
	}

	var firstErr error
	for _, locale := range settings.Locales {
		if err := upsertTranslatedPost(app, source, locale, settings, title, body, false); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			log.Printf("translation locale failed source=%s locale=%s err=%v", source.Id, locale, err)
		}
	}
	return firstErr
}

func upsertTranslatedPost(
	app core.App,
	source *core.Record,
	targetLocale string,
	settings translationSettings,
	title string,
	body string,
	force bool,
) error {
	translationsCollection, err := app.FindCollectionByNameOrId("post_translations")
	if err != nil {
		return err
	}

	translated, err := app.FindFirstRecordByFilter(
		translationsCollection,
		"source_post = {:source} && locale = {:locale}",
		dbx.Params{"source": source.Id, "locale": targetLocale},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if translated != nil && translated.GetBool("translation_done") && !force {
		return nil
	}

	translatedTitle, translatedBody, err := translateWithGemini(
		title,
		body,
		settings.SourceLocale,
		targetLocale,
		settings.Model,
		settings.APIKey,
	)
	if err != nil {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) || translated == nil {
		translated = core.NewRecord(translationsCollection)
		translated.Set("source_post", source.Id)
		translated.Set("locale", targetLocale)
	}

	translated.Set("slug", source.GetString("slug"))
	translated.Set("tags", source.GetString("tags"))
	translated.Set("category", source.GetString("category"))
	translated.Set("author", source.GetString("author"))
	translated.Set("published", source.GetBool("published"))
	translated.Set("published_at", source.GetString("published_at"))
	translated.Set("title", translatedTitle)
	translated.Set("body", translatedBody)
	translated.Set("excerpt", buildExcerpt(translatedBody, 160))
	translated.Set("translation_done", true)

	return app.Save(translated)
}

func loadTranslationSettings(app core.App) (translationSettings, error) {
	record, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return translationSettings{}, err
	}
	if errors.Is(err, sql.ErrNoRows) || record == nil {
		apiKey, keyErr := loadGeminiAPIKey(app, nil)
		if keyErr != nil {
			return translationSettings{}, keyErr
		}
		return translationSettings{
			Enabled:      false,
			SourceLocale: "ja",
			Locales:      []string{"en"},
			Model:        defaultTranslationModel,
			APIKey:       apiKey,
		}, nil
	}

	sourceLocale := normalizeLocale(record.GetString("translation_source_locale"))
	if sourceLocale == "" {
		sourceLocale = normalizeLocale(record.GetString("site_language"))
	}
	if sourceLocale == "" {
		sourceLocale = "ja"
	}

	locales := parseLocaleList(record.GetString("translation_locales"))
	if len(locales) == 0 {
		locales = []string{"en"}
	}
	filtered := make([]string, 0, len(locales))
	for _, locale := range locales {
		if locale == sourceLocale {
			continue
		}
		filtered = append(filtered, locale)
	}

	model := strings.TrimSpace(record.GetString("translation_model"))
	if model == "" {
		model = defaultTranslationModel
	}
	apiKey, keyErr := loadGeminiAPIKey(app, record)
	if keyErr != nil {
		return translationSettings{}, keyErr
	}
	return translationSettings{
		Enabled:      record.GetBool("enable_post_translation"),
		SourceLocale: sourceLocale,
		Locales:      filtered,
		Model:        model,
		APIKey:       apiKey,
	}, nil
}

func loadGeminiAPIKey(app core.App, settingsRecord *core.Record) (string, error) {
	secret, err := app.FindFirstRecordByFilter("app_secrets", "id != ''")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if err == nil && secret != nil {
		key := strings.TrimSpace(secret.GetString("gemini_api_key"))
		if key != "" {
			return key, nil
		}
	}
	if settingsRecord != nil {
		return strings.TrimSpace(settingsRecord.GetString("gemini_api_key")), nil
	}
	return "", nil
}

func parseLocaleList(value string) []string {
	raw := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == ';'
	})
	result := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		locale := normalizeLocale(item)
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}
		seen[locale] = struct{}{}
		result = append(result, locale)
	}
	return result
}

func normalizeLocale(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	return trimmed
}

func buildExcerpt(input string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// Translated body can be HTML. Build a plain-text excerpt by ignoring tags.
	var plain strings.Builder
	inTag := false
	for _, r := range input {
		switch r {
		case '<':
			inTag = true
			plain.WriteRune(' ')
		case '>':
			inTag = false
		default:
			if !inTag {
				plain.WriteRune(r)
			}
		}
	}

	normalized := strings.Join(strings.Fields(plain.String()), " ")
	runes := []rune(normalized)
	if len(runes) <= maxLen {
		return normalized
	}
	return strings.TrimSpace(string(runes[:maxLen]))
}

func translateWithGemini(
	title string,
	body string,
	sourceLocale string,
	targetLocale string,
	model string,
	apiKey string,
) (string, string, error) {
	input := map[string]string{
		"source_locale": sourceLocale,
		"target_locale": targetLocale,
		"title":         title,
		"body":          body,
	}
	inputJSON, _ := json.Marshal(input)

	prompt := "You are a translation engine for blog content. " +
		"Translate title and HTML body faithfully from source_locale to target_locale. " +
		"Preserve HTML tags, links, and code blocks in body. " +
		"Return only JSON with keys translated_title and translated_body.\n" +
		string(inputJSON)

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"temperature":      0.2,
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model,
		apiKey,
	)

	var lastErr error
	for attempt := 1; attempt <= maxTranslateRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			respBody, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				lastErr = &geminiError{
					Status: resp.StatusCode,
					Body:   string(respBody),
				}
			} else {
				result, parseErr := parseGeminiTranslation(respBody)
				if parseErr == nil {
					return result.TranslatedTitle, result.TranslatedBody, nil
				}
				lastErr = parseErr
			}
		}
		if attempt < maxTranslateRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return "", "", lastErr
}

func parseGeminiTranslation(responseBody []byte) (translatedPayload, error) {
	var res geminiResponse
	if err := json.Unmarshal(responseBody, &res); err != nil {
		return translatedPayload{}, err
	}
	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return translatedPayload{}, errors.New("empty gemini candidates")
	}
	text := strings.TrimSpace(res.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var payload translatedPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return translatedPayload{}, err
	}
	payload.TranslatedTitle = strings.TrimSpace(payload.TranslatedTitle)
	payload.TranslatedBody = strings.TrimSpace(payload.TranslatedBody)
	if payload.TranslatedTitle == "" || payload.TranslatedBody == "" {
		return translatedPayload{}, errors.New("gemini translation returned empty title/body")
	}
	return payload, nil
}
