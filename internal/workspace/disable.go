package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/dag"
	"github.com/adamkirk/orca/internal/usererr"
)

type DisableCommand struct {
	bus.Returns[bus.Void]

	Workspace string
	Projects  []string

	// Orphan leaves the projects running, instead of stopping them
	Orphan bool
}

func (DisableCommand) MessageName() string {
	return "ws.disable"
}

func (m *Manager) Disable(ctx context.Context, cmd DisableCommand) error {
	execCtx, err := ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	ws := execCtx.Workspace
	names, err := checkProjectNames(ws, cmd.Projects)

	if err != nil {
		return err
	}

	if err := setProjectsEnabled(m.fs, ws.PreferencesPath, names, false); err != nil {
		return usererr.Wrap(err, "Failed to update the preferences file!")
	}

	m.tui.Success(fmt.Sprintf("Disabled %s.", joinNames(names, "and")))

	for _, name := range names {
		p, _ := ws.GetProject(name)
		p.Disabled = true
	}

	return m.stopDisabled(ws, names, cmd.Orphan)
}

// stopDisabled stops the projects that were just disabled, if they're running,
// unless orphan is set, or a project that's still running requires them.
func (m *Manager) stopDisabled(ws *Workspace, names []string, orphan bool) error {
	running := map[string]bool{}
	check := func(p *Project) (bool, error) {
		if r, ok := running[p.Name]; ok {
			return r, nil
		}

		r, err := m.stack.IsProjectRunning(ws, p)

		if err != nil {
			return false, err
		}

		running[p.Name] = r

		return r, nil
	}
	isRunning := func(p *Project) bool {
		r, err := check(p)

		if err != nil {
			slog.Debug("couldn't check whether a project is running", "project", p.Name, "err", err)
		}

		return r
	}

	candidates := []string{}

	for _, name := range names {
		p, _ := ws.GetProject(name)

		// A library has nothing running to stop
		if p.IsLibrary() {
			continue
		}

		r, err := check(p)

		if err != nil {
			slog.Debug("couldn't check whether a project is running", "project", name, "err", err)
			m.tui.Info(fmt.Sprintf("Warning: couldn't check whether %s is running, so it's been left as it is.", name))
			continue
		}

		if r {
			candidates = append(candidates, name)
		}
	}

	runSet, _ := ws.RunSet()
	requiredBy := keptRunning(runSet, candidates, isRunning)
	toStop := []Project{}

	for _, name := range candidates {
		if by, kept := requiredBy[name]; kept {
			m.tui.Info(fmt.Sprintf("Warning: %s requires %s, so it's been left running.", by, name))
			continue
		}

		if orphan {
			m.tui.Info(fmt.Sprintf("%s is still running, stop it with 'orca down -p %s'.", name, name))
			continue
		}

		p, _ := ws.GetProject(name)
		toStop = append(toStop, *p)
	}

	if orphan {
		return nil
	}

	g, err := dag.NewGraph(TrimRequires(toStop))

	if err != nil {
		return err
	}

	// Projects are stopped before the ones they require
	ordered, err := g.TopologicalKeysFromLeaves()

	if err != nil {
		return err
	}

	for _, name := range ordered {
		p, _ := ws.GetProject(name)

		if err := m.stack.Down(ws, p); err != nil {
			return err
		}
	}

	return nil
}

// keptRunning works out which of the candidates to stop must keep running, as
// a project that stays running requires them, mapped to that project. A
// project stays running if it isn't a candidate, or is itself kept running.
func keptRunning(runSet []Project, candidates []string, isRunning func(p *Project) bool) map[string]string {
	kept := map[string]string{}

	for changed := true; changed; {
		changed = false

		for _, name := range candidates {
			if _, ok := kept[name]; ok {
				continue
			}

			for _, p := range runSet {
				if p.Name == name || !slices.Contains(p.Requires, name) || !isRunning(&p) {
					continue
				}

				_, requirerKept := kept[p.Name]

				if !slices.Contains(candidates, p.Name) || requirerKept {
					kept[name] = p.Name
					changed = true

					break
				}
			}
		}
	}

	return kept
}
