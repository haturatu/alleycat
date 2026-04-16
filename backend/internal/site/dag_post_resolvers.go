package site

import (
	"errors"
	"fmt"
	"strings"

	"alleycat-backend/internal/dag"
)

var errUnsupportedRouteNode = errors.New("unsupported site route node")

func newSiteDAGEngine() *dag.Engine {
	engine := dag.NewEngine()
	engine.Register(nodeSettings, settingsResolver{})
	engine.Register(nodeMenuPages, menuPagesResolver{})
	engine.Register(nodePageByURL, pageByURLResolver{})
	engine.Register(nodePostBySlug, postBySlugResolver{})
	engine.Register(nodeTranslationBySlug, translationBySlugResolver{})
	engine.Register(nodePostFamily, postFamilyResolver{})
	engine.Register(nodeAdjacentPosts, adjacentPostsResolver{})
	engine.Register(nodeRelatedPosts, relatedPostsResolver{})
	engine.Register(nodeHomeListing, homeListingResolver{})
	engine.Register(nodeArchiveListing, archiveListingResolver{})
	engine.Register(nodeHomeRenderInput, homeRenderInputResolver{})
	engine.Register(nodeArchiveRenderInput, archiveRenderInputResolver{})
	engine.Register(nodePageRenderInput, pageRenderInputResolver{})
	engine.Register(nodePostRenderInput, postRenderInputResolver{})
	engine.Register(nodeRoute, routeResolver{})
	return engine
}

type settingsResolver struct{}

func (settingsResolver) Resolve(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		return dag.ResolveResult{
			Value: snapshot.settings,
		}, nil
	}
	return dag.ResolveResult{
		Value: getSettings(),
	}, nil
}

type menuPagesResolver struct{}

func (menuPagesResolver) Resolve(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
	deps := []dag.NodeKey{}
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		deps = make([]dag.NodeKey, 0, len(snapshot.publishedPages))
		for _, page := range snapshot.publishedPages {
			pageURL := strings.TrimSpace(page.URL)
			if pageURL == "" {
				continue
			}
			deps = append(deps, pageByURLNodeKey(pageURL))
		}
	}
	return dag.ResolveResult{
		Value: getPagesMenu(),
		Deps:  deps,
	}, nil
}

type postBySlugResolver struct{}

func (postBySlugResolver) Resolve(_ *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	post := getPostBySlugInLocale(key.ID, "")
	if post == nil {
		return dag.ResolveResult{}, nil
	}
	return dag.ResolveResult{
		Value: post,
	}, nil
}

type translationBySlugResolver struct{}

func (translationBySlugResolver) Resolve(_ *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	translation := getPostTranslationBySlugLocale(key.ID, key.Scope)
	if translation == nil {
		return dag.ResolveResult{}, nil
	}
	return dag.ResolveResult{
		Value: translation,
	}, nil
}

type postFamilyResolver struct{}

func (postFamilyResolver) Resolve(_ *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	settings := getSettings()
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		settings = snapshot.settings
	}
	source, translations := loadPostFamily(key.ID, nil, settings)
	deps := make([]dag.NodeKey, 0, 1+len(translations))
	if source != nil && strings.TrimSpace(source.Slug) != "" {
		deps = append(deps, postBySlugNodeKey("", strings.TrimSpace(source.Slug)))
	}
	for _, item := range translations {
		locale := normalizeLocale(item.Locale)
		if locale == "" || strings.TrimSpace(item.Slug) == "" {
			continue
		}
		deps = appendUniqueDAGNodeKey(deps, postBySlugNodeKey(locale, strings.TrimSpace(item.Slug)))
	}
	return dag.ResolveResult{
		Value: postFamilyValue{
			Source:       source,
			Translations: translations,
		},
		Deps: deps,
	}, nil
}

type adjacentPostsResolver struct{}

