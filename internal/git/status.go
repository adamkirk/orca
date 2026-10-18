package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
)

type StatusQuery struct {
	bus.Returns[bus.Void]

	AllProjects bool
	Workspace   string
	Project     string
}

func (StatusQuery) MessageName() string {
	return "g.status"
}

type workingTreeChanges struct {
	Staged     int
	Unstaged   int
	Untracked  int
	Conflicted int
}

func (c workingTreeChanges) String() string {
	parts := []string{}

	for _, part := range []struct {
		count int
		label string
	}{
		{c.Conflicted, "conflicted"},
		{c.Staged, "staged"},
		{c.Unstaged, "unstaged"},
		{c.Untracked, "untracked"},
	} {
		if part.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", part.count, part.label))
		}
	}

	if len(parts) == 0 {
		return "clean"
	}

	return strings.Join(parts, "\n")
}

// parseStatusPorcelain counts changes from the output of
// `git status --porcelain`, where the first two characters of each line are
// the index (X) and working tree (Y) status. A file can be both staged and
// unstaged, e.g. "MM", in which case it's counted in both.
func parseStatusPorcelain(output string) workingTreeChanges {
	changes := workingTreeChanges{}

	for _, line := range strings.Split(output, "\n") {
		if len(line) < 2 {
			continue
		}

		x, y := line[0], line[1]

		switch {
		case x == '?' && y == '?':
			changes.Untracked++
		case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
			changes.Conflicted++
		default:
			if x != ' ' {
				changes.Staged++
			}

			if y != ' ' {
				changes.Unstaged++
			}
		}
	}

	return changes
}

func (m *Manager) getWorkingTreeChanges(dir string) (workingTreeChanges, error) {
	opt, stdout := process.WithStdout()

	err := m.exec.Exec("git", []string{
		"status",
		"--porcelain",
	}, withDir(dir, opt)...)

	if err != nil {
		return workingTreeChanges{}, err
	}

	return parseStatusPorcelain(stdout.String()), nil
}

// statusRow builds the table row for a single directory, describing anything
// that couldn't be determined in the cell rather than erroring, so that one bad
// project doesn't hide the others
func (m *Manager) statusRow(name string, dir string) []string {
	if !m.isInGitRepository(dir) {
		return []string{name, "(not a git repository)", "-", dir}
	}

	branch, err := m.GetCurrentBranch(dir)

	if err != nil {
		branch = "(unknown)"
	}

	changes := "-"

	if c, err := m.getWorkingTreeChanges(dir); err == nil {
		changes = c.String()
	}

	return []string{name, branch, changes, dir}
}

func (m *Manager) Status(ctx context.Context, query StatusQuery) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	rows := [][]string{}

	switch {
	case query.AllProjects:
		for _, p := range execCtx.Workspace.Projects {
			rows = append(rows, m.statusRow(p.Name, p.ProjectDir))
		}
	case execCtx.Project != nil:
		rows = append(rows, m.statusRow(execCtx.Project.Name, execCtx.Project.ProjectDir))
	default:
		rows = append(rows, m.statusRow("-", execCtx.WorkingDirectory))
	}

	m.tui.Table([]string{
		"Project",
		"Branch",
		"Changes",
		"Path",
	}, rows)

	return nil
}
