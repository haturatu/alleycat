package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PBList[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

type PostRecord struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Body        string `json:"body"`
	Content     string `json:"content"`
	Excerpt     string `json:"excerpt"`
	Tags        string `json:"tags"`
	Category    string `json:"category"`
	Published   bool   `json:"published"`
	PublishedAt string `json:"published_at"`
	Date        string `json:"date"`
}

type PageRecord struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	URL         string `json:"url"`
	Body        string `json:"body"`
	Content     string `json:"content"`
	MenuVisible bool   `json:"menuVisible"`
	MenuOrder   int    `json:"menuOrder"`
	MenuTitle   string `json:"menuTitle"`
	Published   bool   `json:"published"`
	PublishedAt string `json:"published_at"`
	Date        string `json:"date"`
}

type SettingsRecord struct {
	ID                 string `json:"id"`
	SiteName           string `json:"site_name"`
	Description        string `json:"description"`
	WelcomeText        string `json:"welcome_text"`
	HomeTopImage       string `json:"home_top_image"`
	HomeTopImageAlt    string `json:"home_top_image_alt"`
	FooterHTML         string `json:"footer_html"`
	SiteURL            string `json:"site_url"`
	SiteLanguage       string `json:"site_language"`
	EnableFeedXML      bool   `json:"enable_feed_xml"`
	EnableFeedJSON     bool   `json:"enable_feed_json"`
	FeedItemsLimit     int    `json:"feed_items_limit"`
	EnableAnalytics    bool   `json:"enable_analytics"`
	AnalyticsURL       string `json:"analytics_url"`
	AnalyticsSiteID    string `json:"analytics_site_id"`
	EnableAds          bool   `json:"enable_ads"`
	AdsClient          string `json:"ads_client"`
	EnableCodeHighlight bool  `json:"enable_code_highlight"`
	HighlightTheme     string `json:"highlight_theme"`
	ArchivePageSize    int    `json:"archive_page_size"`
	HomePageSize       int    `json:"home_page_size"`
	ShowToc            bool   `json:"show_toc"`
	ShowArchiveTags    bool   `json:"show_archive_tags"`
	ShowArchiveSearch  bool   `json:"show_archive_search"`
}

var (
	pbURL      = getEnv("PB_URL", "http://127.0.0.1:8090")
	adminURL   = getEnv("ADMIN_URL", "http://admin:5173")
	publicDir  = getEnv("PUBLIC_DIR", "/public")
	listenAddr = getEnv("LISTEN_ADDR", ":5173")
)

var headingRe = regexp.MustCompile(`(?is)<h([23])([^>]*)>(.*?)</h[23]>`)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/admin/", adminProxy())
	mux.Handle("/", http.HandlerFunc(routeHandler))

	log.Printf("SSR server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if serveStatic(w, r) {
		return
	}

	if path == "/" {
		html := renderHome()
		writeHTML(w, html)
		return
	}

	if path == "/archive" {
		html := renderArchive("/archive/")
		writeHTML(w, html)
		return
	}

	if path == "/feed.json" {
		settings := getSettings()
		if !settings.EnableFeedJSON {
			http.NotFound(w, r)
			return
		}
		writeJSONFeed(w, r, settings)
		return
	}

	if path == "/feed.xml" {
		settings := getSettings()
		if !settings.EnableFeedXML {
			http.NotFound(w, r)
			return
		}
		writeRSSFeed(w, r, settings)
		return
	}

	if strings.HasPrefix(path, "/archive/") {
		html := renderArchive(path)
		writeHTML(w, html)
		return
	}

	if strings.HasPrefix(path, "/posts/") {
		html := renderPost(path)
		writeHTML(w, html)
		return
	}

	html := renderPage(path)
	writeHTML(w, html)
}

func writeHTML(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func serveStatic(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path == "/" {
		return false
	}

	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return false
	}

	filePath := filepath.Join(publicDir, clean)
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}

	http.ServeFile(w, r, filePath)
	return true
}

func adminProxy() http.Handler {
	target, _ := url.Parse(adminURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = "localhost:5173"
	}
	return proxy
}

