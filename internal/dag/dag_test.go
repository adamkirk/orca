package dag

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type graphable struct {
	key      string
	children []string
	parents  []string
}

func (g graphable) GetKey() string {
	return g.key
}

func (g graphable) GetParents() []string {
	return g.parents
}

func (g graphable) GetChildren() []string {
	return g.children
}

type expectation struct {
	parents  []string
	children []string
}

type expectations map[string]expectation

func Test_ErrorMessages(t *testing.T) {
	errVertexAlreadyExists := ErrVertexAlreadyExists{
		Key: "blah",
	}

	assert.Equal(t, "vertex with key 'blah' already exists in graph", errVertexAlreadyExists.Error())

	errVertexNotFoundForEdge := ErrVertexNotFoundForEdge{
		Source:  "blah",
		Missing: "meh",
		Type:    "child",
	}

	assert.Equal(t, "cannot add child 'meh' to 'blah'", errVertexNotFoundForEdge.Error())
}

func Test_NewGraph(t *testing.T) {
	tests := []struct {
		name      string
		in        []graphable
		expectErr error
		expect    expectations
	}{
		{
			name: "graph with no edges",
			in: []graphable{
				{
					key:      "v1",
					parents:  []string{},
					children: []string{},
				},
				{
					key:      "v2",
					parents:  []string{},
					children: []string{},
				},
			},
			expect: expectations{
				"v1": {
					parents:  []string{},
					children: []string{},
				},
				"v2": {
					parents:  []string{},
					children: []string{},
				},
			},
		},
		{
			name: "graph with duplicate vertices",
			in: []graphable{
				{
					key:      "v1",
					parents:  []string{},
					children: []string{},
				},
				{
					key:      "v1",
					parents:  []string{},
					children: []string{},
				},
			},
			expectErr: ErrVertexAlreadyExists{
				Key: "v1",
			},
		},
		{
			name: "graph with with parent and children defined in graphable",
			in: []graphable{
				{
					key:     "v1",
					parents: []string{},
					children: []string{
						"v2",
					},
				},
				{
					key: "v2",
					parents: []string{
						"v1",
					},
					children: []string{},
				},
			},
			expect: expectations{
				"v1": {
					parents: []string{},
					children: []string{
						"v2",
					},
				},
				"v2": {
					parents: []string{
						"v1",
					},
					children: []string{},
				},
			},
		},
		{
			name: "graph with with only parents defined",
			in: []graphable{
				{
					key:      "v1",
					parents:  []string{},
					children: []string{},
				},
				{
					key: "v2",
					parents: []string{
						"v1",
					},
					children: []string{},
				},
			},
			expect: expectations{
				"v1": {
					parents: []string{},
					children: []string{
						"v2",
					},
				},
				"v2": {
					parents: []string{
						"v1",
					},
					children: []string{},
				},
			},
		},
		{
			name: "graph with with only children defined",
			in: []graphable{
				{
					key:     "v1",
					parents: []string{},
					children: []string{
						"v2",
					},
				},
				{
					key:      "v2",
					parents:  []string{},
					children: []string{},
				},
			},
			expect: expectations{
				"v1": {
					parents: []string{},
					children: []string{
						"v2",
					},
				},
				"v2": {
					parents: []string{
						"v1",
					},
					children: []string{},
				},
			},
		},
		{
			name: "adding edge to missing child vertex",
			in: []graphable{
				{
					key:     "v1",
					parents: []string{},
					children: []string{
						"v3",
					},
				},
				{
					key:      "v2",
					parents:  []string{},
					children: []string{},
				},
			},
			expectErr: ErrVertexNotFoundForEdge{
				Source:  "v1",
				Missing: "v3",
				Type:    "child",
			},
		},
		{
			name: "adding edge to missing parent vertex",
			in: []graphable{
				{
					key: "v1",
					parents: []string{
						"v3",
					},
					children: []string{},
				},
				{
					key:      "v2",
					parents:  []string{},
					children: []string{},
				},
			},
			expectErr: ErrVertexNotFoundForEdge{
				Source:  "v1",
				Missing: "v3",
				Type:    "parent",
			},
		},
		{
			name: "graph with many edges",
			in: []graphable{
				{
					key:      "v1",
					parents:  []string{},
					children: []string{},
				},
				{
					key:      "v2",
					parents:  []string{},
					children: []string{},
				},
				{
					key: "v3",
					parents: []string{
						"v1",
					},
					children: []string{},
				},
				{
					key: "v4",
					parents: []string{
						"v3",
					},
					children: []string{},
				},
				{
					key: "v5",
					parents: []string{
						"v2",
					},
					children: []string{},
				},
				{
					key: "v6",
					parents: []string{
						"v5",
						"v4",
					},
					children: []string{},
				},
			},
			expect: expectations{
				"v1": {
					parents: []string{},
					children: []string{
						"v3",
					},
				},
				"v2": {
					parents: []string{},
					children: []string{
						"v5",
					},
				},
				"v3": {
					parents: []string{
						"v1",
					},
					children: []string{
						"v4",
					},
				},
				"v4": {
					parents: []string{
						"v3",
					},
					children: []string{
						"v6",
					},
				},
				"v5": {
					parents: []string{
						"v2",
					},
					children: []string{
						"v6",
					},
				},
				"v6": {
					parents: []string{
						"v5",
						"v4",
					},
					children: []string{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			g, err := NewGraph(test.in)

			require.Equal(tt, test.expectErr, err)
			assertGraphMatchesExpectations(tt, g, test.expect)
		})
	}
}

func assertGraphMatchesExpectations(t *testing.T, g *Graph, expect expectations) {
	if expect == nil {
		return
	}

	for _, v := range g.vertices {
		e, ok := expect[v.Key]

		require.True(t, ok, "missing expectation for vertex found in graph '%s'", v.Key)

		for _, c := range v.Children {
			assert.Contains(t, e.children, c.Key, "vertex '%s', contains child '%s' that it should not", v.Key, c.Key)
		}
		for _, c := range e.children {
			_, ok := v.Children[c]
			assert.True(t, ok, "vertex '%s' did not contain expected child '%s'", v.Key, c)
		}

		for _, p := range v.Parents {
			assert.Contains(t, e.parents, p.Key, "vertex '%s', contains parent '%s' that it should not", v.Key, p.Key)
		}
		for _, p := range e.parents {
			_, ok := v.Parents[p]
			assert.True(t, ok, "vertex '%s' did not contain expected parent '%s'", v.Key, p)
		}
	}
}

func getComplexGraph() []graphable {
	return []graphable{
		{
			key:      "v1",
			parents:  []string{},
			children: []string{},
		},
		{
			key:      "v2",
			parents:  []string{},
			children: []string{},
		},
		{
			key: "v3",
			parents: []string{
				"v1",
			},
			children: []string{},
		},
		{
			key: "v4",
			parents: []string{
				"v3",
			},
			children: []string{},
		},
		{
			key: "v5",
			parents: []string{
				"v2",
			},
			children: []string{},
		},
		{
			key: "v6",
			parents: []string{
				"v5",
				"v4",
			},
			children: []string{},
		},
		{
			key:      "v7",
			parents:  []string{},
			children: []string{},
		},
	}
}

func Test_Graph_Leaves(t *testing.T) {
	g, err := NewGraph(getComplexGraph())

	require.Nil(t, err)

	leaves := g.Leaves()

	expectKeys := map[string]bool{
		"v6": false,
		"v7": false,
	}

	for _, l := range leaves {
		_, ok := expectKeys[l.Key]

		require.True(t, ok, "found unexpected leaf '%s'", l.Key)
		expectKeys[l.Key] = true
	}

	for k, ok := range expectKeys {
		assert.True(t, ok, "leaf '%s' was not found", k)
	}

	if t.Failed() {
		fmt.Println("[DEBUG] graph structure")
		for k, v := range g.vertices {
			fmt.Printf("%s => %#v\n", k, v)
		}
		fmt.Println("[DEBUG] leaves")
		for i, v := range leaves {
			fmt.Printf("%d => %#v\n", i, v)
		}
		fmt.Println()
	}
}

func Test_Graph_Roots(t *testing.T) {
	g, err := NewGraph(getComplexGraph())

	require.Nil(t, err)

	roots := g.Roots()

	expectKeys := map[string]bool{
		"v1": false,
		"v2": false,
		"v7": false,
	}

	for _, l := range roots {
		_, ok := expectKeys[l.Key]

		require.True(t, ok, "found unexpected root '%s'", l.Key)
		expectKeys[l.Key] = true
	}

	for k, ok := range expectKeys {
		assert.True(t, ok, "root '%s' was not found", k)
	}

	if t.Failed() {
		fmt.Println("[DEBUG] graph structure")
		for k, v := range g.vertices {
			fmt.Printf("%s => %#v\n", k, v)
		}
		fmt.Println("[DEBUG] roots")
		for i, v := range roots {
			fmt.Printf("%d => %#v\n", i, v)
		}
		fmt.Println()
	}
}

func Test_TopologicalKeysFromLeaves(t *testing.T) {
	g, err := NewGraph(getComplexGraph())

	require.Nil(t, err)

	// Store these for use later, we wanna check the graph itself wasn't modified
	// by getting the topological keys; it should create a new copy and perform
	// any operations on it
	roots := g.Roots()
	leaves := g.Leaves()

	keys, err := g.TopologicalKeysFromLeaves()

	require.Nil(t, err)

	assert.Equal(t, []string{
		"v6", "v7", "v4", "v5", "v2", "v3", "v1",
	}, keys)

	// Ensure these are still the same as before, the initial graph should be
	// untouched
	assert.Equal(t, roots, g.Roots())
	assert.Equal(t, leaves, g.Leaves())
}

func Test_TopologicalKeysFromRoots(t *testing.T) {
	g, err := NewGraph(getComplexGraph())

	require.Nil(t, err)

	// Store these for use later, we wanna check the graph itself wasn't modified
	// by getting the topological keys; it should create a new copy and perform
	// any operations on it
	roots := g.Roots()
	leaves := g.Leaves()

	keys, err := g.TopologicalKeysFromRoots()

	require.Nil(t, err)

	assert.Equal(t, []string{
		"v1", "v2", "v7", "v3", "v5", "v4", "v6",
	}, keys)

	// Ensure these are still the same as before, the initial graph should be
	// untouched
	assert.Equal(t, roots, g.Roots())
	assert.Equal(t, leaves, g.Leaves())
}
