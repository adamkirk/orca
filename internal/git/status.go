package git

import (
	"fmt"
	"strings"

	"github.com/adamkirk/orca/internal/hostsys"
)

type StatusDTO struct {
	AllProjects bool
	Workspace   string
	Project     string
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

func (g *Git) getWorkingTreeChanges(dir string) (workingTreeChanges, error) {
	opt, stdout := hostsys.WithStdout()

	err := g.exec.Exec("git", []string{
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
func (g *Git) statusRow(name string, dir string) []string {
	if !g.isInGitRepository(dir) {
		return []string{name, "(not a git repository)", "-", dir}
	}

	branch, err := g.GetCurrentBranch(dir)

	if err != nil {
		branch = "(unknown)"
	}

	changes := "-"

	if c, err := g.getWorkingTreeChanges(dir); err == nil {
		changes = c.String()
	}

	return []string{name, branch, changes, dir}
}

func (g *Git) Status(dto StatusDTO) error {
	ctx, err := g.resolveContext(dto.Workspace, dto.Project, dto.AllProjects)

	if err != nil {
		return err
	}

	rows := [][]string{}

	switch {
	case dto.AllProjects:
		for _, p := range ctx.Workspace.Projects {
			rows = append(rows, g.statusRow(p.Name, p.ProjectDir))
		}
	case ctx.Project != nil:
		rows = append(rows, g.statusRow(ctx.Project.Name, ctx.Project.ProjectDir))
	default:
		rows = append(rows, g.statusRow("-", ctx.WorkingDirectory))
	}

	g.tui.Table([]string{
		"Project",
		"Branch",
		"Changes",
		"Path",
	}, rows)

	return nil
}
