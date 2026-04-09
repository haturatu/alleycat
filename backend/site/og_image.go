package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	postOGImageWidth  = 1200
	postOGImageHeight = 630
)

type fontSource struct {
	path  string
	index int
	data  []byte
}

type ogFontSpec struct {
	size    float64
	sources []fontSource
}

type ogFontSet struct {
	entries []ogFontEntry
}

type ogFontRun struct {
	face font.Face
	text string
}

type ogFontEntry struct {
	font *opentype.Font
	face font.Face
	buf  sfnt.Buffer
}

var ogFontCache = struct {
	mu    sync.Mutex
	items map[string]*ogFontSet
}{
	items: map[string]*ogFontSet{},
}

func buildAbsoluteSiteURL(settings SettingsRecord, path string) string {
	base := normalizeSiteBaseURL(settings.SiteURL)
	clean := cleanPath(path)
	if clean == "" {
		clean = "/"
	}
	if base == "" {
		return clean
	}
	return base + clean
}

func postOGImageRoute(locale, slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	if normalizedLocale := normalizeLocale(locale); normalizedLocale != "" {
		return "/og/" + url.PathEscape(normalizedLocale) + "/posts/" + url.PathEscape(slug) + ".png"
	}
	return "/og/posts/" + url.PathEscape(slug) + ".png"
}

func extractPostOGImageRequest(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 3 && parts[0] == "og" && parts[1] == "posts" && strings.HasSuffix(parts[2], ".png") {
		slug, err := url.PathUnescape(strings.TrimSuffix(parts[2], ".png"))
		if err != nil {
			return "", "", false
		}
		return "", strings.TrimSpace(slug), strings.TrimSpace(slug) != ""
	}
	if len(parts) == 4 && parts[0] == "og" && parts[2] == "posts" && strings.HasSuffix(parts[3], ".png") {
		rawLocale, localeErr := url.PathUnescape(parts[1])
		slug, slugErr := url.PathUnescape(strings.TrimSuffix(parts[3], ".png"))
		if localeErr != nil || slugErr != nil {
			return "", "", false
		}
		slug = strings.TrimSpace(slug)
		locale, ok := parseLocaleSegment(rawLocale)
		if !ok || slug == "" {
			return "", "", false
		}
		return locale, slug, true
	}
	return "", "", false
}

func extractSlugFromPostPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 2 && parts[0] == "posts" {
		return strings.TrimSpace(parts[1])
	}
	if len(parts) == 3 && parts[1] == "posts" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func extractLocaleFromPostPath(path string) string {
	locale, _, ok := extractLocalizedPostRoute(path)
	if ok {
		return locale
	}
	return ""
}

