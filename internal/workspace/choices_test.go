package workspace

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubConfig only implements what the tests call, anything else panics.
type stubConfig struct {
	config

	workspaces []WorkspaceMeta
}

func (c stubConfig) GetAllWorkspaceMeta() []WorkspaceMeta {
	return c.workspaces
}

func TestWorkspaceChoices(t *testing.T) {
	tests := []struct {
		name       string
		workspaces []WorkspaceMeta
		want       []Choice
	}{
		{
			name: "every registered workspace",
			workspaces: []WorkspaceMeta{
				{Name: "acme", Path: "/code/acme"},
				{Name: "other", Path: "/code/other"},
			},
			want: []Choice{{Value: "acme"}, {Value: "other"}},
		},
		{
			name: "no workspaces",
			want: []Choice{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(afero.NewMemMapFs(), stubTUI{}, stubConfig{workspaces: tt.workspaces}, nil, nil, nil)

			got, err := m.WorkspaceChoices(context.Background(), WorkspaceChoicesQuery{})

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProjectChoices(t *testing.T) {
	m := NewManager(afero.NewMemMapFs(), stubTUI{}, nil, nil, nil, nil)

	t.Run("the projects in the context's workspace", func(t *testing.T) {
		ws := &Workspace{Projects: []Project{{Name: "api"}, {Name: "web"}}}
		ctx := WithExecutionContext(context.Background(), ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})

		got, err := m.ProjectChoices(ctx, ProjectChoicesQuery{})

		require.NoError(t, err)
		assert.Equal(t, []Choice{{Value: "api"}, {Value: "web"}}, got)
	})

	t.Run("without an execution context", func(t *testing.T) {
		_, err := m.ProjectChoices(context.Background(), ProjectChoicesQuery{})

		assert.ErrorIs(t, err, ErrNoExecutionContext)
	})
}
