package dag

import (
	"fmt"
	"slices"
	"strings"
)

type ErrVertexAlreadyExists struct {
	Key string
}

func (err ErrVertexAlreadyExists) Error() string {
	return fmt.Sprintf("vertex with key '%s' already exists in graph", err.Key)
}

type ErrVertexNotFoundForEdge struct {
	Source  string
	Missing string
	Type    string
}

func (err ErrVertexNotFoundForEdge) Error() string {
	return fmt.Sprintf("cannot add %s '%s' to '%s'", err.Type, err.Missing, err.Source)
}

type Graphable interface {
	GetKey() string
	GetChildren() []string
	GetParents() []string
}

type Vertices map[string]*Vertex

type Vertex struct {
	Key      string
	Children Vertices
	Parents  Vertices
}

func (v *Vertex) AddChild(key string, child *Vertex) {
	if v.HasChild(key) {
		return
	}

	v.Children[key] = child
}

func (v *Vertex) RemoveChildIfExists(key string) {
	if !v.HasChild(key) {
		return
	}

	delete(v.Children, key)
}

func (v *Vertex) HasChild(key string) bool {
	_, ok := v.Children[key]

	return ok
}

func (v *Vertex) AddParent(key string, parent *Vertex) {
	if v.HasParent(key) {
		return
	}

	v.Parents[key] = parent
}

func (v *Vertex) RemoveParentIfExists(key string) {
	if !v.HasParent(key) {
		return
	}

	delete(v.Parents, key)
}

func (v *Vertex) HasParent(key string) bool {
	_, ok := v.Parents[key]

	return ok
}

type Graph struct {
	vertices Vertices
}

func (g *Graph) GetVertex(key string) *Vertex {
	v, ok := g.vertices[key]

	if !ok {
		return nil
	}

	return v
}

func (g *Graph) AddVertex(key string) error {
	if g.GetVertex(key) != nil {
		return ErrVertexAlreadyExists{
			Key: key,
		}
	}

	g.vertices[key] = &Vertex{
		Key:      key,
		Children: Vertices{},
		Parents:  Vertices{},
	}

	return nil
}

func (g *Graph) RemoveVertex(key string) {
	delete(g.vertices, key)

	for _, other := range g.vertices {
		other.RemoveChildIfExists(key)
		other.RemoveParentIfExists(key)
	}
}

// AddEdge adds an edge between 2 vertices in the graph. 'from' is the parent,
// and 'to' is the child.
// It fills in both sides, so we'll add a child to the parent 'from' pointing
// at 'to'. And we'll add a parent to the child 'to' pointing at 'from'.
// TODO: Check for cycles here to ensure its acylic.
func (g *Graph) AddEdge(from string, to string) error {
	source := g.GetVertex(from)
	destination := g.GetVertex(to)

	if source == nil {
		return ErrVertexNotFoundForEdge{
			Missing: from,
			Source:  to,
			Type:    "parent",
		}
	}

	if destination == nil {
		return ErrVertexNotFoundForEdge{
			Missing: to,
			Source:  from,
			Type:    "child",
		}
	}

	source.AddChild(to, destination)

	destination.AddParent(from, source)

	return nil
}

func (g *Graph) Leaves() []*Vertex {
	leaves := []*Vertex{}

	for _, v := range g.vertices {
		if len(v.Children) == 0 {
			leaves = append(leaves, v)
		}
	}

	sortVertexSlice(leaves)

	return leaves
}

func sortVertexSlice(vertices []*Vertex) {
	slices.SortFunc(vertices, func(a *Vertex, b *Vertex) int {
		return strings.Compare(a.Key, b.Key)
	})
}

func (g *Graph) Roots() []*Vertex {
	roots := []*Vertex{}

	for _, v := range g.vertices {
		if len(v.Parents) == 0 {
			roots = append(roots, v)
		}
	}

	sortVertexSlice(roots)

	return roots
}

func (g *Graph) clone() (*Graph, error) {
	new := &Graph{
		vertices: Vertices{},
	}

	for k := range g.vertices {
		if err := new.AddVertex(k); err != nil {
			return nil, err
		}
	}

	for _, v := range g.vertices {
		for k := range v.Children {
			if err := new.AddEdge(v.Key, k); err != nil {
				return nil, err
			}
		}

		for k := range v.Parents {
			if err := new.AddEdge(k, v.Key); err != nil {
				return nil, err
			}
		}
	}

	return new, nil
}

func (g *Graph) pluckLeaves() []*Vertex {
	leaves := g.Leaves()

	for _, l := range leaves {
		g.RemoveVertex(l.Key)
	}

	return leaves
}

func (g *Graph) pruneRoots() []*Vertex {
	roots := g.Roots()

	for _, r := range roots {
		g.RemoveVertex(r.Key)
	}

	return roots
}

func (g *Graph) TopologicalKeysFromLeaves() ([]string, error) {
	new, err := g.clone()
	if err != nil {
		return nil, err
	}

	sorted := []*Vertex{}

	for {
		leaves := new.pluckLeaves()

		if len(leaves) == 0 {
			break
		}

		sorted = append(sorted, leaves...)
	}

	keys := make([]string, len(sorted))

	for i, v := range sorted {
		keys[i] = v.Key
	}

	return keys, nil
}

func (g *Graph) TopologicalKeysFromRoots() ([]string, error) {
	new, err := g.clone()
	if err != nil {
		return nil, err
	}

	sorted := []*Vertex{}

	for {
		roots := new.pruneRoots()

		if len(roots) == 0 {
			break
		}

		sorted = append(sorted, roots...)
	}

	keys := make([]string, len(sorted))

	for i, v := range sorted {
		keys[i] = v.Key
	}

	return keys, nil
}

func NewGraph[T Graphable](vertices []T) (*Graph, error) {
	g := &Graph{
		vertices: Vertices{},
	}

	for _, v := range vertices {
		if err := g.AddVertex(v.GetKey()); err != nil {
			return nil, err
		}
	}

	// This is possibly a bad idea...
	// Makes it much easier to build the graph within here, but does mean that
	// we're probably doing iterations we don't actually need. By simply looping
	// through the children we should've added all parents, but depending on how
	// the Graphable works, they migh have links to parents, children, or both.
	// So we kinda need to do both the inner loops.
	// But it's more likely that one of them will be empty anyway so no extra
	// iterations...
	for _, v := range vertices {
		for _, c := range v.GetChildren() {
			if err := g.AddEdge(v.GetKey(), c); err != nil {
				return nil, err
			}
		}

		for _, p := range v.GetParents() {
			if err := g.AddEdge(p, v.GetKey()); err != nil {
				return nil, err
			}
		}
	}

	return g, nil
}
