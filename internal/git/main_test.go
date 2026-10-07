package git

import (
	"os"
	"testing"

	"github.com/adamkirk/orca/internal/common"
	git_mocks "github.com/adamkirk/orca/tests/mocks/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_ResolveContext(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	resolved := common.ExecutionContext{
		Workspace:        &common.Workspace{Name: "acme"},
		WorkingDirectory: "/somewhere",
	}
	noWorkspace := common.ErrCouldNotDetermineWorkspace{Message: "nope"}
	otherErr := common.ErrUnknownWorkspace{Name: "missing"}

	// RecordIfError returns the error it's given
	expectRecorded := func(tui *git_mocks.MockTui, err error) {
		tui.EXPECT().RecordIfError(mock.Anything, err).Return(err)
	}

	tests := []struct {
		name         string
		project      string
		allProjects  bool
		resolveCtx   common.ExecutionContext
		resolveErr   error
		configureTui func(*git_mocks.MockTui)
		expectCtx    common.ExecutionContext
		expectErr    error
	}{
		{
			name:       "resolved context is used as is",
			resolveCtx: resolved,
			expectCtx:  resolved,
		},
		{
			name:       "falls back to the working directory without a workspace",
			resolveErr: noWorkspace,
			expectCtx:  common.ExecutionContext{WorkingDirectory: wd},
		},
		{
			name:        "requires a workspace for all projects",
			allProjects: true,
			resolveErr:  noWorkspace,
			configureTui: func(tui *git_mocks.MockTui) {
				expectRecorded(tui, noWorkspace)
			},
			expectErr: noWorkspace,
		},
		{
			name:       "requires a workspace for a specific project",
			project:    "api",
			resolveErr: noWorkspace,
			configureTui: func(tui *git_mocks.MockTui) {
				expectRecorded(tui, noWorkspace)
			},
			expectErr: noWorkspace,
		},
		{
			name:       "other errors are returned",
			resolveErr: otherErr,
			configureTui: func(tui *git_mocks.MockTui) {
				expectRecorded(tui, otherErr)
			},
			expectErr: otherErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := git_mocks.NewMockContextResolver(t)
			resolver.EXPECT().Resolve("", tt.project).Return(tt.resolveCtx, tt.resolveErr)

			// Any tui call not configured fails the test
			tui := git_mocks.NewMockTui(t)
			if tt.configureTui != nil {
				tt.configureTui(tui)
			}

			g := NewGit(nil, tui, resolver)

			ctx, err := g.resolveContext("", tt.project, tt.allProjects)

			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectCtx, ctx)
		})
	}
}
