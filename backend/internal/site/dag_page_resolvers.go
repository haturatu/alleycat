package site

import (
	"strings"

	"alleycat-backend/internal/dag"
)

type pageByURLResolver struct{}

func (pageByURLResolver) Resolve(_ *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	page := getPageByURL(key.ID)
	if page == nil && !strings.HasSuffix(key.ID, "/") {
		page = getPageByURL(key.ID + "/")
	}
	if page == nil {
		return dag.ResolveResult{}, nil
	}
	return dag.ResolveResult{
		Value: page,
	}, nil
}

type pageRenderInputResolver struct{}

func (pageRenderInputResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	pageDep := pageByURLNodeKey(key.ID)
	settingsDep := settingsNodeKey()
	menuDep := menuPagesNodeKey()

	pageValue, err := ctx.Resolve(pageDep)
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

	page, _ := pageValue.(*PageRecord)
	settings, _ := settingsValue.(SettingsRecord)
	menu, _ := menuValue.([]PageRecord)

	return dag.ResolveResult{
		Value: pageRenderInputValue{
			Page:     page,
			Settings: settings,
			Menu:     menu,
		},
		Deps: []dag.NodeKey{pageDep, settingsDep, menuDep},
	}, nil
}