func renderHome() string {
	settings := getSettings()
	menuPages := getMenuPages()
	pageSize := settings.HomePageSize
	if pageSize <= 0 {
		pageSize = 3
	}
	posts := getPostsList(1, pageSize, "published = true")
	topImage := defaultString(settings.HomeTopImage, "/top.png")
	topImageAlt := defaultString(settings.HomeTopImageAlt, "Top Image")
	heroImage := ""
	if topImage != "" {
		heroImage = `<img src="` + html.EscapeString(topImage) + `" alt="` + html.EscapeString(topImageAlt) + `" class="top-image" />`
	}

	return renderHead("Home", settings) +
		renderNav(menuPages) +
		`<main class="body-home">
      <header class="page-header">
        ` + heroImage + `
        <h1 class="page-title">` + html.EscapeString(defaultString(settings.WelcomeText, "Welcome to your blog")) + `</h1>
      </header>
      ` + renderPostList(posts.Items) + `
      <hr>
      <p>More posts can be found in <a href="/archive/">the archive</a>.</p>
    </main>` + renderFooter()
}

func renderArchive(path string) string {
	settings := getSettings()
	menuPages := getMenuPages()
	parts := splitPath(path)
	tag := ""
	pageNumber := 1
	if len(parts) >= 2 {
		if isNumeric(parts[1]) {
			pageNumber = atoi(parts[1], 1)
		} else {
			tag = parts[1]
		}
	}
	if tag != "" && len(parts) >= 3 && isNumeric(parts[2]) {
		pageNumber = atoi(parts[2], 1)
	}

	filter := "published = true"
	if tag != "" {
		filter = fmt.Sprintf("published = true && tags ~ \"%s\"", tag)
	}

	pageSize := settings.ArchivePageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	posts := getPostsList(pageNumber, pageSize, filter)
	title := "Archive"
	if tag != "" {
		title = fmt.Sprintf("tag: %s", tag)
	}

	pagination := renderPagination(archiveBase(tag), pageNumber, posts.TotalPages)
	tagsNav := ""
	if settings.ShowArchiveTags && tag == "" && pageNumber == 1 {
		tagsNav = renderTagsNav(collectTags())
	}
	searchSlot := ""
	if settings.ShowArchiveSearch {
		searchSlot = `<div class="search" id="search"></div>`
	}

	rssLinks := ""
	if settings.EnableFeedXML || settings.EnableFeedJSON {
		rssLinks = `<p>RSS: <a href="/feed.xml">Atom</a>, <a href="/feed.json">JSON</a></p>`
	}
	return renderHead(title, settings) + renderNav(menuPages) +
		`<main class="body-tag">
      <header class="page-header">
        <h1 class="page-title">` + html.EscapeString(title) + `</h1>
        ` + rssLinks + `
        ` + searchSlot + `
      </header>
      ` + renderPostList(posts.Items) + `
      ` + pagination + `
      ` + tagsNav + `
    </main>` + renderFooter()
}

func renderPost(path string) string {
	settings := getSettings()
	menuPages := getMenuPages()
	parts := splitPath(path)
	if len(parts) < 2 {
		return renderNotFound()
	}
	slug := parts[1]
	post := getPostBySlug(slug)
	if post == nil {
		return renderNotFound()
	}

	body := post.Body
	if body == "" {
		body = post.Content
	}
	bodyWithIds, toc := buildTOC(body)
	prev, next := getAdjacentPosts(post)

	return renderHead(post.Title, settings) + renderNav(menuPages) +
		`<main class="body-post">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">` + html.EscapeString(post.Title) + `</h1>
          <div class="post-details">
            ` + renderTime(post) + `
            <p>` + fmt.Sprintf("%d min read", readingMinutes(body)) + `</p>
            ` + renderCategory(post.Category) + `
            ` + renderPostTags(post.Tags) + `
          </div>
        </header>
        ` + renderTocBlock(settings.ShowToc, toc) + `
        <div class="post-body body">` + bodyWithIds + `</div>
      </article>
      ` + renderPostNav(prev, next) + `
    </main>` + renderFooter()
}

func renderPage(path string) string {
	settings := getSettings()
	menuPages := getMenuPages()
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	page := getPageByURL(path)
	if page == nil {
		return renderNotFound()
	}
	body := page.Body
	if body == "" {
		body = page.Content
	}

	return renderHead(page.Title, settings) + renderNav(menuPages) +
		`<main class="body-tag">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">` + html.EscapeString(page.Title) + `</h1>
        </header>
        <div class="post-body body">` + body + `</div>
      </article>
    </main>` + renderFooter()
}

