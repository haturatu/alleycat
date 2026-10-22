package site

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type postMetaInput struct {
	Path                    string
	Locale                  string
	Title                   string
	Description             string
	PublishedAt             string
	ModifiedAt              string
	ImageURL                string
	ImageAlt                string
	FeaturedImage           string
	FeaturedImageCollection string
	FeaturedImageRecordID   string
	Author                  *AuthorRecord
}

func renderPostMetaTags(input postMetaInput, settings SettingsRecord) string {
	canonicalURL := buildAbsoluteSiteURL(settings, input.Path)
	if canonicalURL == "" {
		canonicalURL = input.Path
	}

	description := strings.TrimSpace(input.Description)
	if description == "" {
		description = strings.TrimSpace(settings.Description)
	}
	title := strings.TrimSpace(input.Title)
	imageURL := resolvePostImageURL(input, settings)
	imageAlt := strings.TrimSpace(input.ImageAlt)
	if imageAlt == "" {
		imageAlt = title
	}

	cardType := "summary"
	if imageURL != "" {
		cardType = "summary_large_image"
	}
	parts := []string{
		fmt.Sprintf(`<link rel="canonical" href="%s" />`, escapeHTML(canonicalURL)),
		`<meta property="og:type" content="article" />`,
		fmt.Sprintf(`<meta property="og:title" content="%s" />`, escapeHTML(title)),
		fmt.Sprintf(`<meta property="og:description" content="%s" />`, escapeHTML(description)),
		fmt.Sprintf(`<meta property="og:url" content="%s" />`, escapeHTML(canonicalURL)),
		fmt.Sprintf(`<meta property="og:site_name" content="%s" />`, escapeHTML(settings.SiteName)),
		fmt.Sprintf(`<meta name="twitter:card" content="%s" />`, cardType),
		fmt.Sprintf(`<meta name="twitter:title" content="%s" />`, escapeHTML(title)),
		fmt.Sprintf(`<meta name="twitter:description" content="%s" />`, escapeHTML(description)),
	}

	if locale := normalizeLocale(input.Locale); locale != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="og:locale" content="%s" />`, escapeHTML(strings.ReplaceAll(locale, "-", "_"))))
	}
	if publishedAt := strings.TrimSpace(input.PublishedAt); publishedAt != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="article:published_time" content="%s" />`, escapeHTML(publishedAt)))
	}
	if modifiedAt := strings.TrimSpace(input.ModifiedAt); modifiedAt != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="article:modified_time" content="%s" />`, escapeHTML(modifiedAt)))
	}
	if authorName := postAuthorName(input.Author); authorName != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="article:author" content="%s" />`, escapeHTML(authorName)))
	}
	if imageURL != "" {
		parts = append(parts,
			fmt.Sprintf(`<meta property="og:image" content="%s" />`, escapeHTML(imageURL)),
			fmt.Sprintf(`<meta name="twitter:image" content="%s" />`, escapeHTML(imageURL)),
			fmt.Sprintf(`<meta name="twitter:image:alt" content="%s" />`, escapeHTML(imageAlt)),
		)
		if strings.TrimSpace(input.ImageURL) == "" && strings.TrimSpace(input.FeaturedImage) == "" {
			parts = append(parts,
				fmt.Sprintf(`<meta property="og:image:width" content="%d" />`, postOGImageWidth),
				fmt.Sprintf(`<meta property="og:image:height" content="%d" />`, postOGImageHeight),
			)
		}
	}

	return strings.Join(parts, "\n    ")
}

