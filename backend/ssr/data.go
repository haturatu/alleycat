package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func getPosts(params map[string]string) (PBList[PostRecord], error) {
	return fetchList[PostRecord](fmt.Sprintf("%s/api/collections/posts/records", pbURL), params)
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
	data, err := fetchList[PostRecord](fmt.Sprintf("%s/api/collections/posts/records", pbURL), map[string]string{
		"perPage": "1",
		"filter":  fmt.Sprintf("slug = \"%s\" && published = true", escapeFilter(slug)),
	})
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func getAdjacentPosts(post *PostRecord) (newer *PostRecord, older *PostRecord) {
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
