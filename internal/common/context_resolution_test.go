package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FindProjectForDir(t *testing.T) {
	parent := ProjectMeta{Name: "parent", WorkspaceName: "acme", Path: "/code/acme"}
	nested := ProjectMeta{Name: "nested", WorkspaceName: "acme", Path: "/code/acme/services/api"}
	sibling := ProjectMeta{Name: "sibling", WorkspaceName: "acme", Path: "/code/acme-tools"}

	tests := []struct {
		name     string
		dir      string
		projects []ProjectMeta
		expected *ProjectMeta
	}{
		{
			name:     "project directory itself",
			dir:      "/code/acme",
			projects: []ProjectMeta{parent},
			expected: &parent,
		},
		{
			name:     "subdirectory of a project",
			dir:      "/code/acme/docs",
			projects: []ProjectMeta{parent},
			expected: &parent,
		},
		{
			name:     "nested project wins when listed after its parent",
			dir:      "/code/acme/services/api/cmd",
			projects: []ProjectMeta{parent, nested},
			expected: &nested,
		},
		{
			name:     "nested project wins when listed before its parent",
			dir:      "/code/acme/services/api",
			projects: []ProjectMeta{nested, parent},
			expected: &nested,
		},
		{
			name:     "parent is used outside the nested project",
			dir:      "/code/acme/services",
			projects: []ProjectMeta{nested, parent},
			expected: &parent,
		},
		{
			name:     "a sibling sharing a name prefix doesn't match",
			dir:      "/code/acme-tools/bin",
			projects: []ProjectMeta{parent, sibling},
			expected: &sibling,
		},
		{
			name:     "a directory sharing just a name prefix isn't in the project",
			dir:      "/code/acme-other",
			projects: []ProjectMeta{parent},
			expected: nil,
		},
		{
			name:     "trailing slashes in paths are ignored",
			dir:      "/code/acme/services/api/",
			projects: []ProjectMeta{{Name: "parent", Path: "/code/acme/"}, {Name: "nested", Path: "/code/acme/services/api/"}},
			expected: &ProjectMeta{Name: "nested", Path: "/code/acme/services/api/"},
		},
		{
			name:     "outside every project",
			dir:      "/somewhere/else",
			projects: []ProjectMeta{parent, nested, sibling},
			expected: nil,
		},
		{
			name:     "no projects",
			dir:      "/code/acme",
			projects: []ProjectMeta{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, findProjectForDir(tt.dir, tt.projects))
		})
	}
}
