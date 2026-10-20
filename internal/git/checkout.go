package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	orcatui "github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

const previousBranchAlias = "-"

type CheckoutCommand struct {
	bus.Returns[bus.Void]

	Name        string
	AllProjects bool
	Workspace   string
	Project     string
	Pull        bool
	Rebase      bool
	Create      bool
}

func (CheckoutCommand) MessageName() string {
	return "g.co"
}

func (m *Manager) performCheckout(dir string, branch string, create bool) error {
	args := []string{"checkout"}

	if create {
		args = append(args, "-b")
	}

	args = append(args, branch)

	return m.exec.Exec("git", args, withDir(dir, process.WithHostIO())...)
}

// checkoutOrCreate checks out the branch matching name exactly, creating it if
// it doesn't already exist, locally or on a remote
func (m *Manager) checkoutOrCreate(dir string, name string) error {
	branches, err := m.searchBranches(searchBranchesOpts{
		Search:        name,
		Dir:           dir,
		IncludeRemote: true,
	})

	if err != nil {
		return err
	}

	existing := branches.Filter(func(b Branch) bool {
		return b.Name == name
	})

	if len(existing) == 0 {
		return m.performCheckout(dir, name, true)
	}

	if existing[0].Current {
		m.tui.Info(fmt.Sprintf("Branch is already checked out in '%s'!", dir))
		return nil
	}

	return m.performCheckout(dir, name, false)
}

func (m *Manager) checkoutFromDirectory(branch string, dir string) error {
	if err := m.mustBeInAGitRepository(dir); err != nil {
		return err
	}

	// Checkout the previously checked out branch, handle it as a special case
	if branch == previousBranchAlias {
		err := m.performCheckout(dir, previousBranchAlias, false)

		if err != nil {
			return err
		}

		return nil
	}

	branches, err := m.searchBranches(searchBranchesOpts{
		Search:        branch,
		Dir:           dir,
		IncludeRemote: true,
	})

	if err != nil {
		return err
	}

	if branches.GetCurrent() != nil && branches.GetCurrent().Name == branch {
		m.tui.Info("Branch is already checked out!")
		return nil
	}

	branches = branches.ExcludeCurrent()

	branchesLength := len(branches)

	if branchesLength == 0 {
		return usererr.Wrap(ErrNoBranchesFound, "No branches were found!")
	}

	if branchesLength == 1 {
		err := m.performCheckout(dir, branches[0].Name, false)

		return usererr.Wrap(err, "Failed to checkout branch")
	}

	chosen, err := m.tui.PresentChoices(branches.Names(), "Which branch?")

	if err != nil {
		if errors.Is(err, orcatui.ErrUserAbortedExecution) {
			return err
		}

		return usererr.Wrap(err, "Something went wrong, this is most likely a bug!")
	}

	err = m.performCheckout(dir, chosen, false)

	return usererr.Wrap(err, "Failed to checkout branch")
}

func (m *Manager) checkoutSingle(search string, execCtx workspace.ExecutionContext) error {
	return m.checkoutFromDirectory(search, execCtx.Dir())
}

func (m *Manager) chooseBranchesForAllProjects(search string, execCtx workspace.ExecutionContext) (map[string]Branch, error) {
	chosenBranches := map[string]Branch{}

	for _, p := range execCtx.Workspace.EnabledProjects() {
		branches, err := m.searchBranches(searchBranchesOpts{
			Search:        search,
			Dir:           p.ProjectDir,
			IncludeRemote: true,
		})

		if err != nil {
			return nil, err
		}

		if len(branches) == 1 {
			chosenBranches[p.ProjectDir] = branches[0]
			continue
		}

		chosen, err := m.tui.PresentChoices(branches.Names(), fmt.Sprintf("Which branch? (%s:%s)", p.Name, p.ProjectDir))

		if err != nil {
			if errors.Is(err, orcatui.ErrUserAbortedExecution) {
				return nil, err
			}

			return nil, err
		}

		chosenBranches[p.ProjectDir] = Branch{
			Name:    chosen,
			Current: false,
		}
	}

	return chosenBranches, nil
}

func (m *Manager) handleCreate(execCtx workspace.ExecutionContext, cmd CheckoutCommand) error {
	if cmd.Name == "" || cmd.Name == previousBranchAlias {
		return usererr.Wrap(
			fmt.Errorf("%w to 'co': an exact branch name must be supplied when creating a branch", workspace.ErrInvalidInput),
			"A branch name must be supplied when creating a branch!",
		)
	}

	if !cmd.AllProjects {
		return usererr.Wrap(m.checkoutOrCreate(execCtx.Dir(), cmd.Name), "Failed to checkout branch")
	}

	for _, p := range execCtx.Workspace.EnabledProjects() {
		if err := m.checkoutOrCreate(p.ProjectDir, cmd.Name); err != nil {
			return usererr.Wrap(err, "Checkout failed, you should check the branches for all projects as this failed mid-flow")
		}
	}

	return nil
}

func (m *Manager) handleCheckout(execCtx workspace.ExecutionContext, cmd CheckoutCommand) error {
	if cmd.Create {
		return m.handleCreate(execCtx, cmd)
	}

	if !cmd.AllProjects {
		return m.checkoutSingle(cmd.Name, execCtx)
	}

	if cmd.Name == previousBranchAlias {
		return usererr.Wrap(
			fmt.Errorf("%w to 'co': cannot use '%s' when checking out multiple projects", workspace.ErrInvalidInput, previousBranchAlias),
			fmt.Sprintf("Cannot use '%s' when checking out multiple projects!", previousBranchAlias),
		)
	}

	chosenBranches, err := m.chooseBranchesForAllProjects(cmd.Name, execCtx)

	if err != nil {
		return usererr.Wrap(err, "Something went wrong, this is most likely a bug!")
	}

	for dir, b := range chosenBranches {
		if b.Current {
			m.tui.Info(fmt.Sprintf("Skipping checkout in '%s', as the branch is already checked out...", dir))
			continue
		}

		err := m.performCheckout(dir, b.Name, false)

		if err != nil {
			return usererr.Wrap(err, "Checkout failed, you should check the branches for all projects as this failed mid-flow")
		}
	}

	return nil
}

func (m *Manager) Checkout(ctx context.Context, cmd CheckoutCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if err := m.handleCheckout(execCtx, cmd); err != nil {
		return usererr.Wrap(err, "Failed to checkout")
	}

	if !cmd.Pull {
		return nil
	}

	return usererr.Wrap(m.pullInContext(execCtx, cmd.AllProjects, cmd.Rebase), "Failed to pull")
}
