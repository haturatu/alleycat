package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const criticalBaseStyles = `<style>
    *,*::before,*::after{box-sizing:border-box}
    html,body{margin:0;padding:0}
    body{line-height:1.6;text-rendering:optimizeLegibility}
    img{max-width:100%;height:auto;display:block}
    .navbar{display:flex;justify-content:space-between;align-items:center}
    main{max-width:1100px;margin:0 auto;padding:24px 6vw 80px}
    .postList{display:grid;gap:16px}
    </style>`

var commentsScriptTagPattern = regexp.MustCompile(`(?is)^\s*<script\b[^>]*\ssrc\s*=\s*['"]([^'"]+)['"][^>]*>\s*</script>\s*$`)
var headingIDAttrPattern = regexp.MustCompile(`(?is)\sid\s*=\s*(?:"([^"]+)"|'([^']+)')`)
var nonAlnumPattern = regexp.MustCompile(`[^a-z0-9]+`)

type postRenderInput struct {
	path        string
	locale      string
	slug        string
	post        *PostRecord
	translation *PostTranslationRecord
}

func themeStylesheet(themeOverride string) string {
	if activePublicDir == publicDir {
		return "/styles.css"
	}
	theme := strings.TrimSpace(themeOverride)
	if theme == "" {
		theme = defaultTheme
	}
	return "/themes/" + url.PathEscape(strings.ToLower(theme)) + "/styles.css"
}

func asyncStylesheetTag(href string) string {
	safeHref := escapeHTML(href)
	return fmt.Sprintf("<link rel=\"preload\" href=\"%s\" as=\"style\" onload=\"this.onload=null;this.rel='stylesheet'\" />\n    <noscript><link rel=\"stylesheet\" href=\"%s\" /></noscript>", safeHref, safeHref)
}

func themeFontStylesheet(themeOverride string) string {
	theme := strings.TrimSpace(themeOverride)
	if theme == "" {
		theme = defaultTheme
	}
	switch strings.ToLower(theme) {
	case "wiki":
		return "https://fonts.googleapis.com/css2?family=Source+Serif+4:wght@400;600;700&family=IBM+Plex+Sans:wght@300;400;500;600&display=swap"
	case "ember":
		return "https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,500;9..144,700&family=Manrope:wght@300;400;500;600;700&display=swap"
	case "docs":
		return "https://fonts.googleapis.com/css2?family=Inter:wght@300;400;600;700&family=Space+Mono:wght@400;700&display=swap"
	case "terminal":
		return "https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;500;700&display=swap"
	default:
		return ""
	}
}

func highlightStylesheets(settings SettingsRecord) (string, string) {
	switch strings.ToLower(strings.TrimSpace(settings.HighlightTheme)) {
	case "github":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css"
	case "atom-one-dark", "atom-one-light":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/atom-one-dark.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/atom-one-light.min.css"
	case "tokyo-night-dark", "tokyo-night-light":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/tokyo-night-dark.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/tokyo-night-light.min.css"
	case "solarized-dark", "solarized-light":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/solarized-dark.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/solarized-light.min.css"
	case "monokai":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/monokai.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css"
	case "dracula":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/dracula.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css"
	case "vs":
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/vs.min.css"
	default:
		return "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css",
			"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css"
	}
}

func renderHead(title string, settings SettingsRecord) string {
	return renderHeadWithExtras(title, settings, "")
}

