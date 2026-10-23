package dag

import "testing"

func TestNewContextWithDependenciesCopiesAndDeduplicatesEdges(t *testing.T) {
	source := NodeKey{Kind: "source", ID: "a"}
	route := NodeKey{Kind: "route", ID: "/"}
	deps := map[NodeKey][]NodeKey{route: {source, source}}
	ctx := NewEngine().NewContextWithDependencies(deps)
	deps[route][0] = NodeKey{Kind: "changed"}
	delete(deps, route)
	if dependencies := ctx.Dependencies(route); len(dependencies) != 1 || dependencies[0] != source {
		t.Fatalf("dependencies were not copied and deduplicated: %v", dependencies)
	}
	if affected := ctx.AffectedFrom([]NodeKey{source}); len(affected) != 1 || affected[0] != route {
		t.Fatalf("affected routes = %v, want [%v]", affected, route)
	}
	if len(ctx.cache) != 0 {
		t.Fatal("building a dependency graph cached resolved values")
	}
}
