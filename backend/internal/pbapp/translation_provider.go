package pbapp

import (
	"errors"
	"fmt"
	"strings"
)

type translationProvider interface {
	translateTitleAndBody(title, body, sourceLocale, targetLocale string) (string, string, error)
	translateTitle(title, sourceLocale, targetLocale string) (string, error)
	translateBodyChunk(body, sourceLocale, targetLocale string, chunkIndex, chunkCount int) (string, error)
	generateSlug(title string) (string, error)
}

type geminiTranslationProvider struct {
	model      string
	apiKey     string
	requestsPM int
}

func newTranslationProvider(settings translationSettings) (translationProvider, error) {
	switch settings.Provider {
	case "", "gemini":
		return &geminiTranslationProvider{
			model:      settings.Model,
			apiKey:     settings.APIKey,
			requestsPM: settings.RequestsPM,
		}, nil
	case "opencode-go":
		return &openCodeGoTranslationProvider{
			model:      settings.Model,
			apiKey:     settings.APIKey,
			requestsPM: settings.RequestsPM,
		}, nil
	case "opencode-zen":
		return &openCodeZenTranslationProvider{
			model:      settings.Model,
			apiKey:     settings.APIKey,
			requestsPM: settings.RequestsPM,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported translation provider: %s", settings.Provider)
	}
}

func translate(settings translationSettings, title, body, targetLocale string) (string, string, error) {
	provider, err := newTranslationProvider(settings)
	if err != nil {
		return "", "", err
	}
	return translateWithProvider(provider, title, body, settings.SourceLocale, targetLocale)
}

func translateWithProvider(provider translationProvider, title, body, sourceLocale, targetLocale string) (string, string, error) {
	if provider == nil {
		return "", "", errors.New("translation provider is nil")
	}
	if len([]rune(body)) <= maxTranslationBodyRunes {
		return provider.translateTitleAndBody(title, body, sourceLocale, targetLocale)
	}

	translatedTitle, err := provider.translateTitle(title, sourceLocale, targetLocale)
	if err != nil {
		return "", "", err
	}

	chunks := splitTranslationBody(body, maxTranslationBodyRunes)
	if len(chunks) == 0 {
		return "", "", errors.New("translation body split produced no chunks")
	}

	translatedChunks := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		translatedChunk, err := provider.translateBodyChunk(chunk, sourceLocale, targetLocale, i+1, len(chunks))
		if err != nil {
			return "", "", err
		}
		translatedChunks = append(translatedChunks, translatedChunk)
	}

	return translatedTitle, strings.Join(translatedChunks, ""), nil
}

func (p *geminiTranslationProvider) translateTitleAndBody(title, body, sourceLocale, targetLocale string) (string, string, error) {
	return translateTitleAndBodyWithGemini(title, body, sourceLocale, targetLocale, p.model, p.apiKey, p.requestsPM)
}

func (p *geminiTranslationProvider) translateTitle(title, sourceLocale, targetLocale string) (string, error) {
	return translateTitleWithGemini(title, sourceLocale, targetLocale, p.model, p.apiKey, p.requestsPM)
}

func (p *geminiTranslationProvider) translateBodyChunk(body, sourceLocale, targetLocale string, chunkIndex, chunkCount int) (string, error) {
	return translateBodyChunkWithGemini(body, sourceLocale, targetLocale, p.model, p.apiKey, p.requestsPM, chunkIndex, chunkCount)
}

func (p *geminiTranslationProvider) generateSlug(title string) (string, error) {
	return generateSlugWithRequest(
		func(prompt string) (string, error) {
			return requestGeminiJSON(prompt, p.model, p.apiKey, p.requestsPM, geminiResponseSchema("slug"))
		},
		title,
		"gemini",
	)
}
