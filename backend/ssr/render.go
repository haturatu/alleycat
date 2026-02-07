package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

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

func renderHead(title string, settings SettingsRecord) string {
	pageTitle := escapeHTML(title) + " - " + escapeHTML(settings.SiteName)
	styles := themeStylesheet(settings.Theme)
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
		theme := settings.HighlightTheme
		if theme == "" {
			theme = "github-dark"
		}
		codeHighlight = fmt.Sprintf("<link rel=\"stylesheet\" href=\"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/%s.min.css\" />\n    <script defer src=\"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js\"></script>\n    <script>window.addEventListener('DOMContentLoaded',()=>{if(window.hljs){window.hljs.highlightAll();}});</script>", escapeHTML(theme))
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s</title>
    <meta name="supported-color-schemes" content="light dark" />
    <meta name="theme-color" content="hsl(220, 20%%, 100%%)" media="(prefers-color-scheme: light)" />
    <meta name="theme-color" content="hsl(220, 20%%, 10%%)" media="(prefers-color-scheme: dark)" />
    <link rel="stylesheet" href="%s" />
    <link rel="alternate" href="/feed.xml" type="application/atom+xml" title="%s" />
    <link rel="alternate" href="/feed.json" type="application/json" title="%s" />
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon.png" />
    <meta name="description" content="%s" />
    %s
    %s
    %s
  </head>
  <body>`, escapeHTML(settings.SiteLanguage), pageTitle, styles, escapeHTML(settings.SiteName), escapeHTML(settings.SiteName), metaDesc, analytics, ads, codeHighlight)
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
            let theme = localStorage.getItem("theme") || (window.matchMedia("(prefers-color-scheme: dark)").matches
              ? "dark"
              : "light");
            document.documentElement.dataset.theme = theme;
            function changeTheme() {
              theme = theme === "dark" ? "light" : "dark";
              localStorage.setItem("theme", theme);
              document.documentElement.dataset.theme = theme;
            }
          </script>
          <button class="button" onclick="changeTheme()">
            <span class="icon">◐</span>
          </button>
        </li>
      </ul>
    </nav>`, escapeHTML(settings.SiteName), links.String())
}

func renderFooter(settings SettingsRecord) string {
	if strings.TrimSpace(settings.FooterHTML) == "" {
		return "\n  </body>\n</html>"
	}
	return fmt.Sprintf(`<footer class="footer">%s</footer>
  </body>
</html>`, settings.FooterHTML)
}

func renderPagination(base string, pageNumber, totalPages int) string {
	if totalPages <= 1 {
		return ""
	}
	prev := ""
	next := ""
	if pageNumber > 1 {
		prevPage := pageNumber - 1
		link := fmt.Sprintf("%s/", base)
		if prevPage != 1 {
			link = fmt.Sprintf("%s/%d/", base, prevPage)
		}
		prev = fmt.Sprintf(`<li class="pagination-prev"><a href="%s" rel="prev"><span>Previous</span><strong>%d</strong></a></li>`, link, prevPage)
	}
	if pageNumber < totalPages {
		nextPage := pageNumber + 1
		link := fmt.Sprintf("%s/%d/", base, nextPage)
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

func renderPostList(items []PostRecord, showTags bool) string {
	list := strings.Builder{}
	for _, post := range items {
		body := post.Body
		if body == "" {
			body = post.Content
		}
		excerpt := post.Excerpt
		if strings.TrimSpace(excerpt) == "" {
			excerpt = buildExcerpt(body, 160)
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
	menu := getPagesMenu()
	posts, err := getPosts(map[string]string{
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
    </main>`, imageHTML, escapeHTML(settings.WelcomeText), renderPostList(items, settings.ShowTags)) +
		renderFooter(settings)
}

func renderArchive(path string, settings SettingsRecord) string {
	menu := getPagesMenu()
	parts := strings.Split(strings.Trim(path, "/"), "/")
	pageNumber := 1
	basePath := "/archive"
	var filter string
	var title string
	showTagsNav := false
	showCategoriesNav := false

	if len(parts) >= 1 && parts[0] == "archive" {
		if len(parts) >= 2 {
			if parts[1] == "category" && len(parts) >= 3 {
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

	posts, err := getPosts(map[string]string{
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
	pagination := renderPagination(basePath, pageNumber, posts.TotalPages)
	searchHTML := ""
	if settings.ShowArchiveSearch {
		searchHTML = `<div class="search" id="search"></div>`
	}

	tagsNav := ""
	if showTagsNav {
		tagsNav = renderTagsNav(collectTags())
	}
	categoriesNav := ""
	if showCategoriesNav {
		categoriesNav = renderCategoriesNav(collectCategories())
	}

	return renderHead(title, settings) +
		renderNav(menu, settings) +
		fmt.Sprintf(`<main class="body-tag">
      <header class="page-header">
        <h1 class="page-title">%s</h1>
        <p>RSS: <a href="/feed.xml">Atom</a>, <a href="/feed.json">JSON</a></p>
        %s
      </header>
      %s
      %s
      %s
      %s
    </main>`, escapeHTML(title), searchHTML, renderPostList(posts.Items, settings.ShowTags), pagination, tagsNav, categoriesNav) +
		renderFooter(settings)
}

func renderPost(path string, settings SettingsRecord) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return renderNotFound(settings)
	}
	slug := parts[1]
	post := getPostBySlug(slug)
	if post == nil {
		return renderNotFound(settings)
	}
	menu := getPagesMenu()
	body := post.Body
	if body == "" {
		body = post.Content
	}
	body = rewriteMediaURLs(body)
	date := post.PublishedAt
	if date == "" {
		date = post.Date
	}
	categoryHTML := ""
	if settings.ShowCategories && strings.TrimSpace(post.Category) != "" {
		categoryHTML = fmt.Sprintf(`<p>%s</p>`, escapeHTML(post.Category))
	}
	postTags := renderPostTags(parseTags(post.Tags), settings.ShowTags)

	return renderHead(defaultString(post.Title, "Post"), settings) +
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
          </div>
        </header>
        <div class="post-body body">%s</div>
      </article>
    </main>`, escapeHTML(post.Title), func() string {
			if date == "" {
				return ""
			}
			return fmt.Sprintf(`<p><time datetime="%s">%s</time></p>`, escapeHTML(date), formatDate(date))
		}(), calcReadTime(body), categoryHTML, postTags, body) +
		renderFooter(settings)
}

func renderPage(path string, settings SettingsRecord) string {
	page := getPageByURL(path)
	if page == nil {
		return renderNotFound(settings)
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
		renderFooter(settings)
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
        <div class="post-body body">ページが見つかりませんでした。</div>
      </article>
    </main>` +
		renderFooter(settings)
}
