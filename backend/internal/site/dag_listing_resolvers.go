package site

import "alleycat-backend/internal/dag"

type homeRenderInputResolver struct{}

func (homeRenderInputResolver) Resolve(ctx *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
	settingsDep := settingsNodeKey()
	menuDep := menuPagesNodeKey()

	settingsValue, err := ctx.Resolve(settingsDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	menuValue, err := ctx.Resolve(menuDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}

	settings, _ := settingsValue.(SettingsRecord)
	menu, _ := menuValue.([]PageRecord)

	return dag.ResolveResult{
		Value: homeRenderInputValue{
			Settings: settings,
			Menu:     menu,
		},
		Deps: []dag.NodeKey{settingsDep, menuDep},
	}, nil
}

type archiveRenderInputResolver struct{}

func (archiveRenderInputResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	settingsDep := settingsNodeKey()
	menuDep := menuPagesNodeKey()

	settingsValue, err := ctx.Resolve(settingsDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	menuValue, err := ctx.Resolve(menuDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}

	settings, _ := settingsValue.(SettingsRecord)
	menu, _ := menuValue.([]PageRecord)

	return dag.ResolveResult{
		Value: archiveRenderInputValue{
			Path:     key.ID,
			Settings: settings,
			Menu:     menu,
		},
		Deps: []dag.NodeKey{settingsDep, menuDep},
	}, nil
}
