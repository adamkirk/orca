package stack

import (
	"context"
	"errors"
	"testing"

	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCompose only implements what the tests call, anything else panics.
type stubCompose struct {
	compose

	names []string
	err   error
}

func (c stubCompose) ServiceNames(*workspace.Project) ([]string, error) {
	return c.names, c.err
}

func Test_ServiceChoices(t *testing.T) {
	ws := &workspace.Workspace{Projects: []workspace.Project{{Name: "api"}}}
	inProject := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]})
	parseErr := errors.New("invalid compose file")

	tests := []struct {
		name    string
		ctx     context.Context
		compose stubCompose
		want    []workspace.Choice
		wantErr error
	}{
		{
			name:    "the project's services",
			ctx:     inProject,
			compose: stubCompose{names: []string{"app", "db"}},
			want:    []workspace.Choice{{Value: "app"}, {Value: "db"}},
		},
		{
			name:    "a whole workspace has no one compose file",
			ctx:     workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws}),
			wantErr: ErrNoProject,
		},
		{
			name:    "the compose file can't be read",
			ctx:     inProject,
			compose: stubCompose{err: parseErr},
			wantErr: parseErr,
		},
		{
			name:    "no execution context",
			ctx:     context.Background(),
			wantErr: workspace.ErrNoExecutionContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewManager(tt.compose, nil).ServiceChoices(tt.ctx, ServiceChoicesQuery{})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ExtensionChoices(t *testing.T) {
	ws := &workspace.Workspace{Projects: []workspace.Project{{
		Name: "api",
		Config: workspace.ProjectConfig{Extensions: []workspace.Extension{
			{Name: "artisan", Description: "Run an artisan command"},
			{Name: "test"},
		}},
	}}}

	tests := []struct {
		name    string
		ctx     context.Context
		want    []workspace.Choice
		wantErr error
	}{
		{
			name: "the project's extensions, with their descriptions",
			ctx:  workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]}),
			want: []workspace.Choice{
				{Value: "artisan", Description: "Run an artisan command"},
				{Value: "test"},
			},
		},
		{
			name:    "a whole workspace has no extensions",
			ctx:     workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws}),
			wantErr: ErrNoProject,
		},
		{
			name:    "no execution context",
			ctx:     context.Background(),
			wantErr: workspace.ErrNoExecutionContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewManager(stubCompose{}, nil).ExtensionChoices(tt.ctx, ExtensionChoicesQuery{})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
