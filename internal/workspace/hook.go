package workspace

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/adamkirk/orca/internal/workspace/internal/model"
	"gopkg.in/yaml.v3"
)

// HookPhase is whether a hook runs before or after its operation.
type HookPhase string

const (
	HookPhasePre  HookPhase = "pre"
	HookPhasePost HookPhase = "post"
)

// Hook is an operation and a phase, e.g. clone.post.
type Hook struct {
	Operation string
	Phase     HookPhase
}

func (h Hook) String() string {
	return fmt.Sprintf("%s.%s", h.Operation, h.Phase)
}

var (
	HookInitPost  = Hook{"init", HookPhasePost}
	HookClonePost = Hook{"clone", HookPhasePost}
	HookUpPre     = Hook{"up", HookPhasePre}
	HookUpPost    = Hook{"up", HookPhasePost}
	HookDownPre   = Hook{"down", HookPhasePre}
	HookDownPost  = Hook{"down", HookPhasePost}
)

// The hooks each config can define. There's no init.pre or clone.pre: a
// project's config can't be read until it's cloned, so nothing can run before.
var (
	// Every project's, the workspace repository's own included
	ProjectHooks = []Hook{HookClonePost, HookUpPre, HookUpPost, HookDownPre, HookDownPost}

	// A library is never started or stopped
	libraryHooks = []Hook{HookClonePost}

	// The workspace config's, which run projects' extensions once everything
	// is cloned
	workspaceHooks = []Hook{HookInitPost}
)

// ParseHook parses a hook's name, e.g. "clone.post".
func ParseHook(name string) (Hook, error) {
	op, phase, _ := strings.Cut(name, ".")
	hook := Hook{op, HookPhase(phase)}

	if !slices.Contains(ProjectHooks, hook) && !slices.Contains(workspaceHooks, hook) {
		return Hook{}, fmt.Errorf("%w: %s", ErrUnknownHook, name)
	}

	return hook, nil
}

// HookEntry is one of the project's own extensions that a hook runs.
type HookEntry struct {
	Extension string
}

// InitHookEntry is one extension the workspace's init.post runs. The workspace
// config has no extensions, so it's one of a project's, and runs in that
// project's directory.
type InitHookEntry struct {
	Project   string
	Extension string
}

// HookRun is one of a project's hooks.
type HookRun struct {
	Project *Project
	Hook    Hook
}

// SetUpOpts are the hooks to run once projects are cloned.
type SetUpOpts struct {
	// Each project's clone.post runs, in this order
	Projects []string

	// Then the workspace's init.post
	Initialise bool

	// Run every extension without asking, and without recording them as trusted
	Yes bool
}

func hookNames(hooks []Hook) string {
	names := make([]string, len(hooks))

	for i, h := range hooks {
		names[i] = h.String()
	}

	return strings.Join(names, ", ")
}

// eachHookEntry calls fn with every entry of every hook in the config, checking
// each hook is one of supported. label is what the config is, for messages,
// e.g. "a library project", and hint follows the message for an unsupported
// hook. Every problem is returned, prefixed with the hook and entry.
func eachHookEntry(cfg model.Hooks, label string, supported []Hook, hint string, fn func(hook Hook, node *yaml.Node) []string) []string {
	problems := []string{}

	for _, op := range slices.Sorted(maps.Keys(cfg)) {
		for _, phase := range slices.Sorted(maps.Keys(cfg[op])) {
			hook := Hook{op, HookPhase(phase)}

			if !slices.Contains(supported, hook) {
				problems = append(problems, fmt.Sprintf("hook '%s' isn't supported in %s (it supports: %s)%s", hook, label, hookNames(supported), hint))
				continue
			}

			for i, node := range cfg[op][phase] {
				for _, p := range fn(hook, &node) {
					problems = append(problems, fmt.Sprintf("hook '%s', entry %d: %s", hook, i+1, p))
				}
			}
		}
	}

	return problems
}

