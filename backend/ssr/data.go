package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func getPosts(params map[string]string) (PBList[PostRecord], error) {
	return fetchList[PostRecord](fmt.Sprintf("%s/api/collections/posts/records", pbURL), params)
}

func getPostTranslations(params map[string]string) (PBList[PostTranslationRecord], error) {
	return fetchList[PostTranslationRecord](fmt.Sprintf("%s/api/collections/post_translations/records", pbURL), params)
}

func getPagesMenu() []PageRecord {
	data, err := fetchList[PageRecord](fmt.Sprintf("%s/api/collections/pages/records", pbURL), map[string]string{
		"perPage": "200",
		"filter":  "published = true && menuVisible = true",
		"sort":    "menuOrder",
	})
	if err != nil {
		return nil
	}
	return data.Items
}

func getPageByURL(path string) *PageRecord {
	data, err := fetchList[PageRecord](fmt.Sprintf("%s/api/collections/pages/records", pbURL), map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("url = \"%s\" && published = true", escapeFilter(path)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func getPostBySlug(slug string) *PostRecord {
	return getPostBySlugInLocale(slug, "")
}

func getPostBySlugInLocale(slug string, locale string) *PostRecord {
	if locale == "" {
		data, err := fetchList[PostRecord](fmt.Sprintf("%s/api/collections/posts/records", pbURL), map[string]string{
			"perPage": "1",
			"filter":  fmt.Sprintf("slug = \"%s\" && published = true", escapeFilter(slug)),
		})
		if err != nil || len(data.Items) == 0 {
			return nil
		}
		return &data.Items[0]
	}

	data, err := getPostTranslations(map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("slug = \"%s\" && locale = \"%s\" && published = true", escapeFilter(slug), escapeFilter(locale)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	post := translationToPost(data.Items[0])
	return &post
}

func getPostByID(id string) *PostRecord {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	post, err := fetchRecord[PostRecord](fmt.Sprintf("%s/api/collections/posts/records/%s", pbURL, url.PathEscape(id)))
	if err != nil {
		return nil
	}
	return &post
}

func getPostTranslationBySlugLocale(slug string, locale string) *PostTranslationRecord {
	if strings.TrimSpace(slug) == "" || strings.TrimSpace(locale) == "" {
		return nil
	}
	data, err := getPostTranslations(map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("slug = \"%s\" && locale = \"%s\" && published = true", escapeFilter(slug), escapeFilter(locale)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func getPostTranslationsBySource(sourcePostID string) []PostTranslationRecord {
	if strings.TrimSpace(sourcePostID) == "" {
		return nil
	}
	data, err := getPostTranslations(map[string]string{
		"page":    "1",
		"perPage": "200",
		"filter":  fmt.Sprintf("source_post = \"%s\" && published = true", escapeFilter(sourcePostID)),
		"sort":    "locale",
	})
	if err != nil {
		return nil
	}
	return data.Items
}

func getAdjacentPosts(post *PostRecord) (newer *PostRecord, older *PostRecord) {
	return getAdjacentPostsInLocale(post, "")
}

func getAdjacentPostsInLocale(post *PostRecord, locale string) (newer *PostRecord, older *PostRecord) {
	if post == nil {
		return nil, nil
	}
	field := ""
	value := ""
	if strings.TrimSpace(post.PublishedAt) != "" {
		field = "published_at"
		value = post.PublishedAt
	} else if strings.TrimSpace(post.Date) != "" {
		field = "date"
		value = post.Date
	}
	if field == "" || value == "" {
		return nil, nil
	}

	if locale == "" {
		fetchNearest := func(op, sort string) *PostRecord {
			data, err := getPosts(map[string]string{
				"page":    "1",
				"perPage": "1",
				"filter":  fmt.Sprintf("published = true && %s %s \"%s\"", field, op, escapeFilter(value)),
				"sort":    sort,
			})
			if err != nil || len(data.Items) == 0 {
				return nil
			}
			return &data.Items[0]
		}
		newer = fetchNearest(">", field)
		older = fetchNearest("<", "-"+field)
		return newer, older
	}

	fetchNearestTranslated := func(op, sort string) *PostRecord {
		data, err := getPostTranslations(map[string]string{
			"page":    "1",
			"perPage": "1",
			"filter":  fmt.Sprintf("published = true && locale = \"%s\" && %s %s \"%s\"", escapeFilter(locale), field, op, escapeFilter(value)),
			"sort":    sort,
		})
		if err != nil || len(data.Items) == 0 {
			return nil
		}
		item := translationToPost(data.Items[0])
		return &item
	}
	newer = fetchNearestTranslated(">", field)
	older = fetchNearestTranslated("<", "-"+field)
	return newer, older
}

func getRelatedPostsInLocale(post *PostRecord, locale string, limit int) []PostRecord {
	if post == nil {
		return nil
	}
	if limit <= 0 {
		limit = 4
	}

	targetTags := make(map[string]struct{})
	for _, tag := range parseTags(post.Tags) {
		targetTags[strings.ToLower(tag)] = struct{}{}
	}
	targetCategory := strings.ToLower(strings.TrimSpace(post.Category))

	candidates := []PostRecord{}
	if locale == "" {
		items, err := getPosts(map[string]string{
			"page":    "1",
			"perPage": "120",
			"filter":  fmt.Sprintf("published = true && id != \"%s\"", escapeFilter(post.ID)),
			"sort":    "-published_at",
		})
		if err != nil {
			items, err = getPosts(map[string]string{
				"page":    "1",
				"perPage": "120",
				"filter":  fmt.Sprintf("published = true && id != \"%s\"", escapeFilter(post.ID)),
				"sort":    "-date",
			})
			if err != nil {
				return nil
			}
		}
		candidates = items.Items
	} else {
		items, err := getPostTranslations(map[string]string{
			"page":    "1",
			"perPage": "120",
			"filter":  fmt.Sprintf("published = true && locale = \"%s\" && id != \"%s\"", escapeFilter(locale), escapeFilter(post.ID)),
			"sort":    "-published_at",
		})
		if err != nil {
			return nil
		}
		candidates = make([]PostRecord, 0, len(items.Items))
		for _, item := range items.Items {
			candidates = append(candidates, translationToPost(item))
		}
	}

	type scoredPost struct {
		post  PostRecord
		score int
		date  time.Time
	}

	scored := make([]scoredPost, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Slug) == "" || candidate.Slug == post.Slug {
			continue
		}

		score := 0
		if targetCategory != "" && strings.ToLower(strings.TrimSpace(candidate.Category)) == targetCategory {
			score += 2
		}

		for _, tag := range parseTags(candidate.Tags) {
			if _, ok := targetTags[strings.ToLower(tag)]; ok {
				score++
			}
		}

		if score == 0 {
			continue
		}

		scored = append(scored, scoredPost{
			post:  candidate,
			score: score,
			date:  postPublishedTime(candidate),
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].date.Equal(scored[j].date) {
			return scored[i].date.After(scored[j].date)
		}
		return scored[i].post.Slug < scored[j].post.Slug
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	related := make([]PostRecord, 0, len(scored))
	for _, item := range scored {
		related = append(related, item.post)
	}
	return related
}

func postPublishedTime(post PostRecord) time.Time {
	value := strings.TrimSpace(post.PublishedAt)
	if value == "" {
		value = strings.TrimSpace(post.Date)
	}
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed
	}
	return time.Time{}
}

func getMediaByID(id string) *MediaRecord {
	media, err := fetchRecord[MediaRecord](fmt.Sprintf("%s/api/collections/media/records/%s", pbURL, id))
	if err != nil {
		return nil
	}
	return &media
}

func getMediaByPath(path string) *MediaRecord {
	data, err := fetchList[MediaRecord](fmt.Sprintf("%s/api/collections/media/records", pbURL), map[string]string{
		"page":    "1",
		"perPage": "1",
		"filter":  fmt.Sprintf("path = \"%s\"", escapeFilter(path)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func collectTags() []string {
	return collectUnique("tags")
}

func collectCategories() []string {
	return collectUnique("category")
}

func collectUnique(field string) []string {
	values := map[string]struct{}{}
	page := 1
	perPage := 200
	for {
		data, err := getPosts(map[string]string{
			"page":    strconv.Itoa(page),
			"perPage": strconv.Itoa(perPage),
			"filter":  "published = true",
			"sort":    "-published_at",
		})
		if err != nil {
			data, err = getPosts(map[string]string{
				"page":    strconv.Itoa(page),
				"perPage": strconv.Itoa(perPage),
				"filter":  "published = true",
				"sort":    "-date",
			})
			if err != nil {
				break
			}
		}
		for _, post := range data.Items {
			var value string
			switch field {
			case "tags":
				for _, tag := range parseTags(post.Tags) {
					values[tag] = struct{}{}
				}
				continue
			case "category":
				value = strings.TrimSpace(post.Category)
			}
			if value != "" {
				values[value] = struct{}{}
			}
		}
		if len(data.Items) < perPage {
			break
		}
		page++
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func translationToPost(item PostTranslationRecord) PostRecord {
	return PostRecord{
		ID:          item.ID,
		Title:       item.Title,
		Slug:        item.Slug,
		Body:        item.Body,
		Excerpt:     item.Excerpt,
		Tags:        item.Tags,
		Category:    item.Category,
		Published:   item.Published,
		PublishedAt: item.PublishedAt,
		Date:        item.PublishedAt,
	}
}

func escapeFilter(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	return replacer.Replace(value)
}

func decodePathSegment(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}