func renderHeadWithExtras(title string, settings SettingsRecord, extraHead string) string {
	pageTitle := escapeHTML(title) + " - " + escapeHTML(settings.SiteName)
	styles := themeStylesheet(settings.Theme)
	themeStyles, splitCriticalStyles := splitThemeStylesheetForHead(styles)
	fontStylesheet := themeFontStylesheet(settings.Theme)
	commonContentStyles := criticalBaseStyles + splitCriticalStyles + `
    <style>
    .body pre,
    .body code {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    }
    .body :not(pre) > code {
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
      padding: 0.12rem 0.38rem;
      border-radius: 6px;
      background: rgba(127, 127, 127, 0.12);
      font-size: 0.92em;
    }
    .body pre {
      margin: 1rem 0;
      border: 1px solid rgba(127, 127, 127, 0.24);
      border-radius: 10px;
      background: rgba(127, 127, 127, 0.08);
      overflow-x: auto;
    }
    .body pre code {
      display: block;
      padding: 0;
      background: transparent;
      white-space: pre;
      overflow-wrap: normal;
      word-break: normal;
      line-height: 1.65;
    }
    .post-related {
      margin-top: 1.5rem;
      padding: 1rem;
      border: 1px solid rgba(127, 127, 127, 0.24);
      border-radius: 10px;
    }
    .post-related-list {
      margin: 0;
      padding-left: 1.25rem;
      display: grid;
      gap: 0.5rem;
    }
    .post-related-list li p {
      margin: 0.2rem 0 0;
      opacity: 0.75;
      font-size: 0.9rem;
    }
    .post-comments {
      margin-top: 1.5rem;
      padding-top: 0.5rem;
    }
    .post-toc {
      margin: 1rem 0 1.2rem;
      padding: 0.85rem 1rem;
      border: 1px solid rgba(127, 127, 127, 0.24);
      border-radius: 10px;
    }
    .post-toc h2 {
      margin: 0 0 0.5rem;
      font-size: 1rem;
    }
    .post-toc ul {
      margin: 0;
      padding-left: 1.15rem;
      display: grid;
      gap: 0.3rem;
    }
    .post-toc li[data-level="2"] {
      opacity: 0.96;
    }
    .post-toc li[data-level="3"] {
      margin-left: 0.75rem;
      opacity: 0.9;
    }
    .post-toc li[data-level="4"] {
      margin-left: 1.5rem;
      opacity: 0.85;
    }
    .post-toc li[data-level="5"] {
      margin-left: 2.25rem;
      opacity: 0.8;
    }
    .post-toc li[data-level="6"] {
      margin-left: 3rem;
      opacity: 0.75;
    }
    </style>`
	fontStyles := ""
	if fontStylesheet != "" {
		fontStyles = "<link rel=\"preconnect\" href=\"https://fonts.googleapis.com\" />\n    <link rel=\"preconnect\" href=\"https://fonts.gstatic.com\" crossorigin />\n    " + asyncStylesheetTag(fontStylesheet)
	}
	metaDesc := escapeHTML(settings.Description)
	analytics := ""
	if settings.EnableAnalytics && settings.AnalyticsURL != "" && settings.AnalyticsSiteID != "" {
		analytics = fmt.Sprintf("<script defer src=\"%s\" data-website-id=\"%s\"></script>", escapeHTML(settings.AnalyticsURL), escapeHTML(settings.AnalyticsSiteID))
	}
	ads := ""
	if settings.EnableAds && settings.AdsClient != "" {
		ads = fmt.Sprintf("<script async src=\"https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=%s\" crossorigin=\"anonymous\"></script>", escapeHTML(settings.AdsClient))
	}
	codeHighlight := ""
	if settings.EnableCodeHighlight {
		highlightDarkCSS, highlightLightCSS := highlightStylesheets(settings)
		codeHighlight = fmt.Sprintf("<link rel=\"preconnect\" href=\"https://cdnjs.cloudflare.com\" crossorigin />\n    <link rel=\"preload\" href=\"%s\" as=\"style\" />\n    <link rel=\"preload\" href=\"%s\" as=\"style\" />\n    <link id=\"hljs-theme-link\" rel=\"stylesheet\" href=\"%s\" data-theme-dark=\"%s\" data-theme-light=\"%s\" />\n    <script defer src=\"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js\"></script>\n    <script>window.addEventListener('DOMContentLoaded',()=>{if(window.hljs){window.hljs.highlightAll();}});</script>", highlightDarkCSS, highlightLightCSS, highlightDarkCSS, highlightDarkCSS, highlightLightCSS)
	}
	if strings.TrimSpace(extraHead) != "" {
		extraHead = strings.TrimSpace(extraHead)
	}
	feedAlternates := renderFeedAlternates(settings)

	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s</title>
    <meta name="supported-color-schemes" content="light dark" />
    <meta name="theme-color" content="hsl(220, 20%%, 100%%)" media="(prefers-color-scheme: light)" />
    <meta name="theme-color" content="hsl(220, 20%%, 10%%)" media="(prefers-color-scheme: dark)" />
    %s
    %s
    %s
    %s
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon.png" />
    <meta name="description" content="%s" />
    <meta name="robots" content="max-image-preview:large" />
    %s
    %s
    %s
    %s
  </head>
  <body>`, escapeHTML(settings.SiteLanguage), pageTitle, themeStyles, fontStyles, commonContentStyles, feedAlternates, metaDesc, analytics, ads, codeHighlight, extraHead)
}

type postMetaInput struct {
	Path        string
	Locale      string
	Title       string
	Description string
	PublishedAt string
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

	parts := []string{
		fmt.Sprintf(`<link rel="canonical" href="%s" />`, escapeHTML(canonicalURL)),
		`<meta property="og:type" content="article" />`,
		fmt.Sprintf(`<meta property="og:title" content="%s" />`, escapeHTML(strings.TrimSpace(input.Title))),
		fmt.Sprintf(`<meta property="og:description" content="%s" />`, escapeHTML(description)),
		fmt.Sprintf(`<meta property="og:url" content="%s" />`, escapeHTML(canonicalURL)),
		fmt.Sprintf(`<meta property="og:site_name" content="%s" />`, escapeHTML(settings.SiteName)),
		fmt.Sprintf(`<meta name="twitter:card" content="%s" />`, func() string {
			if settings.EnableOGPImageGeneration {
				return "summary_large_image"
			}
			return "summary"
		}()),
		fmt.Sprintf(`<meta name="twitter:title" content="%s" />`, escapeHTML(strings.TrimSpace(input.Title))),
		fmt.Sprintf(`<meta name="twitter:description" content="%s" />`, escapeHTML(description)),
	}

	if locale := normalizeLocale(input.Locale); locale != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="og:locale" content="%s" />`, escapeHTML(strings.ReplaceAll(locale, "-", "_"))))
	}
	if publishedAt := strings.TrimSpace(input.PublishedAt); publishedAt != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="article:published_time" content="%s" />`, escapeHTML(publishedAt)))
	}
	if settings.EnableOGPImageGeneration {
		imageLocale := extractLocaleFromPostPath(input.Path)
		imageURL := buildAbsoluteSiteURL(settings, postOGImageRoute(imageLocale, extractSlugFromPostPath(input.Path)))
		if imageURL == "" {
			imageURL = postOGImageRoute(imageLocale, extractSlugFromPostPath(input.Path))
		}
		parts = append(parts,
			fmt.Sprintf(`<meta property="og:image" content="%s" />`, escapeHTML(imageURL)),
			fmt.Sprintf(`<meta property="og:image:width" content="%d" />`, postOGImageWidth),
			fmt.Sprintf(`<meta property="og:image:height" content="%d" />`, postOGImageHeight),
			fmt.Sprintf(`<meta name="twitter:image" content="%s" />`, escapeHTML(imageURL)),
			fmt.Sprintf(`<meta name="twitter:image:alt" content="%s" />`, escapeHTML(strings.TrimSpace(input.Title))),
		)
	}

	return strings.Join(parts, "\n    ")
}

func renderNav(menu []PageRecord, settings SettingsRecord) string {
	links := strings.Builder{}
	for _, page := range menu {
		label := page.MenuTitle
		if strings.TrimSpace(label) == "" {
			label = page.Title
		}
		links.WriteString(fmt.Sprintf(`        <li><a href="%s">%s</a></li>`, escapeHTML(page.URL), escapeHTML(label)))
	}
	return fmt.Sprintf(`<nav class="navbar">
      <a href="/" class="navbar-home">
        <strong>%s</strong>
      </a>

      <ul class="navbar-links">
        <li><a href="/archive/">Archive</a></li>
        %s
	        <li>
	          <script>
	            (() => {
	              const root = document.documentElement;
	              const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
	              let theme = localStorage.getItem("theme") || (prefersDark ? "dark" : "light");
	              const applyTheme = (nextTheme) => {
	                root.dataset.theme = nextTheme;
	                const hljsThemeLink = document.getElementById("hljs-theme-link");
	                if (hljsThemeLink) {
	                  const darkHref = hljsThemeLink.getAttribute("data-theme-dark");
	                  const lightHref = hljsThemeLink.getAttribute("data-theme-light");
	                  if (nextTheme === "dark" && darkHref) {
	                    hljsThemeLink.setAttribute("href", darkHref);
	                  }
	                  if (nextTheme === "light" && lightHref) {
	                    hljsThemeLink.setAttribute("href", lightHref);
	                  }
	                }
	              };
	              applyTheme(theme);
	              window.changeTheme = () => {
	                theme = theme === "dark" ? "light" : "dark";
	                localStorage.setItem("theme", theme);
                applyTheme(theme);
              };
            })();
          </script>
          <button class="button" onclick="changeTheme()">
            <span class="icon">◐</span>
          </button>
        </li>
      </ul>
    </nav>`, escapeHTML(settings.SiteName), links.String())
}

func renderFeedAlternates(settings SettingsRecord) string {
	links := make([]string, 0, 2)
	title := escapeHTML(settings.SiteName)
	if settings.EnableFeedXML {
		links = append(links, fmt.Sprintf(`<link rel="alternate" href="/feed.xml" type="application/atom+xml" title="%s" />`, title))
	}
	if settings.EnableFeedJSON {
		links = append(links, fmt.Sprintf(`<link rel="alternate" href="/feed.json" type="application/json" title="%s" />`, title))
	}
	return strings.Join(links, "\n    ")
}

func renderFeedLinkList(settings SettingsRecord) string {
	links := make([]string, 0, 2)
	if settings.EnableFeedXML {
		links = append(links, `<a href="/feed.xml">Atom</a>`)
	}
	if settings.EnableFeedJSON {
		links = append(links, `<a href="/feed.json">JSON</a>`)
	}
	if len(links) == 0 {
		return ""
	}
	return `<p>RSS: ` + strings.Join(links, ", ") + `</p>`
}

func renderFooter(settings SettingsRecord) string {
	if strings.TrimSpace(settings.FooterHTML) == "" {
		return "\n  </body>\n</html>"
	}
	return fmt.Sprintf(`<footer class="footer">%s</footer>
  </body>
