package dag

import "fmt"

type NodeKind string

type NodeKey struct {
	Kind  NodeKind
	Scope string
	ID    string
	Extra string
}

func (k NodeKey) String() string {
	return fmt.Sprintf("%s|%s|%s|%s", k.Kind, k.Scope, k.ID, k.Extra)
}

type Value any

type ResolveResult struct {
	Value Value
	Deps  []NodeKey
}

type Resolver interface {
	Resolve(ctx *ResolveContext, key NodeKey) (ResolveResult, error)
}