func servePostOGImage(w http.ResponseWriter, path string, settings SettingsRecord) bool {
	locale, slug, ok := extractPostOGImageRequest(path)
	if !ok {
		return false
	}
	lookupLocale := locale
	if locale != "" && isSourceLocale(settings, locale) {
		lookupLocale = ""
	} else if locale != "" && !isEnabledTranslationLocale(settings, locale) {
		return false
	}

	var post *PostRecord
	if resolved := resolvePublishedPost(slug, lookupLocale); resolved != nil {
		post = resolved.post
	}
	if post == nil && lookupLocale == "" {
		post = getPublishedSourcePostByTranslationSlug(slug)
	}
	if post == nil {
		return false
	}

	body, err := renderGeneratedPostOGImage(post, locale, settings)
	if err != nil {
		return false
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func renderGeneratedPostOGImage(post *PostRecord, locale string, settings SettingsRecord) ([]byte, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, postOGImageWidth, postOGImageHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{250, 244, 236, 255}), image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(0, 0, postOGImageWidth, 18), image.NewUniform(color.RGBA{197, 105, 57, 255}), image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(70, 92, postOGImageWidth-70, postOGImageHeight-92), image.NewUniform(color.RGBA{255, 251, 246, 255}), image.Point{}, draw.Src)

	titleSources := []fontSource{
		{path: "/usr/share/fonts/noto/NotoSansCJK-Black.ttc", index: 0},
		{path: "/usr/share/fonts/noto/NotoSansCJK-Medium.ttc", index: 0},
		{path: "/usr/share/fonts/noto/NotoSans-Bold.ttf"},
		{data: gobold.TTF},
	}
	_, err := loadOGFontSet(ogFontSpec{
		size:    54,
		sources: titleSources,
	})
	if err != nil {
		return nil, err
	}
	bodySources := []fontSource{
		{path: "/usr/share/fonts/noto/NotoSansCJK-Medium.ttc", index: 0},
		{path: "/usr/share/fonts/noto/NotoSansCJK-Light.ttc", index: 0},
		{path: "/usr/share/fonts/noto/NotoSans-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansDevanagari-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansThai-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansBengali-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansHebrew-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansMyanmar-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansKhmer-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansTamil-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansTelugu-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansKannada-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansMalayalam-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansGujarati-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansGurmukhi-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansOriya-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansSinhala-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansLao-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansGeorgian-Regular.ttf"},
		{path: "/usr/share/fonts/noto/NotoSansArmenian-Regular.ttf"},
		{data: goregular.TTF},
	}
	_, err = loadOGFontSet(ogFontSpec{
		size:    27,
		sources: bodySources,
	})
	if err != nil {
		return nil, err
	}
	metaSources := []fontSource{
		{path: "/usr/share/fonts/noto/NotoSansCJK-Medium.ttc", index: 0},
		{path: "/usr/share/fonts/noto/NotoSansCJK-Light.ttc", index: 0},
		{path: "/usr/share/fonts/noto/NotoSans-Regular.ttf"},
		{data: goregular.TTF},
	}
	metaFonts, err := loadOGFontSet(ogFontSpec{
		size:    22,
		sources: metaSources,
	})
	if err != nil {
		return nil, err
	}

	accent := color.RGBA{197, 105, 57, 255}
	textColor := color.RGBA{33, 31, 28, 255}
	muted := color.RGBA{92, 84, 76, 255}
	left := 110
	top := 150
	maxWidth := postOGImageWidth - (left * 2)
	cardBottom := postOGImageHeight - 92
	footerBaseline := cardBottom - 14
	bodyBottomLimit := footerBaseline - 34

	localeLabel := normalizeLocale(locale)
	if localeLabel == "" {
		localeLabel = normalizeLocale(settings.SiteLanguage)
	}
	header := strings.TrimSpace(settings.SiteName)
	if localeLabel != "" {
		header = strings.TrimSpace(header + "  " + strings.ToUpper(localeLabel))
	}
	drawTextLine(canvas, metaFonts, left, top, accent, truncateText(header, metaFonts, maxWidth))

	title := strings.TrimSpace(defaultString(post.Title, post.Slug))
	titleFonts, titleLineHeight := fitTextToBox(titleSources, title, maxWidth, 4, 42, 54, 6)
	titleLines := wrapText(title, titleFonts, maxWidth, 4)
	y := top + 76
	for _, line := range titleLines {
		drawTextLine(canvas, titleFonts, left, y, textColor, line)
		y += titleLineHeight
	}

	body := post.Body
	if body == "" {
		body = post.Content
	}
	excerpt := strings.TrimSpace(post.Excerpt)
	if excerpt == "" {
		excerpt = buildExcerpt(body, 180)
	}
	availableBodyHeight := bodyBottomLimit - (y + 10)
	maxBodyLines := 4
	if availableBodyHeight > 0 {
		estimated := availableBodyHeight / 30
		if estimated < maxBodyLines {
			maxBodyLines = estimated
		}
	}
	if maxBodyLines < 1 {
		maxBodyLines = 1
	}
	bodyFonts, bodyLineHeight := fitTextToBox(bodySources, excerpt, maxWidth, maxBodyLines, 20, 27, 4)
	for _, line := range wrapText(excerpt, bodyFonts, maxWidth, maxBodyLines) {
		drawTextLine(canvas, bodyFonts, left, y+10, muted, line)
		y += bodyLineHeight
	}

	footerDate := strings.TrimSpace(post.PublishedAt)
	if footerDate == "" {
		footerDate = strings.TrimSpace(post.Date)
	}
	footerParts := []string{}
	if footerDate != "" {
		footerParts = append(footerParts, formatDate(footerDate))
	}
	if slug := strings.TrimSpace(post.Slug); slug != "" {
		footerParts = append(footerParts, "/posts/"+slug+"/")
	}
	drawTextLine(canvas, metaFonts, left, footerBaseline, muted, truncateText(strings.Join(footerParts, "  "), metaFonts, maxWidth))

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func loadOGFontSet(spec ogFontSpec) (*ogFontSet, error) {
	key := ogFontCacheKey(spec)

	ogFontCache.mu.Lock()
	defer ogFontCache.mu.Unlock()
	if cached := ogFontCache.items[key]; cached != nil {
		return cached, nil
	}

	set := &ogFontSet{}
	for _, source := range spec.sources {
		entry, err := loadOGFontEntry(source, spec.size)
		if err != nil || entry.face == nil || entry.font == nil {
			continue
		}
		set.entries = append(set.entries, entry)
	}
	if len(set.entries) == 0 {
		return nil, os.ErrNotExist
	}
	ogFontCache.items[key] = set
	return set, nil
}

func ogFontCacheKey(spec ogFontSpec) string {
	parts := []string{strconv.FormatFloat(spec.size, 'f', 2, 64)}
	for _, source := range spec.sources {
		parts = append(parts, source.path+"#"+strconv.Itoa(source.index))
		if len(source.data) > 0 {
			parts = append(parts, "embedded")
		}
	}
	return strings.Join(parts, "|")
}

func loadOGFontEntry(source fontSource, size float64) (ogFontEntry, error) {
	data := source.data
	if len(data) == 0 && source.path != "" {
		resolved := filepath.Clean(source.path)
		fileData, err := os.ReadFile(resolved)
		if err != nil {
			return ogFontEntry{}, err
		}
		data = fileData
	}
	if len(data) == 0 {
		return ogFontEntry{}, os.ErrNotExist
	}

	var parsed *opentype.Font
	var err error
	if strings.HasSuffix(strings.ToLower(source.path), ".ttc") {
		collection, collectionErr := opentype.ParseCollection(data)
		if collectionErr != nil {
			return ogFontEntry{}, collectionErr
		}
		parsed, err = collection.Font(source.index)
	} else {
		parsed, err = opentype.Parse(data)
	}
	if err != nil {
		return ogFontEntry{}, err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return ogFontEntry{}, err
	}
	return ogFontEntry{
		font: parsed,
		face: face,
	}, nil
}

func chooseFaceForRune(set *ogFontSet, r rune) font.Face {
	if set == nil || len(set.entries) == 0 {
		return nil
	}
	for i := range set.entries {
		entry := &set.entries[i]
		if entry.face == nil || entry.font == nil {
			continue
		}
		glyphIndex, err := entry.font.GlyphIndex(&entry.buf, r)
		if err == nil && glyphIndex != 0 {
			return entry.face
		}
	}
	return set.entries[0].face
}

func splitTextRuns(set *ogFontSet, value string) []ogFontRun {
	runes := []rune(value)
	if len(runes) == 0 {
		return nil
	}
	runs := make([]ogFontRun, 0, len(runes))
	var currentFace font.Face
	var current []rune
	flush := func() {
		if len(current) == 0 || currentFace == nil {
			return
		}
		runs = append(runs, ogFontRun{face: currentFace, text: string(current)})
		current = nil
	}
	for _, r := range runes {
		nextFace := chooseFaceForRune(set, r)
		if currentFace != nil && currentFace != nextFace {
			flush()
		}
		currentFace = nextFace
		current = append(current, r)
	}
	flush()
	return runs
}

func measureText(set *ogFontSet, value string) fixed.Int26_6 {
	var total fixed.Int26_6
	for _, run := range splitTextRuns(set, value) {
		drawer := &font.Drawer{Face: run.face}
		total += drawer.MeasureString(run.text)
	}
	return total
}

func drawTextLine(dst draw.Image, set *ogFontSet, x, y int, src color.Color, value string) {
	if set == nil || strings.TrimSpace(value) == "" {
		return
	}
	offset := fixed.P(x, y)
	for _, run := range splitTextRuns(set, value) {
		drawer := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(src),
			Face: run.face,
			Dot:  offset,
		}
		if drawStringNoPanic(drawer, run.text) {
			offset.X += drawer.MeasureString(run.text)
			continue
		}
		for _, r := range []rune(run.text) {
			piece := string(r)
			drawer.Dot = offset
			if drawStringNoPanic(drawer, piece) {
				offset.X += drawer.MeasureString(piece)
				continue
			}
			offset.X += drawer.MeasureString("\uFFFD")
		}
	}
}

