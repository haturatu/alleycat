package dag

import (
	"errors"
	"testing"
)

type staticResolver struct {
	result ResolveResult
	err    error
}

func (r staticResolver) Resolve(_ *ResolveContext, _ NodeKey) (ResolveResult, error) {
	if r.err != nil {
		return ResolveResult{}, r.err
	}
	return r.result, nil
}

func TestResolveBuildsDependencyIndexes(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	source := NodeKey{Kind: "source", ID: "a"}
	derived := NodeKey{Kind: "derived", ID: "b"}
	route := NodeKey{Kind: "route", ID: "/posts/a/"}

	engine.Register(source.Kind, staticResolver{
		result: ResolveResult{Value: "source"},
	})
	engine.Register(derived.Kind, staticResolver{
		result: ResolveResult{
			Value: "derived",
			Deps:  []NodeKey{source},
		},
	})
	engine.Register(route.Kind, staticResolver{
		result: ResolveResult{
			Value: "route",
			Deps:  []NodeKey{derived},
		},
	})

	ctx := engine.NewContext()
	value, err := ctx.Resolve(route)
	if err != nil {
		t.Fatalf("Resolve(route): %v", err)
	}
	if value != "route" {
		t.Fatalf("Resolve(route) value = %#v", value)
	}

	if deps := ctx.Dependencies(route); len(deps) != 1 || deps[0] != derived {
		t.Fatalf("Dependencies(route) = %#v", deps)
	}
	if deps := ctx.Dependencies(derived); len(deps) != 1 || deps[0] != source {
		t.Fatalf("Dependencies(derived) = %#v", deps)
	}

	reverse := ctx.ReverseDependencies(source)
	if len(reverse) != 1 || reverse[0] != derived {
		t.Fatalf("ReverseDependencies(source) = %#v", reverse)
	}

	affected := ctx.AffectedFrom([]NodeKey{source})
	if len(affected) != 2 {
		t.Fatalf("AffectedFrom(source) length = %d, want 2 (%#v)", len(affected), affected)
	}
	if affected[0] != derived && affected[1] != derived {
		t.Fatalf("AffectedFrom(source) missing derived: %#v", affected)
	}
	if affected[0] != route && affected[1] != route {
		t.Fatalf("AffectedFrom(source) missing route: %#v", affected)
	}

	path := ctx.ExplainPath(source, route)
	if len(path) != 3 || path[0] != source || path[1] != derived || path[2] != route {
		t.Fatalf("ExplainPath(source, route) = %#v", path)
	}
}

func TestResolveReturnsCycleError(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	a := NodeKey{Kind: "node", ID: "a"}
	b := NodeKey{Kind: "node", ID: "b"}

	engine.Register(a.Kind, ResolverFunc(func(_ *ResolveContext, key NodeKey) (ResolveResult, error) {
		switch key.ID {
		case "a":
			return ResolveResult{
				Value: "a",
				Deps:  []NodeKey{b},
			}, nil
		case "b":
			return ResolveResult{
				Value: "b",
				Deps:  []NodeKey{a},
			}, nil
		default:
			return ResolveResult{}, nil
		}
	}))

	ctx := engine.NewContext()
	_, err := ctx.Resolve(a)
	if err == nil {
		t.Fatal("Resolve(a) error = nil, want cycle error")
	}
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("Resolve(a) error = %v, want CycleError", err)
	}
}

func TestResolveReturnsResolverNotRegistered(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	ctx := engine.NewContext()
	_, err := ctx.Resolve(NodeKey{Kind: "missing", ID: "x"})
	if err == nil {
		t.Fatal("Resolve(missing) error = nil")
	}
	if !errors.Is(err, ErrResolverNotRegistered) {
		t.Fatalf("Resolve(missing) error = %v, want ErrResolverNotRegistered", err)
	}
}

func TestExplainPathReturnsNilWhenUnreachable(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	ctx := engine.NewContext()

	source := NodeKey{Kind: "source", ID: "a"}
	target := NodeKey{Kind: "route", ID: "/posts/a/"}
	if path := ctx.ExplainPath(source, target); path != nil {
		t.Fatalf("ExplainPath(source, target) = %#v, want nil", path)
	}
}

type ResolverFunc func(ctx *ResolveContext, key NodeKey) (ResolveResult, error)

func (f ResolverFunc) Resolve(ctx *ResolveContext, key NodeKey) (ResolveResult, error) {
	return f(ctx, key)
}
