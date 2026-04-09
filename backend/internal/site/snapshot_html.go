package site

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

func absolutizePrerenderedSnapshotHTML(body []byte, settings SettingsRecord) []byte {
	if len(body) == 0 || normalizeSiteBaseURL(settings.SiteURL) == "" {
		return body
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return body
	}

	changed := false
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode {
			switch node.Data {
			case "link":
				if absolutizeLinkNode(node, settings) {
					changed = true
				}
			case "meta":
				if absolutizeMetaNode(node, settings) {
					changed = true
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	if !changed {
		return body
	}

	var out bytes.Buffer
	if err := html.Render(&out, doc); err != nil {
		return body
	}
	return out.Bytes()
}

func absolutizeLinkNode(node *html.Node, settings SettingsRecord) bool {
	if !hasRelValue(node, "canonical") {
		return false
	}
	return absolutizeAttribute(node, "href", settings)
}

func absolutizeMetaNode(node *html.Node, settings SettingsRecord) bool {
	key := strings.ToLower(strings.TrimSpace(getHTMLAttr(node, "property")))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(getHTMLAttr(node, "name")))
	}
	switch key {
	case "og:url", "og:image", "og:image:url", "og:image:secure_url", "twitter:image":
		return absolutizeAttribute(node, "content", settings)
	default:
		return false
	}
}

func absolutizeAttribute(node *html.Node, key string, settings SettingsRecord) bool {
	for i := range node.Attr {
		if !strings.EqualFold(node.Attr[i].Key, key) {
			continue
		}
		value := strings.TrimSpace(node.Attr[i].Val)
		if !strings.HasPrefix(value, "/") {
			return false
		}
		node.Attr[i].Val = buildAbsoluteSiteURL(settings, value)
		return true
	}
	return false
}

func hasRelValue(node *html.Node, want string) bool {
	rel := strings.ToLower(strings.TrimSpace(getHTMLAttr(node, "rel")))
	if rel == "" {
		return false
	}
	for _, part := range strings.Fields(rel) {
		if part == want {
			return true
		}
	}
	return false
}

func getHTMLAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}
