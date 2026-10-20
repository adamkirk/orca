package git

import (
	"context"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/workspace"
)

// BranchChoicesQuery lists the branches that can be checked out in the
// repository: local branches first, then those only on a remote, without their
// remote prefix. The current branch is left out.
type BranchChoicesQuery struct {
	bus.Returns[[]workspace.Choice]

	Workspace string
	Project   string
}

func (BranchChoicesQuery) MessageName() string {
	return "choices.branches"
}

func (m *Manager) BranchChoices(ctx context.Context, _ BranchChoicesQuery) ([]workspace.Choice, error) {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return nil, err
	}

	branches, err := m.searchBranches(searchBranchesOpts{
		Dir:           execCtx.Dir(),
		IncludeRemote: true,
	})

	if err != nil {
		return nil, err
	}

	branches = branches.ExcludeCurrent()
	choices := make([]workspace.Choice, len(branches))

	for i, b := range branches {
		choices[i] = workspace.Choice{Value: b.Name}
	}

	return choices, nil
}