func drawStringNoPanic(drawer *font.Drawer, text string) (ok bool) {
	if drawer == nil || drawer.Face == nil || text == "" {
		return false
	}
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	drawer.DrawString(text)
	return true
}

func wrapText(value string, set *ogFontSet, maxWidth int, maxLines int) []string {
	tokens := splitWrapTokens(strings.TrimSpace(value))
	if len(tokens) == 0 {
		return nil
	}
	lines := make([]string, 0, maxLines)
	current := tokens[0]

	for _, token := range tokens[1:] {
		candidate := current + token
		if measureText(set, candidate).Ceil() <= maxWidth {
			current = candidate
			continue
		}
		lines = append(lines, strings.TrimSpace(current))
		current = strings.TrimLeft(token, " ")
		if len(lines) == maxLines-1 {
			break
		}
	}

	if len(lines) < maxLines && current != "" {
		lines = append(lines, strings.TrimSpace(current))
	}
	if len(lines) == 0 {
		return []string{truncateText(value, set, maxWidth)}
	}

	consumed := strings.Join(lines, "")
	full := strings.TrimSpace(value)
	if len([]rune(consumed)) < len([]rune(full)) {
		remaining := []rune(full)[len([]rune(consumed)):]
		lines[len(lines)-1] = truncateText(lines[len(lines)-1]+string(remaining), set, maxWidth)
	}
	return lines
}