</html>`, settings.FooterHTML)
}

func renderPagination(base string, pageNumber, totalPages int, query string) string {
	if totalPages <= 1 {
		return ""
	}
	query = strings.TrimSpace(query)
	querySuffix := ""
	if query != "" {
		querySuffix = "?q=" + url.QueryEscape(query)
	}
	prev := ""
	next := ""
	if pageNumber > 1 {
		prevPage := pageNumber - 1
		link := fmt.Sprintf("%s/", base)
		if prevPage != 1 {
			link = fmt.Sprintf("%s/%d/", base, prevPage)
		}
		link += querySuffix
		prev = fmt.Sprintf(`<li class="pagination-prev"><a href="%s" rel="prev"><span>Previous</span><strong>%d</strong></a></li>`, link, prevPage)
	}
	if pageNumber < totalPages {
		nextPage := pageNumber + 1
		link := fmt.Sprintf("%s/%d/%s", base, nextPage, querySuffix)
		next = fmt.Sprintf(`<li class="pagination-next"><a href="%s" rel="next"><span>Next</span><strong>%d</strong></a></li>`, link, nextPage)
	}
	return fmt.Sprintf(`<nav class="page-pagination pagination">
    <ul>
      %s
      %s
    </ul>
  </nav>`, prev, next)
}

func renderTagsNav(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	items := strings.Builder{}
	for _, tag := range tags {
		items.WriteString(fmt.Sprintf(`<li><a href="/archive/%s/" class="badge">%s</a></li>`, url.PathEscape(tag), escapeHTML(tag)))
	}
	return fmt.Sprintf(`<nav class="page-navigation">
    <h2>tags:</h2>
    <ul class="page-navigation-tags">
      %s
    </ul>
  </nav>`, items.String())
}

func renderCategoriesNav(categories []string) string {
	if len(categories) == 0 {
		return ""
	}
	items := strings.Builder{}
	for _, category := range categories {
		items.WriteString(fmt.Sprintf(`<li><a href="/archive/category/%s/" class="badge">%s</a></li>`, url.PathEscape(category), escapeHTML(category)))
	}
	return fmt.Sprintf(`<nav class="page-navigation">
    <h2>categories:</h2>
    <ul class="page-navigation-tags">
      %s
    </ul>
  </nav>`, items.String())
}

func renderSearchForm(actionPath, query string) string {
	safeAction := escapeHTML(actionPath)
	safeQuery := escapeHTML(strings.TrimSpace(query))
	clearHTML := ""
	if safeQuery != "" {
		clearHTML = fmt.Sprintf(`<a class="search-clear" href="%s">Clear</a>`, safeAction)
	}
	return fmt.Sprintf(`<div class="search" id="search">
    <form class="search-form" action="%s" method="get">
      <input class="search-input" type="search" name="q" value="%s" placeholder="Search posts..." aria-label="Search posts" />
      <button class="search-submit" type="submit">Search</button>
      %s
    </form>
  </div>`, safeAction, safeQuery, clearHTML)
}

func renderPostTags(tags []string, show bool) string {
	if !show || len(tags) == 0 {
		return ""
	}
	items := strings.Builder{}
	for _, tag := range tags {
		items.WriteString(fmt.Sprintf(`<a class="badge" href="/archive/%s/">%s</a>`, url.PathEscape(tag), escapeHTML(tag)))
	}
	return fmt.Sprintf(`<div class="post-tags">%s</div>`, items.String())
}

func renderPostList(items []PostRecord, showTags bool, excerptLength int) string {
	list := strings.Builder{}
	for _, post := range items {
		body := post.Body
		if body == "" {
			body = post.Content
		}
		excerpt := post.Excerpt
		if strings.TrimSpace(excerpt) == "" {
			length := excerptLength
			if length <= 0 {
				length = 160
			}
			excerpt = buildExcerpt(body, length)
		}
		tags := parseTags(post.Tags)
		tagsHTML := renderPostTags(tags, showTags)
		date := post.PublishedAt
		if date == "" {
			date = post.Date
		}
		postDetails := ""
		if date != "" || tagsHTML != "" {
			postDetails = fmt.Sprintf(`<div class="post-details">
              %s
              <p>%d min</p>
              %s
            </div>`, func() string {
				if date == "" {
					return ""
				}
				return fmt.Sprintf(`<p><time datetime="%s">%s</time></p>`, escapeHTML(date), formatDate(date))
			}(), calcReadTime(body), tagsHTML)
		}

		list.WriteString(fmt.Sprintf(`<article class="post">
          <header class="post-header">
            <h2 class="post-title">
              <a href="/posts/%s/">%s</a>
            </h2>
            %s
          </header>
          <div class="post-excerpt body">%s</div>
          <a href="/posts/%s/" class="post-link">Read →</a>
        </article>`, escapeHTML(post.Slug), escapeHTML(defaultString(post.Title, post.Slug)), postDetails, excerpt, escapeHTML(post.Slug)))
	}
	return fmt.Sprintf(`<section class="postList">
    %s
  </section>`, list.String())
}

func renderHome(settings SettingsRecord) string {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return renderHomeFromSnapshot(ctx, settings)
	}
	var menu []PageRecord
	var posts PBList[PostRecord]
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		menu = getPagesMenu()
	}()
	go func() {
		defer wg.Done()
		var err error
		posts, err = getPosts(map[string]string{
			"page":    "1",
			"perPage": strconv.Itoa(settings.HomePageSize),
			"filter":  "published = true",
			"sort":    "-published_at",
		})
		if err != nil {
			posts, _ = getPosts(map[string]string{
				"page":    "1",
				"perPage": strconv.Itoa(settings.HomePageSize),
				"filter":  "published = true",
				"sort":    "-date",
			})
		}
	}()
	wg.Wait()
	items := posts.Items

	topImage := strings.TrimSpace(settings.HomeTopImage)
	imageHTML := ""
	if topImage != "" {
		imageHTML = fmt.Sprintf(`<img src="%s" alt="%s" class="top-image" />`, escapeHTML(topImage), escapeHTML(settings.HomeTopImageAlt))
	}

	return renderHead("Home", settings) +
		renderNav(menu, settings) +
		fmt.Sprintf(`<main class="body-home">
      <header class="page-header">
        %s
        <h1 class="page-title">%s</h1>
      </header>
      %s
      <hr>
      <p>More posts can be found in <a href="/archive/">the archive</a>.</p>
    </main>`, imageHTML, escapeHTML(settings.WelcomeText), renderPostList(items, settings.ShowTags, settings.ExcerptLength)) +
		renderFooter(settings)
}

func renderArchive(path, query string, settings SettingsRecord) string {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		return renderArchiveFromSnapshot(ctx, path, query, settings)
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	pageNumber := 1
	basePath := "/archive"
	var filter string
	var title string
	showTagsNav := false
	showCategoriesNav := false

	if len(parts) >= 1 && parts[0] == "archive" {
		if len(parts) >= 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				pageNumber = n
			} else if parts[1] == "category" && len(parts) >= 3 {
				category := decodePathSegment(parts[2])
				title = "category: " + category
				filter = fmt.Sprintf("published = true && category = \"%s\"", escapeFilter(category))
				basePath = "/archive/category/" + url.PathEscape(category)
				if len(parts) >= 4 {
					if n, err := strconv.Atoi(parts[3]); err == nil && n > 0 {
						pageNumber = n
					}
				}
			} else {
				tag := decodePathSegment(parts[1])
				title = "tag: " + tag
				filter = fmt.Sprintf("published = true && tags ~ \"%s\"", escapeFilter(tag))
				basePath = "/archive/" + url.PathEscape(tag)
				if len(parts) >= 3 {
					if n, err := strconv.Atoi(parts[2]); err == nil && n > 0 {
						pageNumber = n
					}
				}
			}
		}
	}
	if filter == "" {
		filter = "published = true"
		title = "Archive"
		showTagsNav = settings.ShowArchiveTags && settings.ShowTags && pageNumber == 1
		showCategoriesNav = settings.ShowCategories && pageNumber == 1
	}
	searchQuery := strings.TrimSpace(query)
	if searchQuery != "" {
		searchFilter := escapeFilter(searchQuery)
		filter = fmt.Sprintf(`%s && (title ~ "%s" || slug ~ "%s" || tags ~ "%s" || excerpt ~ "%s" || body ~ "%s")`,
			filter, searchFilter, searchFilter, searchFilter, searchFilter, searchFilter)
	}

	var menu []PageRecord
	var posts PBList[PostRecord]
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		menu = getPagesMenu()
	}()
	go func() {
		defer wg.Done()
		var err error
		posts, err = getPosts(map[string]string{
			"page":    strconv.Itoa(pageNumber),
			"perPage": strconv.Itoa(settings.ArchivePageSize),
			"filter":  filter,
			"sort":    "-published_at",
		})
		if err != nil {
			posts, _ = getPosts(map[string]string{
				"page":    strconv.Itoa(pageNumber),
				"perPage": strconv.Itoa(settings.ArchivePageSize),
				"filter":  filter,
				"sort":    "-date",
			})
		}
	}()
	wg.Wait()
	pagination := renderPagination(basePath, pageNumber, posts.TotalPages, searchQuery)
	searchHTML := ""
	if settings.ShowArchiveSearch {
		searchAction := basePath + "/"
		searchHTML = renderSearchForm(searchAction, searchQuery)
	}

	tagsNav := ""
	if showTagsNav {
		tagsNav = renderTagsNav(collectTags())
	}
	categoriesNav := ""
	if showCategoriesNav {
		categoriesNav = renderCategoriesNav(collectCategories())
	}
	feedLinks := renderFeedLinkList(settings)

	return renderHead(title, settings) +
		renderNav(menu, settings) +
		fmt.Sprintf(`<main class="body-tag">
      <header class="page-header">
        <h1 class="page-title">%s</h1>
        %s
        %s
      </header>
      %s
      %s
      %s
      %s
    </main>`, escapeHTML(title), feedLinks, searchHTML, renderPostList(posts.Items, settings.ShowTags, settings.ExcerptLength), pagination, tagsNav, categoriesNav) +
		renderFooter(settings)
}

func renderHomeFromSnapshot(ctx *snapshotBuildContext, settings SettingsRecord) string {
	limit := settings.HomePageSize
	if limit <= 0 {
		limit = 3
	}
	items := append([]PostRecord(nil), ctx.publishedPosts...)
	if len(items) > limit {
		items = items[:limit]
	}

	topImage := strings.TrimSpace(settings.HomeTopImage)
	imageHTML := ""
	if topImage != "" {
		imageHTML = fmt.Sprintf(`<img src="%s" alt="%s" class="top-image" />`, escapeHTML(topImage), escapeHTML(settings.HomeTopImageAlt))
	}

	return renderHead("Home", settings) +
		renderNav(ctx.menu, settings) +
		fmt.Sprintf(`<main class="body-home">
      <header class="page-header">
        %s
        <h1 class="page-title">%s</h1>
      </header>
      %s
      <hr>
      <p>More posts can be found in <a href="/archive/">the archive</a>.</p>
    </main>`, imageHTML, escapeHTML(settings.WelcomeText), renderPostList(items, settings.ShowTags, settings.ExcerptLength)) +
		renderFooter(settings)
}

func renderArchiveFromSnapshot(ctx *snapshotBuildContext, path, query string, settings SettingsRecord) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	pageNumber := 1
	basePath := "/archive"
	title := "Archive"
	showTagsNav := false
	showCategoriesNav := false
	listing := ctx.archiveIndex["/archive/"]

	if len(parts) >= 1 && parts[0] == "archive" {
		if len(parts) >= 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				pageNumber = n
			} else if parts[1] == "category" && len(parts) >= 3 {
				category := decodePathSegment(parts[2])
				title = "category: " + category
				basePath = "/archive/category/" + url.PathEscape(category)
				listing = ctx.archiveIndex["/archive/category/"+url.PathEscape(category)+"/"]
				if len(parts) >= 4 {
					if n, err := strconv.Atoi(parts[3]); err == nil && n > 0 {
						pageNumber = n
					}
				}
			} else {
				tag := decodePathSegment(parts[1])
				title = "tag: " + tag
				basePath = "/archive/" + url.PathEscape(tag)
				listing = ctx.archiveIndex["/archive/"+url.PathEscape(tag)+"/"]
				if len(parts) >= 3 {
					if n, err := strconv.Atoi(parts[2]); err == nil && n > 0 {
						pageNumber = n
					}
				}
			}
		}
	}
	if title == "Archive" {
		showTagsNav = settings.ShowArchiveTags && settings.ShowTags && pageNumber == 1
		showCategoriesNav = settings.ShowCategories && pageNumber == 1
	}

	items := append([]PostRecord(nil), listing.posts...)
	searchQuery := strings.TrimSpace(query)
	if searchQuery != "" {
		filtered := make([]PostRecord, 0, len(items))
		for _, item := range items {
			if snapshotPostMatchesQuery(item, searchQuery) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	posts := paginateSnapshotPosts(items, strconv.Itoa(pageNumber), strconv.Itoa(settings.ArchivePageSize))
	pagination := renderPagination(basePath, pageNumber, posts.TotalPages, searchQuery)
	searchHTML := ""
	if settings.ShowArchiveSearch {
		searchHTML = renderSearchForm(basePath+"/", searchQuery)
	}
	tagsNav := ""
	if showTagsNav {
		tagsNav = renderTagsNav(ctx.tags)
	}
	categoriesNav := ""
	if showCategoriesNav {
		categoriesNav = renderCategoriesNav(ctx.categories)
	}
	feedLinks := renderFeedLinkList(settings)

	return renderHead(title, settings) +
		renderNav(ctx.menu, settings) +
		fmt.Sprintf(`<main class="body-tag">
      <header class="page-header">
        <h1 class="page-title">%s</h1>
        %s
        %s
      </header>
      %s
      %s
      %s
      %s
    </main>`, escapeHTML(title), feedLinks, searchHTML, renderPostList(posts.Items, settings.ShowTags, settings.ExcerptLength), pagination, tagsNav, categoriesNav) +
		renderFooter(settings)
}

func prefetchPostRenderInput(path string) (*postRenderInput, bool) {
	locale, slug, ok := resolvePostPath(path)
	if !ok {
		return nil, false
	}
	if locale == "" {
		post := getPostBySlugInLocale(slug, "")
		if post == nil {
			return nil, false
		}
		return &postRenderInput{
			path:   path,
			locale: "",
			slug:   slug,
			post:   post,
		}, true
	}
	translation := getPostTranslationBySlugLocale(slug, locale)
	if translation == nil {
		return nil, false
	}
	post := translationToPost(*translation)
	return &postRenderInput{
		path:        path,
		locale:      locale,
		slug:        slug,
		post:        &post,
		translation: translation,
	}, true
}

func renderPostFromInput(input *postRenderInput, settings SettingsRecord) (string, bool) {
	if input == nil || input.post == nil {
		return renderNotFound(settings), false
	}
	locale := input.locale
	sourceLocale := normalizeLocale(settings.TranslationSourceLocale)
	if sourceLocale == "" {
		sourceLocale = normalizeLocale(settings.SiteLanguage)
	}
	if sourceLocale == "" {
		sourceLocale = "ja"
	}
	currentLocale := sourceLocale
	post := input.post
	var sourcePost *PostRecord
	translations := []PostTranslationRecord{}

	if locale != "" {
		currentLocale = normalizeLocale(locale)
		translation := input.translation
		if translation == nil {
			return renderNotFound(settings), false
		}
		sourceID := translation.SourcePost
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			sourcePost = getPostByID(sourceID)
		}()
		go func() {
			defer wg.Done()
			translations = filterTranslationsByEnabledLocales(getPostTranslationsBySource(sourceID), settings)
		}()
		wg.Wait()
		if sourcePost == nil {
			sourcePost = post
		}
	} else {
		sourcePost = post
	}

	var menu []PageRecord
	var newer *PostRecord
	var older *PostRecord
	related := []PostRecord{}
	var wg sync.WaitGroup
	if locale == "" {
		wg.Add(1)
		go func(sourceID string) {
			defer wg.Done()
			translations = filterTranslationsByEnabledLocales(getPostTranslationsBySource(sourceID), settings)
		}(post.ID)
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		menu = getPagesMenu()
	}()
	go func() {
		defer wg.Done()
		newer, older = getAdjacentPostsInLocale(post, locale)
	}()
	if settings.ShowRelatedPosts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			related = getRelatedPostsInLocale(post, locale, 4)
		}()
	}
	wg.Wait()

	body := post.Body
	if body == "" {
		body = post.Content
	}
	body = rewriteMediaURLs(body)
	body, tocHTML := buildTOC(body, settings.ShowToc)
	date := post.PublishedAt
	if date == "" {
		date = post.Date
	}
	categoryHTML := ""
	if settings.ShowCategories && strings.TrimSpace(post.Category) != "" {
		categoryHTML = fmt.Sprintf(`<p>%s</p>`, escapeHTML(post.Category))
	}
	postTags := renderPostTags(parseTags(post.Tags), settings.ShowTags)
	languageHTML := renderLanguageLinks(sourceLocale, currentLocale, sourcePost, translations)
	postPathPrefix := "/posts/"
	if locale != "" {
		postPathPrefix = "/" + locale + "/posts/"
	}
	navHTML := ""
	if newer != nil || older != nil {
		olderHTML := ""
		newerHTML := ""
		if older != nil {
			olderHTML = fmt.Sprintf(`<li class="pagination-prev"><a href="%s%s/" rel="prev"><span>← Older post</span><strong>%s</strong></a></li>`, postPathPrefix, url.PathEscape(older.Slug), escapeHTML(defaultString(older.Title, "Post")))
		}
		if newer != nil {
			newerHTML = fmt.Sprintf(`<li class="pagination-next"><a href="%s%s/" rel="next"><span>Newer post →</span><strong>%s</strong></a></li>`, postPathPrefix, url.PathEscape(newer.Slug), escapeHTML(defaultString(newer.Title, "Post")))
		}
		navHTML = fmt.Sprintf(`<nav class="page-pagination pagination post-pagination">
      <ul>
        %s
        %s
      </ul>
    </nav>`, olderHTML, newerHTML)
	}
	relatedHTML := ""
	if settings.ShowRelatedPosts {
		relatedHTML = renderRelatedPosts(related, postPathPrefix)
	}
	commentsHTML := renderCommentsSection(settings)
	excerpt := strings.TrimSpace(post.Excerpt)
	if excerpt == "" {
		excerpt = buildExcerpt(body, settings.ExcerptLength)
	}
	postPath := postPathPrefix + strings.TrimSpace(post.Slug) + "/"
	headExtras := renderPostMetaTags(postMetaInput{
		Path:        postPath,
		Locale:      currentLocale,
		Title:       defaultString(post.Title, "Post"),
		Description: excerpt,
		PublishedAt: date,
	}, settings)

	return renderHeadWithExtras(defaultString(post.Title, "Post"), settings, headExtras) +
		renderNav(menu, settings) +
		fmt.Sprintf(`<main class="body-post">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">%s</h1>
          <div class="post-details">
            %s
            <p>%d min</p>
            %s
            %s
            %s
          </div>
        </header>
        %s
        <div class="post-body body">%s</div>
      </article>
      %s
      %s
      %s
    </main>`, escapeHTML(post.Title), func() string {
			if date == "" {
				return ""
			}
			return fmt.Sprintf(`<p><time datetime="%s">%s</time></p>`, escapeHTML(date), formatDate(date))
		}(), calcReadTime(body), categoryHTML, postTags, languageHTML, tocHTML, body, commentsHTML, relatedHTML, navHTML) +
		renderFooter(settings), true
}

func buildTOC(body string, enabled bool) (string, string) {
	if !enabled || strings.TrimSpace(body) == "" {
		return body, ""
	}

	type tocItem struct {
		level int
		id    string
		text  string
	}

	seen := map[string]int{}
	items := []tocItem{}
	updated := headingRe.ReplaceAllStringFunc(body, func(match string) string {
		parts := headingRe.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		levelRaw := parts[1]
		level := 2
		if parsed, err := strconv.Atoi(levelRaw); err == nil && parsed >= 1 && parsed <= 6 {
			level = parsed
		}
		attrs := parts[2]
		content := parts[3]
		title := strings.TrimSpace(stripHTML(content))
		if title == "" {
			return match
		}

		anchorID := headingIDFromAttrs(attrs)
		if anchorID == "" {
			baseID := slugifyHeading(title)
			anchorID = uniqueHeadingID(baseID, seen)
			attrs = attrs + ` id="` + anchorID + `"`
		} else {
			seen[anchorID]++
		}

		items = append(items, tocItem{
			level: level,
			id:    anchorID,
			text:  title,
		})

		return fmt.Sprintf("<h%s%s>%s</h%s>", levelRaw, attrs, content, levelRaw)
	})

	if len(items) == 0 {
		return updated, ""
	}

	list := strings.Builder{}
	for _, item := range items {
		list.WriteString(fmt.Sprintf(`<li data-level="%d"><a href="#%s">%s</a></li>`, item.level, escapeHTML(item.id), escapeHTML(item.text)))
	}

	toc := fmt.Sprintf(`<nav class="post-toc" aria-label="Table of contents">
      <h2>Table of contents</h2>
      <ul>
        %s
      </ul>
    </nav>`, list.String())
	return updated, toc
}

func headingIDFromAttrs(attrs string) string {
	parts := headingIDAttrPattern.FindStringSubmatch(attrs)
	if len(parts) < 3 {
		return ""
	}
	if strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(parts[2])
}

func slugifyHeading(text string) string {
	base := strings.ToLower(strings.TrimSpace(text))
	base = nonAlnumPattern.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		return "section"
	}
	return base
}

func uniqueHeadingID(base string, seen map[string]int) string {
	count := seen[base]
	seen[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, count+1)
}

func renderCommentsSection(settings SettingsRecord) string {
	if !settings.EnableComments {
		return ""
	}
	tag := sanitizeCommentsScriptTag(settings.CommentsScriptTag)
	if tag == "" {
		return ""
	}
	return fmt.Sprintf(`<section class="post-comments">%s</section>`, tag)
}

func sanitizeCommentsScriptTag(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	match := commentsScriptTagPattern.FindStringSubmatch(trimmed)
	if len(match) < 2 {
		return ""
	}
	src := strings.TrimSpace(strings.ToLower(match[1]))
	if !strings.HasPrefix(src, "https://utteranc.es/") && !strings.HasPrefix(src, "https://giscus.app/") {
		return ""
	}
	return trimmed
}

func renderRelatedPosts(items []PostRecord, postPathPrefix string) string {
	if len(items) == 0 {
		return ""
	}
	list := strings.Builder{}
	for _, post := range items {
		date := strings.TrimSpace(post.PublishedAt)
		if date == "" {
			date = strings.TrimSpace(post.Date)
		}
		dateHTML := ""
		if date != "" {
			dateHTML = fmt.Sprintf(`<p><time datetime="%s">%s</time></p>`, escapeHTML(date), formatDate(date))
		}
		list.WriteString(fmt.Sprintf(`<li>
          <a href="%s%s/">%s</a>
          %s
        </li>`, postPathPrefix, url.PathEscape(post.Slug), escapeHTML(defaultString(post.Title, "Post")), dateHTML))
	}
	return fmt.Sprintf(`<section class="post-related">
      <h2>Related Posts</h2>
      <ul class="post-related-list">
        %s
      </ul>
    </section>`, list.String())
}

func renderLanguageLinks(sourceLocale, currentLocale string, sourcePost *PostRecord, translations []PostTranslationRecord) string {
	type linkItem struct {
		locale string
		href   string
	}

	items := make([]linkItem, 0, len(translations)+1)
	seen := map[string]struct{}{}

	if sourcePost != nil && strings.TrimSpace(sourcePost.Slug) != "" && sourceLocale != "" {
		items = append(items, linkItem{
			locale: sourceLocale,
			href:   "/posts/" + url.PathEscape(sourcePost.Slug) + "/",
		})
		seen[sourceLocale] = struct{}{}
	}

	for _, t := range translations {
		locale := normalizeLocale(t.Locale)
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}
		slug := strings.TrimSpace(t.Slug)
		if slug == "" {
			continue
		}
		items = append(items, linkItem{
			locale: locale,
			href:   "/" + url.PathEscape(locale) + "/posts/" + url.PathEscape(slug) + "/",
		})
		seen[locale] = struct{}{}
	}

	if len(items) <= 1 {
		return ""
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.locale == currentLocale {
			parts = append(parts, fmt.Sprintf(`<strong>%s</strong>`, escapeHTML(item.locale)))
			continue
		}
		parts = append(parts, fmt.Sprintf(`<a class="badge" href="%s">%s</a>`, escapeHTML(item.href), escapeHTML(item.locale)))
	}

	return fmt.Sprintf(`<div class="post-languages">language: %s</div>`, strings.Join(parts, " "))
}

func renderPageFromRecord(page *PageRecord, settings SettingsRecord) (string, bool) {
	if page == nil {
		return renderNotFound(settings), false
	}
	menu := getPagesMenu()
	body := page.Body
	if body == "" {
		body = page.Content
	}
	body = rewriteMediaURLs(body)

	return renderHead(defaultString(page.Title, "Page"), settings) +
		renderNav(menu, settings) +
		fmt.Sprintf(`<main class="body-tag">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">%s</h1>
        </header>
        <div class="post-body body">%s</div>
      </article>
    </main>`, escapeHTML(page.Title), body) +
		renderFooter(settings), true
}

func renderNotFound(settings SettingsRecord) string {
	menu := getPagesMenu()
	return renderHead("Not Found", settings) +
		renderNav(menu, settings) +
		`<main class="body-post">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">Not Found</h1>
        </header>
        <div class="post-body body">Page not found.</div>
      </article>
    </main>` +
		renderFooter(settings)
}

func resolvePostPath(path string) (locale string, slug string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}
	if parts[0] == "posts" && len(parts) >= 2 {
		return "", parts[1], true
	}
	if len(parts) >= 3 && parts[1] == "posts" {
		return normalizeLocale(parts[0]), parts[2], true
	}
	return "", "", false
}