func renderPostStructuredData(input postMetaInput, settings SettingsRecord) string {
	canonicalURL := buildAbsoluteSiteURL(settings, input.Path)
	if canonicalURL == "" {
		canonicalURL = input.Path
	}
	description := strings.TrimSpace(input.Description)
	if description == "" {
		description = strings.TrimSpace(settings.Description)
	}

	data := blogPostingStructuredData{
		Context:       "https://schema.org",
		Type:          "BlogPosting",
		Headline:      strings.TrimSpace(input.Title),
		Description:   description,
		URL:           canonicalURL,
		Image:         resolvePostImageURL(input, settings),
		DatePublished: strings.TrimSpace(input.PublishedAt),
		DateModified:  strings.TrimSpace(input.ModifiedAt),
		Author:        buildStructuredDataAuthor(input.Author),
		Publisher: structuredDataPublisher{
			Type: "Organization",
			Name: strings.TrimSpace(settings.SiteName),
		},
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(`<script type="application/ld+json">%s</script>`, encoded)
}

type blogPostingStructuredData struct {
	Context       string                  `json:"@context"`
	Type          string                  `json:"@type"`
	Headline      string                  `json:"headline"`
	Description   string                  `json:"description,omitempty"`
	URL           string                  `json:"url"`
	Image         string                  `json:"image,omitempty"`
	DatePublished string                  `json:"datePublished,omitempty"`
	DateModified  string                  `json:"dateModified,omitempty"`
	Author        *structuredDataAuthor   `json:"author,omitempty"`
	Publisher     structuredDataPublisher `json:"publisher"`
}

type structuredDataAuthor struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

type structuredDataPublisher struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

func buildStructuredDataAuthor(author *AuthorRecord) *structuredDataAuthor {
	name := postAuthorName(author)
	if name == "" {
		return nil
	}
	return &structuredDataAuthor{Type: "Person", Name: name}
}

func postAuthorName(author *AuthorRecord) string {
	if author == nil {
		return ""
	}
	return strings.TrimSpace(author.Name)
}

func postAuthor(post *PostRecord) *AuthorRecord {
	if post == nil || post.Expand == nil || post.Expand.Author == nil {
		return nil
	}
	return post.Expand.Author
}

func featuredImageURL(post *PostRecord, translation *PostTranslationRecord, settings SettingsRecord) string {
	if post == nil {
		return ""
	}
	collection := "posts"
	recordID := post.ID
	filename := post.FeaturedImage
	if translation != nil {
		collection = "post_translations"
		recordID = translation.ID
		filename = translation.FeaturedImage
	}
	if strings.TrimSpace(filename) == "" {
		return ""
	}
	return buildAbsoluteSiteURL(settings, featuredImagePath(collection, recordID, filename))
}

func featuredImagePath(collection, recordID, filename string) string {
	collection = strings.TrimSpace(collection)
	recordID = strings.TrimSpace(recordID)
	filename = strings.TrimSpace(filename)
	if collection == "" || recordID == "" || filename == "" {
		return ""
	}
	return fmt.Sprintf("/api/files/%s/%s/%s", url.PathEscape(collection), url.PathEscape(recordID), url.PathEscape(filename))
}

func resolvePostImageURL(input postMetaInput, settings SettingsRecord) string {
	if strings.TrimSpace(input.FeaturedImage) != "" {
		collection := input.FeaturedImageCollection
		if strings.TrimSpace(collection) == "" {
			collection = "posts"
		}
		path := featuredImagePath(collection, input.FeaturedImageRecordID, input.FeaturedImage)
		if path != "" {
			return buildAbsoluteSiteURL(settings, path)
		}
	}
	if imageURL := strings.TrimSpace(input.ImageURL); imageURL != "" {
		return imageURL
	}
	if !settings.EnableOGPImageGeneration {
		return ""
	}
	imageLocale := extractLocaleFromPostPath(input.Path)
	if imageLocale == "" {
		if sourceLocale := normalizeLocale(settings.TranslationSourceLocale); sourceLocale != "" {
			imageLocale = sourceLocale
		} else {
			imageLocale = normalizeLocale(settings.SiteLanguage)
		}
	}
	path := postOGImageRoute(imageLocale, extractSlugFromPostPath(input.Path))
	if path == "" {
		return ""
	}
	return buildAbsoluteSiteURL(settings, path)
}

func renderPostAuthor(author *AuthorRecord) string {
	name := postAuthorName(author)
	if name == "" {
		return ""
	}
	return fmt.Sprintf(`<p class="post-author">Written by <a href="/about/">%s</a></p>`, escapeHTML(name))
}

func renderPostFeaturedImage(imageURL, alt string) string {
	if strings.TrimSpace(imageURL) == "" {
		return ""
	}
	return fmt.Sprintf(`<figure class="post-featured-image"><img src="%s" alt="%s" width="1200" height="630" fetchpriority="high" decoding="async" /></figure>`, escapeHTML(imageURL), escapeHTML(strings.TrimSpace(alt)))
}