// decodeHookEntry checks the entry only has the fields given, then decodes it.
// It reports whether it's a mapping, without which there's nothing more to
// check.
func decodeHookEntry(node *yaml.Node, fields []string, into any) ([]string, bool) {
	if node.Kind != yaml.MappingNode {
		return []string{"must be a mapping, e.g. '- extension: env'"}, false
	}

	problems := []string{}

	for i := 0; i+1 < len(node.Content); i += 2 {
		if key := node.Content[i].Value; !slices.Contains(fields, key) {
			problems = append(problems, fmt.Sprintf("'%s' can't be used in a hook (it takes: %s); define the extension in a project's 'extensions', and reference it by name", key, strings.Join(fields, ", ")))
		}
	}

	if err := node.Decode(into); err != nil {
		problems = append(problems, fmt.Sprintf("invalid fields: %s", err))
	}

	return problems, true
}

// buildHooks checks every entry of a project's hooks. Each can only run one of
// the project's own extensions, exts. Every problem is returned, to be listed
// with the config's others. The hooks are only complete when there are none.
func buildHooks(cfg model.Hooks, label string, supported []Hook, exts []Extension) (map[Hook][]HookEntry, []string) {
	hooks := map[Hook][]HookEntry{}

	problems := eachHookEntry(cfg, label, supported, "", func(hook Hook, node *yaml.Node) []string {
		var entry model.HookEntry

		problems, ok := decodeHookEntry(node, []string{"extension"}, &entry)

		switch {
		case !ok:
		case entry.Extension == "":
			problems = append(problems, "'extension' is required")
		case !slices.ContainsFunc(exts, func(e Extension) bool { return e.Name == entry.Extension }):
			problems = append(problems, fmt.Sprintf("extension '%s' isn't defined in this config", entry.Extension))
		}

		hooks[hook] = append(hooks[hook], HookEntry{Extension: entry.Extension})

		return problems
	})

	return hooks, problems
}

// buildInitHooks checks the workspace config's hooks, which can only be
// init.post, with entries naming one of the projects. Whether the project
// defines the extension can only be checked once it's cloned, see
// checkInitHookExtensions.
func buildInitHooks(cfg model.Hooks, projects []string) ([]InitHookEntry, []string) {
	entries := []InitHookEntry{}

	problems := eachHookEntry(cfg, "the workspace config", workspaceHooks, "; put the others in a project's config", func(_ Hook, node *yaml.Node) []string {
		var entry model.InitHookEntry

		problems, ok := decodeHookEntry(node, []string{"extension", "project"}, &entry)

		if !ok {
			return problems
		}

		if entry.Extension == "" {
			problems = append(problems, "'extension' is required")
		}

		switch {
		case entry.Project == "":
			problems = append(problems, "'project' is required, naming the project whose extension it runs, as the workspace config has no extensions")
		case !slices.Contains(projects, entry.Project):
			problems = append(problems, fmt.Sprintf("project '%s' isn't in the workspace", entry.Project))
		}

		entries = append(entries, InitHookEntry{Project: entry.Project, Extension: entry.Extension})

		return problems
	})

	return entries, problems
}

// checkInitHookExtensions checks the extensions the workspace's init.post runs
// are defined by their projects. A project that isn't cloned yet isn't checked,
// as its config can't be read.
func checkInitHookExtensions(ws *Workspace) []string {
	problems := []string{}

	for i, entry := range ws.InitHooks {
		p, err := ws.GetProject(entry.Project)

		if err != nil || !p.IsRegistered {
			continue
		}

		if _, err := p.FindExtension(entry.Extension); err != nil {
			problems = append(problems, fmt.Sprintf("hook 'init.post', entry %d: project '%s' doesn't define extension '%s'", i+1, entry.Project, entry.Extension))
		}
	}

	return problems
}

// definedHooks returns the hooks that run something, in the order they're
// listed in hooks, e.g. ProjectHooks.
func definedHooks(defined map[Hook][]HookEntry, hooks []Hook) []Hook {
	return slices.DeleteFunc(slices.Clone(hooks), func(h Hook) bool {
		return len(defined[h]) == 0
	})
}