func truncateText(value string, set *ogFontSet, maxWidth int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if measureText(set, value).Ceil() <= maxWidth {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 {
		candidate := string(runes) + "..."
		if measureText(set, candidate).Ceil() <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return "..."
}

func fitTextToBox(sources []fontSource, value string, maxWidth int, maxLines int, minSize float64, initialSize float64, lineGap int) (*ogFontSet, int) {
	size := initialSize
	for size >= minSize {
		set, err := loadOGFontSet(ogFontSpec{
			size:    size,
			sources: sources,
		})
		if err == nil && set != nil {
			lines := wrapText(value, set, maxWidth, maxLines)
			fits := len(lines) <= maxLines
			if fits {
				return set, int(size) + lineGap
			}
		}
		size -= 2
	}
	set, _ := loadOGFontSet(ogFontSpec{
		size:    minSize,
		sources: sources,
	})
	return set, int(minSize) + lineGap
}

func splitWrapTokens(value string) []string {
	if value == "" {
		return nil
	}
	runes := []rune(value)
	tokens := make([]string, 0, len(runes))
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, string(current))
		current = nil
	}
	isASCIIWord := func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
	for _, r := range runes {
		switch {
		case r == ' ' || r == '\n' || r == '\t':
			flush()
			if len(tokens) > 0 {
				tokens = append(tokens, " ")
			}
		case isASCIIWord(r):
			if len(current) > 0 && !isASCIIWord(current[len(current)-1]) {
				flush()
			}
			current = append(current, r)
		default:
			flush()
			tokens = append(tokens, string(r))
		}
	}
	flush()
	return tokens
}
