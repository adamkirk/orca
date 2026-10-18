package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
)

type Branch struct {
	Name    string
	Current bool
}
type Branches []Branch

func (branches Branches) GetCurrent() *Branch {
	for _, b := range branches {
		if b.Current {
			return &b
		}
	}

	return nil
}

func (branches Branches) Names() []string {
	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Name
	}

	return names
}

func (branches Branches) ExcludeCurrent() Branches {
	new := Branches{}

	for _, b := range branches {
		if b.Current {
			continue
		}

		new = append(new, b)
	}

	return new
}

type branchFilterFunc func(Branch) bool

func (branches Branches) Filter(f branchFilterFunc) Branches {
	new := Branches{}

	for _, b := range branches {
		if f(b) {
			new = append(new, b)
		}
	}

	return new
}

func (branches Branches) StrictSearch(search string) Branches {
	return branches.Filter(func(b Branch) bool {
		return strings.Contains(b.Name, search)
	})
}

type searchBranchesOpts struct {
	Search string
	// The directory to run the command in, defaults to the current directory
	Dir string
}

func parseGitBranchOutput(output string) Branches {
	lines := strings.Split(output, "\n")
	branches := Branches{}

	foundCurrent := false

	for _, line := range lines {
		if line == "" {
			continue
		}

		isCurrent := false

		if !foundCurrent {
			isCurrent = strings.HasPrefix(line, "*")

			if isCurrent {
				line = strings.TrimPrefix(line, "*")
				foundCurrent = true
			}
		}

		line = strings.TrimLeft(line, " ")

		branches = append(branches, Branch{
			Name:    line,
			Current: isCurrent,
		})
	}

	return branches
}

func (g *Git) searchBranches(dto searchBranchesOpts) (Branches, error) {
	if err := g.mustBeInAGitRepository(dto.Dir); err != nil {
		return nil, err
	}

	opt, stdout := process.WithStdout()
	err := g.exec.Exec("git", []string{
		"branch",
		"-l",
	}, withDir(dto.Dir, opt)...)

	if err != nil {
		return nil, workspace.ErrInvalidExecutionContext{
			Msg: "failed to execute 'git branch'",
		}
	}

	branches := parseGitBranchOutput(stdout.String())

	if dto.Search == "" {
		return branches, nil
	}

	return branches.StrictSearch(dto.Search), nil
}

type BranchesQuery struct {
	bus.Returns[bus.Void]

	Search string
}

func (BranchesQuery) MessageName() string {
	return "g.branches"
}

func (g *Git) ShowBranches(_ context.Context, query BranchesQuery) error {
	branches, err := g.searchBranches(searchBranchesOpts{
		Search: query.Search,
	})
	if err != nil {
		return g.tui.RecordIfError(
			"Failed to list branches!",
			err,
		)
	}

	if len(branches) == 0 {
		g.tui.Info("No matching branches found!")
		return nil
	}

	if b := branches.GetCurrent(); b != nil {
		g.tui.Success(fmt.Sprintf("[current] %s", b.Name))
		g.tui.NewLine()
	}

	for _, b := range branches.ExcludeCurrent() {
		g.tui.Info(fmt.Sprintf("%s", b.Name))
	}

	return nil
}

func (g *Git) GetCurrentBranch(dir string) (string, error) {
	branches, err := g.searchBranches(searchBranchesOpts{
		Dir: dir,
	})

	if err != nil {
		return "", err
	}

	if branches.GetCurrent() == nil {
		return "", ErrCouldNotDetermineCurrentBranch{}
	}

	return branches.GetCurrent().Name, nil
}

type PullBranchOpts struct {
	// The directory to run the command in, defaults to the current directory
	Dir string
	// Passes --rebase to git pull
	Rebase bool
}

func (g *Git) PullBranch(dto PullBranchOpts) error {
	if err := g.mustBeInAGitRepository(dto.Dir); err != nil {
		return err
	}

	current, err := g.GetCurrentBranch(dto.Dir)

	if err != nil {
		return err
	}

	args := []string{"pull"}

	if dto.Rebase {
		args = append(args, "--rebase")
	}

	args = append(args, "origin", current)

	err = g.exec.Exec("git", args, withDir(dto.Dir, process.WithHostIO())...)

	return g.tui.RecordIfError("Failed to pull branch!", err)
}
