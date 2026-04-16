package site

import "alleycat-backend/internal/dag"

type homeListingResolver struct{}

func (homeListingResolver) Resolve(_ *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		return dag.ResolveResult{
			Value: append([]PostRecord(nil), snapshot.publishedPosts...),
		}, nil
	}
	return dag.ResolveResult{
		Value: nil,
	}, nil
}

type archiveListingResolver struct{}

func (archiveListingResolver) Resolve(_ *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	route := parseArchiveRoute(key.ID)
	if snapshot := currentSnapshotBuildContext(); snapshot != nil {
		listing, ok := snapshot.archiveIndex[route.listingKey()]
		if !ok {
			return dag.ResolveResult{}, nil
		}
		return dag.ResolveResult{
			Value: listing,
		}, nil
	}
	return dag.ResolveResult{
		Value: archiveListing{},
	}, nil
}

type homeRenderInputResolver struct{}

func (homeRenderInputResolver) Resolve(ctx *dag.ResolveContext, _ dag.NodeKey) (dag.ResolveResult, error) {
	settingsDep := settingsNodeKey()
	menuDep := menuPagesNodeKey()
	listingDep := homeListingNodeKey()

	settingsValue, err := ctx.Resolve(settingsDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	menuValue, err := ctx.Resolve(menuDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	listingValue, err := ctx.Resolve(listingDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}

	settings, _ := settingsValue.(SettingsRecord)
	menu, _ := menuValue.([]PageRecord)
	listing, _ := listingValue.([]PostRecord)

	return dag.ResolveResult{
		Value: homeRenderInputValue{
			Settings: settings,
			Menu:     menu,
			Listing:  listing,
		},
		Deps: []dag.NodeKey{settingsDep, menuDep, listingDep},
	}, nil
}

type archiveRenderInputResolver struct{}

func (archiveRenderInputResolver) Resolve(ctx *dag.ResolveContext, key dag.NodeKey) (dag.ResolveResult, error) {
	settingsDep := settingsNodeKey()
	menuDep := menuPagesNodeKey()
	listingDep := archiveListingNodeKey(key.ID)

	settingsValue, err := ctx.Resolve(settingsDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	menuValue, err := ctx.Resolve(menuDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}
	listingValue, err := ctx.Resolve(listingDep)
	if err != nil {
		return dag.ResolveResult{}, err
	}

	settings, _ := settingsValue.(SettingsRecord)
	menu, _ := menuValue.([]PageRecord)
	listing, _ := listingValue.(archiveListing)

	return dag.ResolveResult{
		Value: archiveRenderInputValue{
			Path:     key.ID,
			Settings: settings,
			Menu:     menu,
			Listing:  listing,
		},
		Deps: []dag.NodeKey{settingsDep, menuDep, listingDep},
	}, nil
}