func renderNotFound() string {
	menuPages := getMenuPages()
	return renderHead("Not Found", getSettings()) + renderNav(menuPages) +
		`<main class="body-post">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">Not Found</h1>
        </header>
        <div class="post-body body">ページが見つかりませんでした。</div>
      </article>
    </main>` + renderFooter()
}

func renderHead(title string, settings SettingsRecord) string {
	siteName := defaultString(settings.SiteName, "Example Blog")
	description := defaultString(settings.Description, "A calm place to write.")
	highlight := ""
	highlightScript := ""
	if settings.EnableCodeHighlight {
		theme := defaultString(settings.HighlightTheme, "github-dark")
		highlight = `<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/` + html.EscapeString(theme) + `.min.css" />`
		highlightScript = `<script defer src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
    <script>window.addEventListener('DOMContentLoaded',()=>{if(window.hljs){window.hljs.highlightAll();}});</script>`
	}

	analytics := ""
	if settings.EnableAnalytics && settings.AnalyticsURL != "" && settings.AnalyticsSiteID != "" {
		analytics = `<script defer src="` + html.EscapeString(settings.AnalyticsURL) + `" data-website-id="` + html.EscapeString(settings.AnalyticsSiteID) + `"></script>`
	}
	ads := ""
	if settings.EnableAds && settings.AdsClient != "" {
		ads = `<script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=` + html.EscapeString(settings.AdsClient) + `" crossorigin="anonymous"></script>`
	}

	return `<!doctype html>
<html lang="ja">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>` + html.EscapeString(title) + ` - ` + html.EscapeString(siteName) + `</title>
    <meta name="supported-color-schemes" content="light dark" />
    <meta name="theme-color" content="hsl(220, 20%, 100%)" media="(prefers-color-scheme: light)" />
    <meta name="theme-color" content="hsl(220, 20%, 10%)" media="(prefers-color-scheme: dark)" />
    <link rel="stylesheet" href="/styles.css" />
    ` + feedLinkXML(settings, siteName) + `
    ` + feedLinkJSON(settings, siteName) + `
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon.png" />
    <meta name="description" content="` + html.EscapeString(description) + `" />
    ` + highlight + `
    <style>
      .page-title {
        font-size: 1.2em;
        background: var(--color-highlight);
        padding: 0.5em;
      }
    </style>
    ` + analytics + `
    ` + ads + `
    ` + highlightScript + `
  </head>
  <body>`
}

func renderNav(menuPages []PageRecord) string {
	settings := getSettings()
	var b strings.Builder
	b.WriteString(`<li><a href="/archive/">Archive</a></li>`)
	for _, page := range menuPages {
		label := page.MenuTitle
		if label == "" {
			label = page.Title
		}
		b.WriteString(`<li><a href="` + page.URL + `">` + html.EscapeString(label) + `</a></li>`)
	}

	return `<nav class="navbar">
      <a href="/" class="navbar-home">
        <strong>` + html.EscapeString(defaultString(settings.SiteName, "Example Blog")) + `</strong>
      </a>

      <ul class="navbar-links">
        ` + b.String() + `
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
    </nav>`
}

func renderFooter() string {
	settings := getSettings()
	footer := defaultString(settings.FooterHTML, `<div style="text-align: center;"><a href="/pgp/">PGP</a> --- <a href="/contact/">Contact</a> --- <a href="/machines/">Machines</a> --- <a href="/cat-v/">cat -v</a></div>`)
	return `
    ` + footer + `
  </body>
</html>`
}

func renderPostList(items []PostRecord) string {
	var b strings.Builder
	for _, post := range items {
		body := post.Body
		if body == "" {
			body = post.Content
		}
		excerpt := post.Excerpt
		if excerpt == "" {
			excerpt = buildExcerpt(body, 160)
		}
		b.WriteString(`<article class="post">
          <header class="post-header">
            <h2 class="post-title">
              <a href="/posts/` + post.Slug + `/">` + html.EscapeString(post.Title) + `</a>
            </h2>
            <div class="post-details">
              ` + renderTime(&post) + `
              <p>` + fmt.Sprintf("%d min read", readingMinutes(body)) + `</p>
              ` + renderPostTags(post.Tags) + `
            </div>
          </header>
          <div class="post-excerpt body">` + excerpt + `</div>
          <a href="/posts/` + post.Slug + `/" class="post-link">Read →</a>
        </article>`)
	}
	return `<section class="postList">` + b.String() + `</section>`
}