func (adjacentPostsResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	post, err := resolvePostNodeForLocale(ctx, key.Scope, key.ID)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	if post == nil {
		return dag.ResolveResult{}, nil
	}
	newer, older := getAdjacentPostsInLocale(post, localeScopeValue(key.Scope))
	deps := []dag.NodeKey{
		postBySlugNodeKey(localeScopeValue(key.Scope), key.ID),
	}
	for _, candidate := range []*PostRecord{newer, older} {
		if candidate == nil || strings.TrimSpace(candidate.Slug) == "" {
			continue
		}
		deps = append(deps, postBySlugNodeKey(localeScopeValue(key.Scope), candidate.Slug))
	}
	return dag.ResolveResult{
		Value: adjacentPostsValue{
			Newer: newer,
			Older: older,
		},
		Deps: deps,
	}, nil
}

type relatedPostsResolver struct{}

func (relatedPostsResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	post, err := resolvePostNodeForLocale(ctx, key.Scope, key.ID)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	if post == nil {
		return dag.ResolveResult{}, nil
	}
	related := getRelatedPostsInLocale(post, localeScopeValue(key.Scope), 4)
	return dag.ResolveResult{
		Value: related,
		Deps: []dag.NodeKey{
			postBySlugNodeKey(localeScopeValue(key.Scope), key.ID),
		},
	}, nil
}

type postRenderInputResolver struct{}

