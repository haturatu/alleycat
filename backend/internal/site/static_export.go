package site

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var prerenderedSnapshot = struct {
	mu  sync.RWMutex
	dir string
}{}

func warmStaticSnapshotAtBoot() {
	slog.Info("static snapshot bootstrap start", "static_export_dir", staticExportDir)
	if dir, err := buildStaticSnapshot(); err == nil {
		setPrerenderedSnapshotDir(dir)
		slog.Info("static snapshot bootstrap completed", "dir", dir)
		return
	} else {
		slog.Error("static snapshot bootstrap failed", "error", err)
	}

	go func() {
		backoff := 2 * time.Second
		for {
			slog.Info("static snapshot retry start", "backoff", backoff.String())
			dir, err := buildStaticSnapshot()
			if err == nil {
				setPrerenderedSnapshotDir(dir)
				slog.Info("static snapshot retry completed", "dir", dir)
				return
			}
			slog.Error("static snapshot retry failed", "backoff", backoff.String(), "error", err)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}()
}

func setPrerenderedSnapshotDir(next string) {
	prerenderedSnapshot.mu.Lock()
	prev := prerenderedSnapshot.dir
	prerenderedSnapshot.dir = next
	prerenderedSnapshot.mu.Unlock()
	if prev != "" && prev != next {
		_ = os.RemoveAll(prev)
	}
}

func getPrerenderedSnapshotDir() string {
	prerenderedSnapshot.mu.RLock()
	dir := prerenderedSnapshot.dir
	prerenderedSnapshot.mu.RUnlock()
	return dir
}

func buildStaticSnapshot() (string, error) {
	if err := os.MkdirAll(staticExportDir, 0o755); err != nil {
		return "", err
	}

	root, err := os.MkdirTemp(staticExportDir, "snapshot-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(root)
		}
	}()

	ctx, err := newSnapshotBuildContext()
	if err != nil {
		return "", err
	}
	settings := ctx.settings
	slog.Info("static snapshot build start",
		"root", root,
		"posts", len(ctx.publishedPosts),
		"pages", len(ctx.publishedPages),
		"tags", len(ctx.tags),
		"categories", len(ctx.categories),
		"locales", len(parseTranslationLocales(settings.TranslationLocales)),
	)

	err = withSnapshotBuildContext(ctx, func() error {
		if err := writeSnapshotRoute(root, "/", []byte(renderHome(settings))); err != nil {
			return err
		}
		if err := exportArchiveSeries(root, "/archive/", "published = true", settings); err != nil {
			return err
		}
		for _, tag := range ctx.tags {
			filter := fmt.Sprintf("published = true && tags ~ \"%s\"", escapeFilter(tag))
			basePath := "/archive/" + url.PathEscape(tag) + "/"
			if err := exportArchiveSeries(root, basePath, filter, settings); err != nil {
				return err
			}
		}
		for _, category := range ctx.categories {
			filter := fmt.Sprintf("published = true && category = \"%s\"", escapeFilter(category))
			basePath := "/archive/category/" + url.PathEscape(category) + "/"
			if err := exportArchiveSeries(root, basePath, filter, settings); err != nil {
				return err
			}
		}

		if err := renderSnapshotRoutesInParallel(root, snapshotBuildWorkers(), buildSnapshotRenderTasks(ctx, settings)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	slog.Info("static snapshot build completed", "root", root)
	return root, nil
}

func exportArchiveSeries(root, basePath, filter string, settings SettingsRecord) error {
	if ctx := currentSnapshotBuildContext(); ctx != nil {
		if listing, ok := ctx.archiveListing(basePath); ok {
			return exportArchiveListing(root, basePath, listing, settings)
		}
	}
	totalPages, err := archiveTotalPages(filter, settings.ArchivePageSize)
	if err != nil {
		return err
	}
	if totalPages < 1 {
		totalPages = 1
	}
	for pageNumber := 1; pageNumber <= totalPages; pageNumber++ {
		renderPath := strings.TrimSuffix(basePath, "/")
		writePath := basePath
		if pageNumber > 1 {
			renderPath = renderPath + "/" + strconv.Itoa(pageNumber)
			writePath = basePath + strconv.Itoa(pageNumber) + "/"
		}
		if err := writeSnapshotRoute(root, writePath, []byte(renderArchive(renderPath+"/", "", settings))); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *snapshotBuildContext) archiveListing(basePath string) (archiveListing, bool) {
	clean := cleanPath(basePath)
	if !strings.HasSuffix(clean, "/") {
		clean += "/"
	}
	listing, ok := ctx.archiveIndex[clean]
	return listing, ok
}

func exportArchiveListing(root, basePath string, listing archiveListing, settings SettingsRecord) error {
	pageCount := listing.pageCount
	if pageCount < 1 {
		pageCount = 1
	}
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		renderPath := strings.TrimSuffix(basePath, "/")
		writePath := basePath
		if pageNumber > 1 {
			renderPath = renderPath + "/" + strconv.Itoa(pageNumber)
			writePath = basePath + strconv.Itoa(pageNumber) + "/"
		}
		if err := writeSnapshotRoute(root, writePath, []byte(renderArchive(renderPath+"/", "", settings))); err != nil {
			return err
		}
	}
	return nil
}

type snapshotRenderTask struct {
	route  string
	render func() ([]byte, bool)
}

func buildSnapshotRenderTasks(ctx *snapshotBuildContext, settings SettingsRecord) []snapshotRenderTask {
	tasks := make([]snapshotRenderTask, 0, len(ctx.publishedPosts)+len(ctx.publishedPages)+len(ctx.translationByKey))
	for _, post := range ctx.publishedPosts {
		post := post
		slug := strings.TrimSpace(post.Slug)
		if slug == "" {
			continue
		}
		tasks = append(tasks, snapshotRenderTask{
			route: "/posts/" + url.PathEscape(slug) + "/",
			render: func() ([]byte, bool) {
				html, ok := renderPostFromInput(&postRenderInput{
					path:   "/posts/" + slug + "/",
					locale: "",
					slug:   slug,
					post:   &post,
				}, settings)
				return []byte(html), ok
			},
		})
	}
	for locale, items := range ctx.translationsByLocale {
		locale := locale
		for _, item := range items {
			item := item
			slug := strings.TrimSpace(item.Slug)
			if slug == "" {
				continue
			}
			tasks = append(tasks, snapshotRenderTask{
				route: "/" + url.PathEscape(locale) + "/posts/" + url.PathEscape(slug) + "/",
				render: func() ([]byte, bool) {
					post := translationToPost(item)
					html, ok := renderPostFromInput(&postRenderInput{
						path:        "/" + locale + "/posts/" + slug + "/",
						locale:      locale,
						slug:        slug,
						post:        &post,
						translation: &item,
					}, settings)
					return []byte(html), ok
				},
			})
		}
	}
	for _, page := range ctx.publishedPages {
		page := page
		pageURL := strings.TrimSpace(page.URL)
		if pageURL == "" {
			continue
		}
		tasks = append(tasks, snapshotRenderTask{
			route: pageURL,
			render: func() ([]byte, bool) {
				html, ok := renderPageFromRecord(&page, settings)
				return []byte(html), ok
			},
		})
	}
	return tasks
}

func snapshotBuildWorkers() int {
	workers := runtime.GOMAXPROCS(0)
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}
	return workers
}

func renderSnapshotRoutesInParallel(root string, workers int, tasks []snapshotRenderTask) error {
	if workers <= 1 || len(tasks) <= 1 {
		for _, task := range tasks {
			body, ok := task.render()
			if !ok {
				continue
			}
			if err := writeSnapshotRoute(root, task.route, body); err != nil {
				return err
			}
		}
		return nil
	}

	taskCh := make(chan snapshotRenderTask)
	errCh := make(chan error, 1)
	var once sync.Once
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for task := range taskCh {
			body, ok := task.render()
			if !ok {
				continue
			}
			if err := writeSnapshotRoute(root, task.route, body); err != nil {
				once.Do(func() { errCh <- err })
				return
			}
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}
	for _, task := range tasks {
		select {
		case err := <-errCh:
			close(taskCh)
			wg.Wait()
			return err
		default:
		}
		taskCh <- task
	}
	close(taskCh)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func archiveTotalPages(filter string, perPage int) (int, error) {
	if perPage <= 0 {
		perPage = 10
	}
	posts, err := getPosts(map[string]string{
		"page":    "1",
		"perPage": strconv.Itoa(perPage),
		"filter":  filter,
		"sort":    "-published_at",
	})
	if err != nil {
		posts, err = getPosts(map[string]string{
			"page":    "1",
			"perPage": strconv.Itoa(perPage),
			"filter":  filter,
			"sort":    "-date",
		})
		if err != nil {
			return 0, err
		}
	}
	return posts.TotalPages, nil
}

func writeSnapshotRoute(root, route string, body []byte) error {
	return writeSnapshotFile(root, route, body)
}

func writeSnapshotFile(root, route string, body []byte) error {
	target, err := snapshotFilePath(root, route)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, body, 0o644)
}

func snapshotFilePath(root, route string) (string, error) {
	clean := cleanPath(route)
	if clean == "/" {
		return filepath.Join(root, "index.html"), nil
	}
	trimmed := strings.TrimPrefix(clean, "/")
	if trimmed == "" {
		return filepath.Join(root, "index.html"), nil
	}
	target := filepath.Join(root, trimmed)
	if filepath.Ext(trimmed) == "" {
		target = filepath.Join(target, "index.html")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid snapshot path: %s", route)
	}
	return absTarget, nil
}
