package extension

import (
	"context"
	"testing"

	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Choices(t *testing.T) {
	ws := &workspace.Workspace{
		Projects: []workspace.Project{{
			Name: "api",
			Config: workspace.ProjectConfig{Extensions: []workspace.Extension{
				{Name: "artisan", Description: "Run an artisan command"},
				{Name: "install"},
			}},
		}},
	}

	tests := []struct {
		name    string
		ctx     context.Context
		want    []workspace.Choice
		wantErr error
	}{
		{
			name: "the project's, with their descriptions",
			ctx:  workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws, Project: &ws.Projects[0]}),
			want: []workspace.Choice{
				{Value: "artisan", Description: "Run an artisan command"},
				{Value: "install"},
			},
		},
		{
			name: "none outside of a project",
			ctx:  workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: ws}),
			want: []workspace.Choice{},
		},
		{
			name:    "no execution context",
			ctx:     context.Background(),
			wantErr: workspace.ErrNoExecutionContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewManager(nil, nil, nil, nil, nil).Choices(tt.ctx, ChoicesQuery{})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
