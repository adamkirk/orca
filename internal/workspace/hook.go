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
	ProjectHooks   = []Hook{HookClonePost, HookUpPre, HookUpPost, HookDownPre, HookDownPost}
	WorkspaceHooks = []Hook{HookInitPost, HookUpPre, HookUpPost, HookDownPre, HookDownPost}

	// A library is never started or stopped
	libraryHooks = []Hook{HookClonePost}
)

// ParseHook parses a hook's name, e.g. "clone.post".
func ParseHook(name string) (Hook, error) {
	op, phase, _ := strings.Cut(name, ".")
	hook := Hook{op, HookPhase(phase)}

	if !slices.Contains(ProjectHooks, hook) && !slices.Contains(WorkspaceHooks, hook) {
		return Hook{}, fmt.Errorf("%w: %s", ErrUnknownHook, name)
	}

	return hook, nil
}

// HookScope is the config an extension in a hook is defined in.
type HookScope string

const (
	HookScopeProject   HookScope = "project"
	HookScopeWorkspace HookScope = "workspace"
)

// HookEntry is one extension a hook runs, by name.
type HookEntry struct {
	Extension string

	// Always set: the config the hook is in, unless it says otherwise
	Scope HookScope
}

// HookRun is one of a project's hooks, or one of the workspace's when Project is
// nil.
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

// hooksConfig is what a config's hooks can reference, for checking them.
type hooksConfig struct {
	// What the config is, for messages, e.g. "a library project"
	label     string
	supported []Hook
	scope     HookScope

	// The extensions each scope defines. A workspace config's hooks can only
	// use its own.
	own       []Extension
	workspace []Extension
}

var hookEntryFields = []string{"extension", "scope"}

func hookNames(hooks []Hook) string {
	names := make([]string, len(hooks))

	for i, h := range hooks {
		names[i] = h.String()
	}

	return strings.Join(names, ", ")
}

// buildHooks checks every entry of every hook, and returns them with their
// scopes filled in. Every problem is returned, to be listed with the config's
// others. The hooks are only complete when there are no problems.
func buildHooks(cfg model.Hooks, hc hooksConfig) (map[Hook][]HookEntry, []string) {
	hooks := map[Hook][]HookEntry{}
	problems := []string{}

	for _, op := range slices.Sorted(maps.Keys(cfg)) {
		for _, phase := range slices.Sorted(maps.Keys(cfg[op])) {
			hook := Hook{op, HookPhase(phase)}

			if !slices.Contains(hc.supported, hook) {
				problems = append(problems, fmt.Sprintf("hook '%s' isn't supported in %s (it supports: %s)", hook, hc.label, hookNames(hc.supported)))
				continue
			}

			entries := []HookEntry{}

			for i, node := range cfg[op][phase] {
				entry, entryProblems := buildHookEntry(&node, hc)

				for _, p := range entryProblems {
					problems = append(problems, fmt.Sprintf("hook '%s', entry %d: %s", hook, i+1, p))
				}

				entries = append(entries, entry)
			}

			hooks[hook] = entries
		}
	}

	return hooks, problems
}

func buildHookEntry(node *yaml.Node, hc hooksConfig) (HookEntry, []string) {
	if node.Kind != yaml.MappingNode {
		return HookEntry{}, []string{"must be a mapping, e.g. '- extension: env'"}
	}

	problems := []string{}

	for i := 0; i+1 < len(node.Content); i += 2 {
		if key := node.Content[i].Value; !slices.Contains(hookEntryFields, key) {
			problems = append(problems, fmt.Sprintf("'%s' can't be used in a hook (it takes: %s); define the extension in 'extensions', and reference it by name", key, strings.Join(hookEntryFields, ", ")))
		}
	}

	var cfg model.HookEntry

	if err := node.Decode(&cfg); err != nil {
		return HookEntry{}, append(problems, fmt.Sprintf("invalid fields: %s", err))
	}

	if cfg.Extension == "" {
		problems = append(problems, "'extension' is required")
	}

	entry := HookEntry{Extension: cfg.Extension, Scope: HookScope(cfg.Scope)}

	if entry.Scope == "" {
		entry.Scope = hc.scope
	}

	var exts []Extension

	switch {
	case entry.Scope == HookScopeWorkspace:
		exts = hc.workspace
	case entry.Scope == HookScopeProject && hc.scope == HookScopeProject:
		exts = hc.own
	case entry.Scope == HookScopeProject:
		return entry, append(problems, "'scope' can't be 'project' in the workspace config, as its hooks aren't tied to a project")
	default:
		return entry, append(problems, fmt.Sprintf("'scope' must be '%s' or '%s', not '%s'", HookScopeProject, HookScopeWorkspace, cfg.Scope))
	}

	if cfg.Extension != "" && !slices.ContainsFunc(exts, func(e Extension) bool { return e.Name == cfg.Extension }) {
		problems = append(problems, fmt.Sprintf("extension '%s' isn't defined in the %s config", cfg.Extension, entry.Scope))
	}

	return entry, problems
}

// definedHooks returns the hooks that run something, in the order they're
// listed in hooks, e.g. ProjectHooks.
func definedHooks(defined map[Hook][]HookEntry, hooks []Hook) []Hook {
	return slices.DeleteFunc(slices.Clone(hooks), func(h Hook) bool {
		return len(defined[h]) == 0
	})
}
