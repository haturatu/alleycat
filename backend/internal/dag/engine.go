package dag

import (
	"errors"
	"fmt"
)

var ErrResolverNotRegistered = errors.New("dag resolver not registered")

type CycleError struct {
	Key NodeKey
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("dag cycle detected while resolving %s", e.Key.String())
}

type Engine struct {
	resolvers map[NodeKind]Resolver
}

func NewEngine() *Engine {
	return &Engine{
		resolvers: map[NodeKind]Resolver{},
	}
}

func (e *Engine) Register(kind NodeKind, resolver Resolver) {
	e.resolvers[kind] = resolver
}

func (e *Engine) NewContext() *ResolveContext {
	return &ResolveContext{
		engine:    e,
		cache:     map[NodeKey]Value{},
		deps:      map[NodeKey][]NodeKey{},
		revDeps:   map[NodeKey][]NodeKey{},
		resolving: map[NodeKey]bool{},
	}
}

func (e *Engine) Resolve(ctx *ResolveContext, key NodeKey) (Value, error) {
	if ctx == nil {
		ctx = e.NewContext()
	}
	return ctx.Resolve(key)
}

type ResolveContext struct {
	engine    *Engine
	cache     map[NodeKey]Value
	deps      map[NodeKey][]NodeKey
	revDeps   map[NodeKey][]NodeKey
	resolving map[NodeKey]bool
}

func (ctx *ResolveContext) Resolve(key NodeKey) (Value, error) {
	if value, ok := ctx.cache[key]; ok {
		return value, nil
	}
	if ctx.resolving[key] {
		return nil, &CycleError{Key: key}
	}

	resolver, ok := ctx.engine.resolvers[key.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrResolverNotRegistered, key.Kind)
	}

	ctx.resolving[key] = true
	defer delete(ctx.resolving, key)

	result, err := resolver.Resolve(ctx, key)
	if err != nil {
		return nil, err
	}

	ctx.replaceDeps(key, result.Deps)
	for _, dep := range result.Deps {
		if _, err := ctx.Resolve(dep); err != nil {
			return nil, err
		}
	}

	ctx.cache[key] = result.Value
	return result.Value, nil
}

func (ctx *ResolveContext) replaceDeps(key NodeKey, deps []NodeKey) {
	if previous, ok := ctx.deps[key]; ok {
		for _, dep := range previous {
			ctx.revDeps[dep] = filterKeys(ctx.revDeps[dep], key)
		}
	}

	next := append([]NodeKey(nil), deps...)
	ctx.deps[key] = next
	for _, dep := range next {
		ctx.revDeps[dep] = appendUniqueKey(ctx.revDeps[dep], key)
	}
}

func (ctx *ResolveContext) Dependencies(key NodeKey) []NodeKey {
	return append([]NodeKey(nil), ctx.deps[key]...)
}

func (ctx *ResolveContext) ReverseDependencies(key NodeKey) []NodeKey {
	return append([]NodeKey(nil), ctx.revDeps[key]...)
}

func (ctx *ResolveContext) AffectedFrom(changed []NodeKey) []NodeKey {
	queue := append([]NodeKey(nil), changed...)
	seen := map[NodeKey]struct{}{}
	affected := make([]NodeKey, 0, len(changed))

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		for _, next := range ctx.revDeps[current] {
			queue = append(queue, next)
			affected = appendUniqueKey(affected, next)
		}
	}

	return affected
}

func appendUniqueKey(items []NodeKey, candidate NodeKey) []NodeKey {
	for _, item := range items {
		if item == candidate {
			return items
		}
	}
	return append(items, candidate)
}

func filterKeys(items []NodeKey, remove NodeKey) []NodeKey {
	if len(items) == 0 {
		return nil
	}
	filtered := items[:0]
	for _, item := range items {
		if item == remove {
			continue
		}
		filtered = append(filtered, item)
	}
	return append([]NodeKey(nil), filtered...)
}