func (postRenderInputResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	locale := localeScopeValue(key.Scope)
	postDep := postBySlugNodeKey(locale, key.ID)
	settingsDep := settingsNodeKey()
	menuDep := menuPagesNodeKey()
	adjacentDep := adjacentPostsNodeKey(locale, key.ID)

	postValue, err := ctx.Resolve(postDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	settingsValue, err := ctx.Resolve(settingsDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	menuValue, err := ctx.Resolve(menuDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	adjacentValueRaw, err := ctx.Resolve(adjacentDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}

	settings, _ := settingsValue.(SettingsRecord)
	menu, _ := menuValue.([]PageRecord)
	adjacent, _ := adjacentValueRaw.(adjacentPostsValue)

	var (
		post         *PostRecord
		translation  *PostTranslationRecord
		family       postFamilyValue
		relatedPosts []PostRecord
		deps         = []dag.NodeKey{postDep, settingsDep, menuDep, adjacentDep}
	)

	switch value := postValue.(type) {
	case *PostRecord:
		post = value
		if post != nil && strings.TrimSpace(post.ID) != "" {
			familyDep := postFamilyNodeKey(post.ID)
			familyValue, err := ctx.Resolve(familyDep)
			if err != nil {
				return dag.ResolveResult{}, err
			}
			family, _ = familyValue.(postFamilyValue)
			deps = append(deps, familyDep)
		}
	case *PostTranslationRecord:
		translation = value
		if translation != nil {
			converted := translationToPost(*translation)
			post = &converted
			if sourceID := strings.TrimSpace(translation.SourcePost); sourceID != "" {
				familyDep := postFamilyNodeKey(sourceID)
				familyValue, err := ctx.Resolve(familyDep)
				if err != nil {
					return dag.ResolveResult{}, err
				}
				family, _ = familyValue.(postFamilyValue)
				deps = append(deps, familyDep)
			}
		}
	}

	if settings.ShowRelatedPosts {
		relatedDep := relatedPostsNodeKey(locale, key.ID)
		relatedValue, err := ctx.Resolve(relatedDep)
		if err != nil {
			return dag.ResolveResult{}, err
		}
		relatedPosts, _ = relatedValue.([]PostRecord)
		deps = append(deps, relatedDep)
	}

	input := &postRenderInput{
		path:        postRoutePath(locale, key.ID),
		locale:      locale,
		slug:        key.ID,
		post:        post,
		translation: translation,
	}

	return dag.ResolveResult{
		Value: postRenderInputValue{
			Input:       input,
			Settings:    settings,
			Menu:        menu,
			Family:      family,
			Adjacent:    adjacent,
			Related:     relatedPosts,
			CurrentPost: post,
		},
		Deps: deps,
	}, nil
}

type routeResolver struct{}

func (routeResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	if key.ID == "/" {
		inputDep := homeRenderInputNodeKey()
		inputValueRaw, err := ctx.Resolve(inputDep)
		if err != nil {
			return dag.ResolveResult{}, err
		}
		inputValue, _ := inputValueRaw.(homeRenderInputValue)
		return dag.ResolveResult{
			Value: routeValue{
				Path: key.ID,
				Body: []byte(renderHome(inputValue.Settings)),
			},
			Deps: []dag.NodeKey{inputDep},
		}, nil
	}

	if strings.HasPrefix(key.ID, "/archive") {
		inputDep := archiveRenderInputNodeKey(key.ID)
		inputValueRaw, err := ctx.Resolve(inputDep)
		if err != nil {
			return dag.ResolveResult{}, err
		}
		inputValue, _ := inputValueRaw.(archiveRenderInputValue)
		return dag.ResolveResult{
			Value: routeValue{
				Path: key.ID,
				Body: []byte(renderArchive(key.ID+"/", "", inputValue.Settings)),
			},
			Deps: []dag.NodeKey{inputDep},
		}, nil
	}

	locale, slug, ok := resolvePostPath(key.ID)
	if !ok {
		inputDep := pageRenderInputNodeKey(key.ID)
		inputValueRaw, err := ctx.Resolve(inputDep)
		if err != nil {
			return dag.ResolveResult{}, err
		}
		inputValue, _ := inputValueRaw.(pageRenderInputValue)
		if inputValue.Page == nil {
			return dag.ResolveResult{}, fmt.Errorf("%w: %s", errUnsupportedRouteNode, key.ID)
		}

		html, ok := renderPageFromRecord(inputValue.Page, inputValue.Settings)
		if !ok {
			return dag.ResolveResult{
				Value: routeValue{
					Path: key.ID,
					Body: []byte(renderNotFound(inputValue.Settings)),
				},
				Deps: []dag.NodeKey{inputDep},
			}, nil
		}

		return dag.ResolveResult{
			Value: routeValue{
				Path: key.ID,
				Body: []byte(html),
			},
			Deps: []dag.NodeKey{inputDep},
		}, nil
	}
	inputDep := postRenderInputNodeKey(locale, slug)
	inputValueRaw, err := ctx.Resolve(inputDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	inputValue, _ := inputValueRaw.(postRenderInputValue)

	html, ok := renderPostFromInput(inputValue.Input, inputValue.Settings)
	if !ok {
		return dag.ResolveResult{
			Value: routeValue{
				Path: key.ID,
				Body: []byte(renderNotFound(inputValue.Settings)),
			},
			Deps: []dag.NodeKey{inputDep},
		}, nil
	}

	return dag.ResolveResult{
		Value: routeValue{
			Path: key.ID,
			Body: []byte(html),
		},
		Deps: []dag.NodeKey{inputDep},
	}, nil
}

func resolvePostNodeForLocale(ctx *dag.ResolveContext, scope, slug string) (*PostRecord, error) {
	value, err := ctx.Resolve(postBySlugNodeKey(localeScopeValue(scope), slug))
	if err != nil {
		return nil, err
	}
	switch resolved := value.(type) {
	case *PostRecord:
		return resolved, nil
	case *PostTranslationRecord:
		if resolved == nil {
			return nil, nil
		}
		post := translationToPost(*resolved)
		return &post, nil
	default:
		return nil, nil
	}
}

func localeScopeValue(scope string) string {
	if scope == nodeScopeBase {
		return ""
	}
	return normalizeLocale(scope)
}

func postRoutePath(locale, slug string) string {
	if normalizeLocale(locale) == "" {
		return "/posts/" + slug + "/"
	}
	return "/" + normalizeLocale(locale) + "/posts/" + slug + "/"
}
