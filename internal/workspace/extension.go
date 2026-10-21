package workspace

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/adamkirk/orca/internal/workspace/internal/model"
	"gopkg.in/yaml.v3"
)

// ExtensionType is what kind of thing an extension does, and so which fields it
// takes.
type ExtensionType string

const (
	ExtensionTypeCommand  ExtensionType = "command"
	ExtensionTypeSyncFile ExtensionType = "syncFile"
	ExtensionTypeSymlink  ExtensionType = "symlink"
)

// ExtensionSpec is what an extension does, one of CommandSpec, SyncFileSpec or
// SymlinkSpec. Code handling extensions switches on the concrete type, and
// panics on any other, as the set is closed: only this package's types
// implement it.
type ExtensionSpec interface {
	Type() ExtensionType
	isExtensionSpec()
}

// CommandSpec runs a command: in a compose service when Service is set,
// otherwise on the host.
type CommandSpec struct {
	Command     string
	Service     string
	DefaultArgs []string

	// Relative to the directory the extension runs in
	Chdir string
}

func (CommandSpec) Type() ExtensionType { return ExtensionTypeCommand }
func (CommandSpec) isExtensionSpec()    {}

// SyncFileSpec creates Target from Source, when Target doesn't exist.
type SyncFileSpec struct {
	Source string
	Target string
}

func (SyncFileSpec) Type() ExtensionType { return ExtensionTypeSyncFile }
func (SyncFileSpec) isExtensionSpec()    {}

// SymlinkSpec creates Target as a symlink to Source.
type SymlinkSpec struct {
	Source string
	Target string
}

func (SymlinkSpec) Type() ExtensionType { return ExtensionTypeSymlink }
func (SymlinkSpec) isExtensionSpec()    {}

type Extension struct {
	Name string

	// Shown next to the name when completing it
	Description string
	Spec        ExtensionSpec
}

// extensionKind is how to decode one type of extension from its config.
type extensionKind struct {
	// The keys the type takes, in the order they're declared
	fields   []string
	required []string
	decode   func(node *yaml.Node) (ExtensionSpec, error)
}

// sharedExtensionFields are taken by every type of extension.
var sharedExtensionFields = []string{"name", "type", "description"}

// extensionKinds are the types of extension, keyed by the value of 'type'.
// Adding a type means adding its struct to the model, an entry here, and a case
// wherever extensions are switched on.
var extensionKinds = map[ExtensionType]extensionKind{
	ExtensionTypeCommand: newExtensionKind(func(s model.CommandSpec) ExtensionSpec {
		return CommandSpec{Command: s.Command, Service: s.Service, DefaultArgs: s.DefaultArgs, Chdir: s.Chdir}
	}),
	ExtensionTypeSyncFile: newExtensionKind(func(s model.SyncFileSpec) ExtensionSpec {
		return SyncFileSpec{Source: s.Source, Target: s.Target}
	}),
	ExtensionTypeSymlink: newExtensionKind(func(s model.SymlinkSpec) ExtensionSpec {
		return SymlinkSpec{Source: s.Source, Target: s.Target}
	}),
}

// newExtensionKind reads the type's keys from the yaml tags of its model struct,
// T, and decodes it into T before converting it.
func newExtensionKind[T any](convert func(T) ExtensionSpec) extensionKind {
	kind := extensionKind{
		decode: func(node *yaml.Node) (ExtensionSpec, error) {
			var spec T

			if err := node.Decode(&spec); err != nil {
				return nil, err
			}

			return convert(spec), nil
		},
	}

	t := reflect.TypeFor[T]()

	for i := range t.NumField() {
		f := t.Field(i)
		name := f.Tag.Get("yaml")

		if name == "" {
			panic(fmt.Sprintf("%s.%s needs a yaml tag", t, f.Name))
		}

		kind.fields = append(kind.fields, name)

		if f.Tag.Get("required") == "true" {
			kind.required = append(kind.required, name)
		}
	}

	return kind
}

// extensionTypeNames lists the types, for messages.
func extensionTypeNames() []string {
	names := []string{}

	for t := range extensionKinds {
		names = append(names, string(t))
	}

	slices.Sort(names)

	return names
}

// buildExtensions decodes each extension into the struct for its type. Every
// problem is returned, rather than stopping at the first, so they can be listed
// with the config's other problems. The extensions are only complete when there
// are no problems.
func buildExtensions(cfgExts []model.Extension) ([]Extension, []string) {
	exts := make([]Extension, 0, len(cfgExts))
	problems := []string{}
	seen := map[string]bool{}

	for i, cfg := range cfgExts {
		label := fmt.Sprintf("extension '%s'", cfg.Name)

		if cfg.Name == "" {
			label = fmt.Sprintf("extension at index %d", i)
			problems = append(problems, fmt.Sprintf("%s: 'name' is required", label))
		} else if seen[cfg.Name] {
			problems = append(problems, fmt.Sprintf("%s: defined more than once", label))
		}

		seen[cfg.Name] = true

		spec, specProblems := decodeExtensionSpec(cfg)

		for _, p := range specProblems {
			problems = append(problems, fmt.Sprintf("%s: %s", label, p))
		}

		if len(specProblems) == 0 {
			exts = append(exts, Extension{Name: cfg.Name, Description: cfg.Description, Spec: spec})
		}
	}

	return exts, problems
}

// decodeExtensionSpec decodes the fields of the extension's type, after
// checking it only has the keys its type takes, and has every required one.
func decodeExtensionSpec(cfg model.Extension) (ExtensionSpec, []string) {
	t := ExtensionType(cfg.Type)

	if t == "" {
		t = ExtensionTypeCommand
	}

	kind, ok := extensionKinds[t]

	if !ok {
		return nil, []string{fmt.Sprintf("unknown type '%s' (expected %s)", cfg.Type, strings.Join(extensionTypeNames(), ", "))}
	}

	problems := []string{}
	present := map[string]bool{}

	// The node is a mapping, alternating keys and values
	for i := 0; i+1 < len(cfg.Spec.Content); i += 2 {
		key, value := cfg.Spec.Content[i].Value, cfg.Spec.Content[i+1]

		// An empty value counts as missing, e.g. 'command:'
		present[key] = value.Kind != yaml.ScalarNode || value.Value != ""

		if !slices.Contains(sharedExtensionFields, key) && !slices.Contains(kind.fields, key) {
			problems = append(problems, fmt.Sprintf("'%s' can't be used with type '%s' (it takes: %s)", key, t, strings.Join(kind.fields, ", ")))
		}
	}

	for _, key := range kind.required {
		if !present[key] {
			problems = append(problems, fmt.Sprintf("'%s' is required for type '%s'", key, t))
		}
	}

	if len(problems) > 0 {
		return nil, problems
	}

	spec, err := kind.decode(&cfg.Spec)

	if err != nil {
		return nil, []string{fmt.Sprintf("invalid fields for type '%s': %s", t, err)}
	}

	return spec, nil
}
