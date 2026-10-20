package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func names(projects []Project) []string {
	n := make([]string, len(projects))

	for i, p := range projects {
		n[i] = p.Name
	}

	return n
}

func TestRunSet(t *testing.T) {
	tests := []struct {
		name         string
		projects     []Project
		wantNames    []string
		wantPulledIn []PulledInProject
	}{
		{
			name: "every project is enabled",
			projects: []Project{
				{Name: "local-env"},
				{Name: "api", Requires: []string{"local-env"}},
			},
			wantNames:    []string{"local-env", "api"},
			wantPulledIn: []PulledInProject{},
		},
		{
			name: "disabled projects are left out",
			projects: []Project{
				{Name: "local-env"},
				{Name: "api", Requires: []string{"local-env"}},
				{Name: "web", Requires: []string{"api"}, Disabled: true},
			},
			wantNames:    []string{"local-env", "api"},
			wantPulledIn: []PulledInProject{},
		},
		{
			name: "requirements are pulled in, even indirectly",
			projects: []Project{
				{Name: "db", Disabled: true},
				{Name: "local-env", Requires: []string{"db"}, Disabled: true},
				{Name: "api", Requires: []string{"local-env"}},
				{Name: "admin", Disabled: true},
			},
			wantNames: []string{"db", "local-env", "api"},
			wantPulledIn: []PulledInProject{
				{Name: "local-env", RequiredBy: "api"},
				{Name: "db", RequiredBy: "local-env"},
			},
		},
		{
			name: "everything disabled",
			projects: []Project{
				{Name: "api", Disabled: true},
			},
			wantNames:    []string{},
			wantPulledIn: []PulledInProject{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{Projects: tt.projects}

			projects, pulledIn := ws.RunSet()

			assert.Equal(t, tt.wantNames, names(projects))
			assert.Equal(t, tt.wantPulledIn, pulledIn)
		})
	}
}

func TestEnabledProjects(t *testing.T) {
	ws := &Workspace{Projects: []Project{{Name: "api"}, {Name: "web", Disabled: true}, {Name: "admin"}}}

	assert.Equal(t, []string{"api", "admin"}, names(ws.EnabledProjects()))
	assert.Len(t, ws.Projects, 3)
}

func TestTrimRequires(t *testing.T) {
	projects := []Project{
		{Name: "api", Requires: []string{"local-env", "db"}},
		{Name: "db"},
	}

	trimmed := TrimRequires(projects)

	assert.Equal(t, []string{"db"}, trimmed[0].Requires)
	assert.Equal(t, []string{"local-env", "db"}, projects[0].Requires)
}
