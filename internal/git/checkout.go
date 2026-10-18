package git

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/hostsys"
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

func (g *Git) performCheckout(dir string, branch string, create bool) error {
	args := []string{"checkout"}

	if create {
		args = append(args, "-b")
	}

	args = append(args, branch)

	return g.exec.Exec("git", args, withDir(dir, hostsys.WithHostIO())...)
}

// checkoutOrCreate checks out the branch matching name exactly, creating it if
// it doesn't already exist
func (g *Git) checkoutOrCreate(dir string, name string) error {
	branches, err := g.searchBranches(SearchBranchesDTO{
		Search: name,
		Dir:    dir,
	})

	if err != nil {
		return err
	}

	existing := branches.Filter(func(b Branch) bool {
		return b.Name == name
	})

	if len(existing) == 0 {
		return g.performCheckout(dir, name, true)
	}

	if existing[0].Current {
		g.tui.Info(fmt.Sprintf("Branch is already checked out in '%s'!", dir))
		return nil
	}

	return g.performCheckout(dir, name, false)
}

func (g *Git) checkoutFromDirectory(branch string, dir string) error {
	if err := g.mustBeInAGitRepository(dir); err != nil {
		return err
	}

	// Checkout the previously checked out branch, handle it as a special case
	if branch == previousBranchAlias {
		err := g.performCheckout(dir, previousBranchAlias, false)

		if err != nil {
			return err
		}

		return nil
	}

	branches, err := g.searchBranches(SearchBranchesDTO{
		Search: branch,
		Dir:    dir,
	})

	if err != nil {
		return err
	}

	if branches.GetCurrent() != nil && branches.GetCurrent().Name == branch {
		g.tui.Info("Branch is already checked out!")
		return nil
	}

	branches = branches.ExcludeCurrent()

	branchesLength := len(branches)

	if branchesLength == 0 {
		g.tui.Error("No branches were found!")
		return common.ErrNoBranchesFound{}
	}

	if branchesLength == 1 {
		err := g.performCheckout(dir, branches[0].Name, false)

		return g.tui.RecordIfError("Failed to checkout branch", err)
	}

	chosen, err := g.tui.PresentChoices(branches.Names(), "Which branch?")

	if err != nil {
		if _, ok := err.(common.ErrUserAbortedExecution); ok {
			return err
		}

		return g.tui.RecordIfError("Something went wrong, this is most likely a bug!", err)
	}

	err = g.performCheckout(dir, chosen, false)

	return g.tui.RecordIfError("Failed to checkout branch", err)
}

func (g *Git) checkoutSingle(search string, execCtx common.ExecutionContext) error {
	return g.checkoutFromDirectory(search, execCtx.Dir())
}

func (g *Git) chooseBranchesForAllProjects(search string, execCtx common.ExecutionContext) (map[string]Branch, error) {
	chosenBranches := map[string]Branch{}

	for _, p := range execCtx.Workspace.Projects {
		branches, err := g.searchBranches(SearchBranchesDTO{
			Search: search,
			Dir:    p.ProjectDir,
		})

		if err != nil {
			return nil, err
		}

		if len(branches) == 1 {
			chosenBranches[p.ProjectDir] = branches[0]
			continue
		}

		chosen, err := g.tui.PresentChoices(branches.Names(), fmt.Sprintf("Which branch? (%s:%s)", p.Name, p.ProjectDir))

		if err != nil {
			if _, ok := err.(common.ErrUserAbortedExecution); ok {
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

func (g *Git) handleCreate(execCtx common.ExecutionContext, cmd CheckoutCommand) error {
	if cmd.Name == "" || cmd.Name == previousBranchAlias {
		return g.tui.RecordIfError("A branch name must be supplied when creating a branch!", common.ErrInvalidInput{
			To:  "co",
			Msg: "an exact branch name must be supplied when creating a branch",
		})
	}

	if !cmd.AllProjects {
		return g.tui.RecordIfError("Failed to checkout branch", g.checkoutOrCreate(execCtx.Dir(), cmd.Name))
	}

	for _, p := range execCtx.Workspace.Projects {
		if err := g.checkoutOrCreate(p.ProjectDir, cmd.Name); err != nil {
			return g.tui.RecordIfError("Checkout failed, you should check the branches for all projects as this failed mid-flow", err)
		}
	}

	return nil
}

func (g *Git) handleCheckout(execCtx common.ExecutionContext, cmd CheckoutCommand) error {
	if cmd.Create {
		return g.handleCreate(execCtx, cmd)
	}

	if !cmd.AllProjects {
		return g.checkoutSingle(cmd.Name, execCtx)
	}

	if cmd.Name == previousBranchAlias {
		return g.tui.RecordIfError(fmt.Sprintf("Cannot use '%s' when checking out multiple projects!", previousBranchAlias), common.ErrInvalidInput{
			To:  "co",
			Msg: fmt.Sprintf("cannot use '%s' when checking out multiple projects", previousBranchAlias),
		})
	}

	chosenBranches, err := g.chooseBranchesForAllProjects(cmd.Name, execCtx)

	if err != nil {
		return g.tui.RecordIfError("Something went wrong, this is most likely a bug!", err)
	}

	for dir, b := range chosenBranches {
		if b.Current {
			g.tui.Info(fmt.Sprintf("Skipping checkout in '%s', as the branch is already checked out...", dir))
			continue
		}

		err := g.performCheckout(dir, b.Name, false)

		if err != nil {
			return g.tui.RecordIfError("Checkout failed, you should check the branches for all projects as this failed mid-flow", err)
		}
	}

	return nil
}

func (g *Git) Checkout(ctx context.Context, cmd CheckoutCommand) error {
	execCtx, err := common.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	if err := g.handleCheckout(execCtx, cmd); err != nil {
		return g.tui.RecordIfError("Failed to checkout", err)
	}

	if !cmd.Pull {
		return nil
	}

	return g.tui.RecordIfError("Failed to pull", g.pullInContext(execCtx, cmd.AllProjects, cmd.Rebase))
}