func renderTime(post *PostRecord) string {
	date := post.PublishedAt
	if date == "" {
		date = post.Date
	}
	if date == "" {
		return ""
	}
	return `<p><time datetime="` + date + `">` + formatDate(date) + `</time></p>`
}

func renderCategory(category string) string {
	if category == "" {
		return ""
	}
	return `<p>` + html.EscapeString(category) + `</p>`
}

func renderPostTags(tags string) string {
	parsed := parseTags(tags)
	if len(parsed) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="post-tags">`)
	for _, tag := range parsed {
		b.WriteString(`<a class="badge" href="/archive/` + url.PathEscape(tag) + `/">` + html.EscapeString(tag) + `</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderPostNav(prev *PostRecord, next *PostRecord) string {
	if prev == nil && next == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="page-pagination pagination"><ul>`)
	if prev != nil {
		b.WriteString(`<li class="pagination-prev"><a href="/posts/` + prev.Slug + `/" rel="prev"><span>← Older</span><strong>` + html.EscapeString(prev.Title) + `</strong></a></li>`)
	}
	if next != nil {
		b.WriteString(`<li class="pagination-next"><a href="/posts/` + next.Slug + `/" rel="next"><span>Newer →</span><strong>` + html.EscapeString(next.Title) + `</strong></a></li>`)
	}
	b.WriteString(`</ul></nav>`)
	return b.String()
}

func renderTocBlock(enabled bool, tocHTML string) string {
	if !enabled || tocHTML == "" {
		return ""
	}
	return `<nav class="toc"><h2>Content</h2>` + tocHTML + `</nav>`
}

func renderPagination(baseURL string, pageNumber, totalPages int) string {
	if totalPages <= 1 {
		return ""
	}
	prev := ""
	next := ""
	if pageNumber > 1 {
		prevPage := pageNumber - 1
		link := paginationLink(baseURL, prevPage)
		prev = `<li class="pagination-prev"><a href="` + link + `" rel="prev"><span>Previous</span><strong>` + strconv.Itoa(prevPage) + `</strong></a></li>`
	}
	if pageNumber < totalPages {
		nextPage := pageNumber + 1
		link := paginationLink(baseURL, nextPage)
		next = `<li class="pagination-next"><a href="` + link + `" rel="next"><span>Next</span><strong>` + strconv.Itoa(nextPage) + `</strong></a></li>`
	}
	return `<nav class="page-pagination pagination"><ul>` + prev + next + `</ul></nav>`
}

func renderTagsNav(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var b strings.Builder
	for _, tag := range tags {
		b.WriteString(`<li><a href="/archive/` + url.PathEscape(tag) + `/" class="badge">` + html.EscapeString(tag) + `</a></li>`)
	}
	return `<nav class="page-navigation"><h2>tags:</h2><ul class="page-navigation-tags">` + b.String() + `</ul></nav>`
}

func collectTags() []string {
	set := map[string]struct{}{}
	page := 1
	perPage := 200
	for {
		posts := getPostsList(page, perPage, "published = true")
		for _, post := range posts.Items {
			for _, tag := range parseTags(post.Tags) {
				set[tag] = struct{}{}
			}
		}
		if len(posts.Items) < perPage {
			break
		}
		page++
	}

	result := make([]string, 0, len(set))
	for tag := range set {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

func getMenuPages() []PageRecord {
	params := url.Values{}
	params.Set("page", "1")
	params.Set("perPage", "200")
	params.Set("filter", "published = true && menuVisible = true")
	params.Set("sort", "menuOrder")
	data, err := fetchPages(params)
	if err != nil {
		return []PageRecord{}
	}
	return data.Items
}

func getPostsList(page, perPage int, filter string) PBList[PostRecord] {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("perPage", strconv.Itoa(perPage))
	params.Set("filter", filter)
	params.Set("sort", "-published_at")
	data, err := fetchPosts(params)
	if err == nil {
		return data
	}
	params.Set("sort", "-date")
	data, err = fetchPosts(params)
	if err != nil {
		return PBList[PostRecord]{}
	}
	return data
}

func getPostBySlug(slug string) *PostRecord {
	params := url.Values{}
	params.Set("page", "1")
	params.Set("perPage", "1")
	params.Set("filter", fmt.Sprintf("slug = \"%s\" && published = true", slug))
	data, err := fetchPosts(params)
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func getAdjacentPosts(post *PostRecord) (*PostRecord, *PostRecord) {
	field := "published_at"
	value := post.PublishedAt
	if value == "" {
		field = "date"
		value = post.Date
	}
	if value == "" {
		return nil, nil
	}

	filterPrev := fmt.Sprintf(`published = true && %s < "%s"`, field, escapeFilter(value))
	filterNext := fmt.Sprintf(`published = true && %s > "%s"`, field, escapeFilter(value))

	prevList := getPostsListWithSort(1, 1, filterPrev, "-"+field)
	nextList := getPostsListWithSort(1, 1, filterNext, field)

	var prev *PostRecord
	var next *PostRecord
	if len(prevList.Items) > 0 {
		prev = &prevList.Items[0]
	}
	if len(nextList.Items) > 0 {
		next = &nextList.Items[0]
	}
	return prev, next
}

func getPostsListWithSort(page, perPage int, filter, sortField string) PBList[PostRecord] {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("perPage", strconv.Itoa(perPage))
	params.Set("filter", filter)
	params.Set("sort", sortField)
	data, err := fetchPosts(params)
	if err != nil {
		return PBList[PostRecord]{}
	}
	return data
}

func getPageByURL(path string) *PageRecord {
	params := url.Values{}
	params.Set("page", "1")
	params.Set("perPage", "1")
	params.Set("filter", fmt.Sprintf("url = \"%s\" && published = true", path))
	data, err := fetchPages(params)
	if err != nil || len(data.Items) == 0 {
		return nil
	}
	return &data.Items[0]
}

func fetchPosts(params url.Values) (PBList[PostRecord], error) {
	var data PBList[PostRecord]
	endpoint := pbURL + "/api/collections/posts/records?" + params.Encode()
	if err := fetchJSON(endpoint, &data); err != nil {
		return data, err
	}
	return data, nil
}

func fetchPages(params url.Values) (PBList[PageRecord], error) {
	var data PBList[PageRecord]
	endpoint := pbURL + "/api/collections/pages/records?" + params.Encode()
	if err := fetchJSON(endpoint, &data); err != nil {
		return data, err
	}
	return data, nil
}

func getSettings() SettingsRecord {
	params := url.Values{}
	params.Set("page", "1")
	params.Set("perPage", "1")
	data, err := fetchSettings(params)
	if err != nil || len(data.Items) == 0 {
		return SettingsRecord{
			SiteName:           "Example Blog",
			Description:        "A calm place to write.",
			WelcomeText:        "Welcome to your blog",
			HomeTopImage:       "/top.png",
			HomeTopImageAlt:    "Top Image",
			FooterHTML:         `<div style="text-align: center;"><a href="/pgp/">PGP</a> --- <a href="/contact/">Contact</a> --- <a href="/machines/">Machines</a> --- <a href="/cat-v/">cat -v</a></div>`,
			SiteURL:            "",
			SiteLanguage:       "ja",
			EnableFeedXML:      true,
			EnableFeedJSON:     true,
			FeedItemsLimit:     30,
			EnableAnalytics:    false,
			EnableAds:          false,
			EnableCodeHighlight: true,
			HighlightTheme:     "github-dark",
			ArchivePageSize:    10,
			HomePageSize:       3,
			ShowToc:            true,
			ShowArchiveTags:    true,
			ShowArchiveSearch:  true,
		}
	}
	set := data.Items[0]
	if set.ArchivePageSize == 0 {
		set.ArchivePageSize = 10
	}
	if set.HomePageSize == 0 {
		set.HomePageSize = 3
	}
	if set.SiteName == "" {
		set.SiteName = "Example Blog"
	}
	if set.Description == "" {
		set.Description = "A calm place to write."
	}
	if set.WelcomeText == "" {
		set.WelcomeText = "Welcome to your blog"
	}
	if set.HomeTopImage == "" {
		set.HomeTopImage = "/top.png"
	}
	if set.HomeTopImageAlt == "" {
		set.HomeTopImageAlt = "Top Image"
	}
	if set.FooterHTML == "" {
		set.FooterHTML = `<div style="text-align: center;"><a href="/pgp/">PGP</a> --- <a href="/contact/">Contact</a> --- <a href="/machines/">Machines</a> --- <a href="/cat-v/">cat -v</a></div>`
	}
	if set.SiteLanguage == "" {
		set.SiteLanguage = "ja"
	}
	if set.FeedItemsLimit <= 0 {
		set.FeedItemsLimit = 30
	}
	return set
}

func feedLinkXML(settings SettingsRecord, siteName string) string {
	if !settings.EnableFeedXML {
		return ""
	}
	return `<link rel="alternate" href="/feed.xml" type="application/atom+xml" title="` + html.EscapeString(siteName) + `" />`
}

func feedLinkJSON(settings SettingsRecord, siteName string) string {
	if !settings.EnableFeedJSON {
		return ""
	}
	return `<link rel="alternate" href="/feed.json" type="application/json" title="` + html.EscapeString(siteName) + `" />`
}

type JSONFeedItem struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	ContentHTML string `json:"content_html"`
	Date        string `json:"date_published"`
}

type JSONFeed struct {
	Version     string         `json:"version"`
	Title       string         `json:"title"`
	HomePageURL string         `json:"home_page_url"`
	FeedURL     string         `json:"feed_url"`
	Description string         `json:"description"`
	Items       []JSONFeedItem `json:"items"`
}

func writeJSONFeed(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	limit := settings.FeedItemsLimit
	posts := getPostsList(1, limit, "published = true")
	base := resolveSiteURL(settings, r)
	items := make([]JSONFeedItem, 0, len(posts.Items))
	for _, post := range posts.Items {
		body := post.Body
		if body == "" {
			body = post.Content
		}
		date := pickPostDate(post)
		url := base + "/posts/" + post.Slug + "/"
		items = append(items, JSONFeedItem{
			ID:          url,
			URL:         url,
			Title:       post.Title,
			ContentHTML: body,
			Date:        date,
		})
	}
	feed := JSONFeed{
		Version:     "https://jsonfeed.org/version/1",
		Title:       defaultString(settings.SiteName, "Example Blog"),
		HomePageURL: base + "/",
		FeedURL:     base + "/feed.json",
		Description: defaultString(settings.Description, ""),
		Items:       items,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(feed)
}

func writeRSSFeed(w http.ResponseWriter, r *http.Request, settings SettingsRecord) {
	limit := settings.FeedItemsLimit
	posts := getPostsList(1, limit, "published = true")
	base := resolveSiteURL(settings, r)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rss xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">` + "\n")
	b.WriteString("<channel>\n")
	b.WriteString(`<title>` + xmlEscape(defaultString(settings.SiteName, "Example Blog")) + `</title>` + "\n")
	b.WriteString(`<link>` + xmlEscape(base+"/") + `</link>` + "\n")
	b.WriteString(`<atom:link href="` + xmlEscape(base+"/feed.xml") + `" rel="self" type="application/rss+xml" />` + "\n")
	b.WriteString(`<description>` + xmlEscape(defaultString(settings.Description, "")) + `</description>` + "\n")
	b.WriteString(`<language>` + xmlEscape(defaultString(settings.SiteLanguage, "ja")) + `</language>` + "\n")
	for _, post := range posts.Items {
		body := post.Body
		if body == "" {
			body = post.Content
		}
		date := pickPostDate(post)
		pubDate := rfc1123(date)
		url := base + "/posts/" + post.Slug + "/"
		b.WriteString("<item>\n")
		b.WriteString(`<title>` + xmlEscape(post.Title) + `</title>` + "\n")
		b.WriteString(`<link>` + xmlEscape(url) + `</link>` + "\n")
		b.WriteString(`<guid isPermaLink="false">` + xmlEscape(url) + `</guid>` + "\n")
		if pubDate != "" {
			b.WriteString(`<pubDate>` + xmlEscape(pubDate) + `</pubDate>` + "\n")
		}
		b.WriteString(`<content:encoded><![CDATA[` + body + `]]></content:encoded>` + "\n")
		b.WriteString("</item>\n")
	}
	b.WriteString("</channel>\n</rss>")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func resolveSiteURL(settings SettingsRecord, r *http.Request) string {
	if settings.SiteURL != "" {
		return strings.TrimRight(settings.SiteURL, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func pickPostDate(post PostRecord) string {
	if post.PublishedAt != "" {
		return post.PublishedAt
	}
	return post.Date
}

func rfc1123(value string) string {
	if value == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	return t.Format(time.RFC1123Z)
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func fetchSettings(params url.Values) (PBList[SettingsRecord], error) {
	var data PBList[SettingsRecord]
	endpoint := pbURL + "/api/collections/settings/records?" + params.Encode()
	if err := fetchJSON(endpoint, &data); err != nil {
		return data, err
	}
	return data, nil
}

func fetchJSON(url string, target any) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return []string{}
	}
	return parts
}

func archiveBase(tag string) string {
	if tag == "" {
		return "/archive"
	}
	return "/archive/" + url.PathEscape(tag)
}

func paginationLink(base string, page int) string {
	if page == 1 {
		return base + "/"
	}
	return base + "/" + strconv.Itoa(page) + "/"
}

func parseTags(tags string) []string {
	parts := strings.Split(tags, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func buildExcerpt(body string, length int) string {
	text := stripHTML(body)
	if len(text) <= length {
		return text
	}
	return text[:length] + "..."
}

func stripHTML(body string) string {
	inTag := false
	var b strings.Builder
	for _, r := range body {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		case '\n', '\r', '\t':
			b.WriteRune(' ')
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func readingMinutes(body string) int {
	length := len([]rune(stripHTML(body)))
	minutes := length / 700
	if length%700 != 0 {
		minutes++
	}
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}

func formatDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return humanDate(parsed)
}

func humanDate(t time.Time) string {
	day := t.Day()
	suffix := "th"
	if day%100 < 11 || day%100 > 13 {
		switch day % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return t.Format("January ") + strconv.Itoa(day) + suffix + t.Format(", 2006")
}

func atoi(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func escapeFilter(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func buildTOC(body string) (string, string) {
	used := map[string]int{}
	type tocEntry struct {
		level int
		text  string
		id    string
	}
	entries := []tocEntry{}
	var out strings.Builder
	last := 0
	matches := headingRe.FindAllStringSubmatchIndex(body, -1)
	for _, m := range matches {
		out.WriteString(body[last:m[0]])
		levelStr := body[m[2]:m[3]]
		attrs := body[m[4]:m[5]]
		content := body[m[6]:m[7]]
		text := stripHTML(content)
		id := extractID(attrs)
		if id == "" {
			id = slugify(text)
			if id == "" {
				id = "section"
			}
			used[id]++
			if used[id] > 1 {
				id = fmt.Sprintf("%s-%d", id, used[id])
			}
			attrs = strings.TrimSpace(attrs) + ` id="` + id + `"`
		}
		level := atoi(levelStr, 2)
		entries = append(entries, tocEntry{level: level, text: text, id: id})
		out.WriteString(`<h` + levelStr + ` ` + strings.TrimSpace(attrs) + `>` + content + `</h` + levelStr + `>`)
		last = m[1]
	}
	out.WriteString(body[last:])

	if len(entries) == 0 {
		return out.String(), ""
	}

	var toc strings.Builder
	toc.WriteString("<ol>")
	var inSub bool
	for i, entry := range entries {
		if entry.level == 3 && !inSub {
			toc.WriteString("<ul>")
			inSub = true
		}
		if entry.level == 2 && inSub {
			toc.WriteString("</ul>")
			inSub = false
		}
		if entry.level == 2 {
			if i > 0 {
				toc.WriteString("</li>")
			}
			toc.WriteString(`<li><a href="#` + entry.id + `">` + html.EscapeString(entry.text) + `</a>`)
		} else {
			toc.WriteString(`<li><a href="#` + entry.id + `">` + html.EscapeString(entry.text) + `</a></li>`)
		}
	}
	if inSub {
		toc.WriteString("</ul>")
	}
	toc.WriteString("</li></ol>")

	return out.String(), toc.String()
}

func extractID(attrs string) string {
	re := regexp.MustCompile(`id="([^"]+)"`)
	m := re.FindStringSubmatch(attrs)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func slugify(text string) string {
	lower := strings.ToLower(text)
	var b strings.Builder
	prevDash := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	return result
}
