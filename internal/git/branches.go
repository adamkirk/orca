package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

type Branch struct {
	Name    string
	Current bool
	// Only on a remote, with no local branch yet. Name has no remote prefix
	// (feature/x, not origin/feature/x), as checking it out by that name
	// creates a local branch tracking it.
	RemoteOnly bool
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

// Search returns the branches whose names contain search, ignoring case.
func (branches Branches) Search(search string) Branches {
	search = strings.ToLower(search)

	return branches.Filter(func(b Branch) bool {
		return strings.Contains(strings.ToLower(b.Name), search)
	})
}

type searchBranchesOpts struct {
	Search string
	// The directory to run the command in, defaults to the current directory
	Dir string
	// Also search branches that are only on a remote, after the local ones
	IncludeRemote bool
}

// remoteBranchesFormat lists remote branches without the refs/remotes/<remote>/
// prefix, so feature/x rather than origin/feature/x.
const remoteBranchesFormat = "--format=%(refname:lstrip=3)"

// parseRemoteBranches returns the remote branches in the output of
// remoteBranchesFormat that aren't also local, once each, in the order given.
func parseRemoteBranches(output string, local Branches) Branches {
	seen := map[string]bool{}

	for _, b := range local {
		seen[b.Name] = true
	}

	branches := Branches{}

	for _, name := range strings.Split(output, "\n") {
		name = strings.TrimSpace(name)

		// HEAD points to the remote's default branch, which is listed anyway
		if name == "" || name == "HEAD" || seen[name] {
			continue
		}

		seen[name] = true

		branches = append(branches, Branch{Name: name, RemoteOnly: true})
	}

	return branches
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

func (m *Manager) searchBranches(dto searchBranchesOpts) (Branches, error) {
	if err := m.mustBeInAGitRepository(dto.Dir); err != nil {
		return nil, err
	}

	opt, stdout := process.WithStdout()
	err := m.exec.Exec("git", []string{
		"branch",
		"-l",
	}, withDir(dto.Dir, opt)...)

	if err != nil {
		return nil, fmt.Errorf("%w: %s", workspace.ErrInvalidExecutionContext, "failed to execute 'git branch'")
	}

	branches := parseGitBranchOutput(stdout.String())

	if dto.IncludeRemote {
		remote, err := m.remoteOnlyBranches(dto.Dir, branches)

		if err != nil {
			return nil, err
		}

		branches = append(branches, remote...)
	}

	if dto.Search == "" {
		return branches, nil
	}

	return branches.Search(dto.Search), nil
}

// remoteOnlyBranches lists the branches that are only on a remote, from git's
// local refs, so it doesn't fetch. A branch pushed since the last fetch won't
// be listed.
func (m *Manager) remoteOnlyBranches(dir string, local Branches) (Branches, error) {
	opt, stdout := process.WithStdout()

	err := m.exec.Exec("git", []string{
		"for-each-ref",
		remoteBranchesFormat,
		"refs/remotes",
	}, withDir(dir, opt)...)

	if err != nil {
		return nil, fmt.Errorf("%w: %s", workspace.ErrInvalidExecutionContext, "failed to execute 'git for-each-ref'")
	}

	return parseRemoteBranches(stdout.String(), local), nil
}

type BranchesQuery struct {
	bus.Returns[bus.Void]

	Search string
}

func (BranchesQuery) MessageName() string {
	return "g.branches"
}

func (m *Manager) ShowBranches(_ context.Context, query BranchesQuery) error {
	branches, err := m.searchBranches(searchBranchesOpts{
		Search: query.Search,
	})
	if err != nil {
		return usererr.Wrap(
			err,
			"Failed to list branches!",
		)
	}

	if len(branches) == 0 {
		m.tui.Info("No matching branches found!")
		return nil
	}

	if b := branches.GetCurrent(); b != nil {
		m.tui.Success(fmt.Sprintf("[current] %s", b.Name))
		m.tui.NewLine()
	}

	for _, b := range branches.ExcludeCurrent() {
		m.tui.Info(fmt.Sprintf("%s", b.Name))
	}

	return nil
}

func (m *Manager) GetCurrentBranch(dir string) (string, error) {
	branches, err := m.searchBranches(searchBranchesOpts{
		Dir: dir,
	})

	if err != nil {
		return "", err
	}

	if branches.GetCurrent() == nil {
		return "", ErrCouldNotDetermineCurrentBranch
	}

	return branches.GetCurrent().Name, nil
}

type PullBranchOpts struct {
	// The directory to run the command in, defaults to the current directory
	Dir string
	// Passes --rebase to git pull
	Rebase bool
}

func (m *Manager) PullBranch(dto PullBranchOpts) error {
	if err := m.mustBeInAGitRepository(dto.Dir); err != nil {
		return err
	}

	current, err := m.GetCurrentBranch(dto.Dir)

	if err != nil {
		return err
	}

	args := []string{"pull"}

	if dto.Rebase {
		args = append(args, "--rebase")
	}

	args = append(args, "origin", current)

	err = m.exec.Exec("git", args, withDir(dto.Dir, process.WithHostIO())...)

	return usererr.Wrap(err, "Failed to pull branch!")
}
