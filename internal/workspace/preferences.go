package workspace

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/adamkirk/orca/internal/workspace/internal/model"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const preferencesHeader = `Your own preferences for this workspace, such as which projects are enabled.
A project without 'enabled' set follows the workspace config, which enables it unless it says otherwise.`

// preferencesPath is where the preferences file for the workspace config at
// wsConfigPath lives: configured is relative to the workspace config, and
// defaults to the file next to it.
func preferencesPath(wsConfigPath string, configured string) string {
	if configured == "" {
		configured = DefaultPreferencesFileName
	}

	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}

	return filepath.Join(filepath.Dir(wsConfigPath), configured)
}

// readPreferencesDocument reads and validates the preferences file. A missing,
// or empty, file gives an empty document.
func readPreferencesDocument(fs afero.Fs, path string) (*yaml.Node, error) {
	exists, err := afero.Exists(fs, path)

	if err != nil {
		return nil, err
	}

	doc := &yaml.Node{}

	if !exists {
		return doc, nil
	}

	contents, err := afero.ReadFile(fs, path)

	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(contents, doc); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrConfigIsNotValidYAML, path, err)
	}

	if problems := validatePreferences(doc); len(problems) > 0 {
		return nil, InvalidPreferencesError{Path: path, Problems: problems}
	}

	return doc, nil
}

func isNull(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

// validatePreferences lists every problem with the preferences, rather than
// stopping at the first, as decoding them would.
func validatePreferences(doc *yaml.Node) []string {
	problems := []string{}

	if doc.Kind == 0 || len(doc.Content) == 0 || isNull(doc.Content[0]) {
		return problems
	}

	root := doc.Content[0]

	if root.Kind != yaml.MappingNode {
		return append(problems, "the file must be a map, e.g. 'projects:'")
	}

	for i := 0; i < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]

		if key.Value != "projects" {
			problems = append(problems, fmt.Sprintf("unknown property '%s'", key.Value))
			continue
		}

		if isNull(value) {
			continue
		}

		if value.Kind != yaml.MappingNode {
			problems = append(problems, "'projects' must be a map of project names")
			continue
		}

		for j := 0; j < len(value.Content); j += 2 {
			name, project := value.Content[j], value.Content[j+1]
			problems = append(problems, validateProjectPreferences(name.Value, project)...)
		}
	}

	return problems
}

func validateProjectPreferences(name string, project *yaml.Node) []string {
	problems := []string{}
	label := fmt.Sprintf("project '%s'", name)

	if isNull(project) {
		return problems
	}

	if project.Kind != yaml.MappingNode {
		return append(problems, fmt.Sprintf("%s: must be a map, e.g. 'enabled: true'", label))
	}

	for i := 0; i < len(project.Content); i += 2 {
		key, value := project.Content[i], project.Content[i+1]

		if key.Value != "enabled" {
			problems = append(problems, fmt.Sprintf("%s: unknown property '%s'", label, key.Value))
			continue
		}

		if value.Kind != yaml.ScalarNode || value.Tag != "!!bool" {
			problems = append(problems, fmt.Sprintf("%s: 'enabled' must be true or false", label))
		}
	}

	return problems
}

// loadPreferences loads the preferences file. A missing file means every
// project is enabled.
func loadPreferences(fs afero.Fs, path string) (*model.Preferences, error) {
	doc, err := readPreferencesDocument(fs, path)

	if err != nil {
		return nil, err
	}

	prefs := &model.Preferences{}

	if doc.Kind == 0 {
		return prefs, nil
	}

	if err := doc.Decode(prefs); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrConfigIsNotValidYAML, path, err)
	}

	return prefs, nil
}

// mappingValue returns the value for key in the mapping, adding it as an empty
// map if it's missing or null.
func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != key {
			continue
		}

		value := mapping.Content[i+1]

		if isNull(value) {
			*value = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		}

		return value
	}

	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	mapping.Content = append(
		mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)

	return value
}

func setEnabled(doc *yaml.Node, names []string, enabled bool) {
	if doc.Kind == 0 || len(doc.Content) == 0 {
		*doc = yaml.Node{
			Kind:        yaml.DocumentNode,
			HeadComment: preferencesHeader,
			Content:     []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
		}
	}

	if isNull(doc.Content[0]) {
		doc.Content[0] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}

	projects := mappingValue(doc.Content[0], "projects")

	for _, name := range names {
		project := mappingValue(projects, name)
		value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprintf("%t", enabled)}

		i := 0
		for ; i < len(project.Content); i += 2 {
			if project.Content[i].Value == "enabled" {
				break
			}
		}

		if i < len(project.Content) {
			project.Content[i+1] = value
			continue
		}

		project.Content = append(project.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "enabled"}, value)
	}
}

func writePreferencesDocument(fs afero.Fs, path string, doc *yaml.Node) error {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	if err := enc.Encode(doc); err != nil {
		return err
	}

	if err := enc.Close(); err != nil {
		return err
	}

	if err := fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return afero.WriteFile(fs, path, buf.Bytes(), 0644)
}

// setProjectsEnabled enables or disables projects in the preferences file,
// creating it if it's missing. The rest of the file, including comments, is
// kept as it is.
func setProjectsEnabled(fs afero.Fs, path string, names []string, enabled bool) error {
	doc, err := readPreferencesDocument(fs, path)

	if err != nil {
		return err
	}

	setEnabled(doc, names, enabled)

	return writePreferencesDocument(fs, path, doc)
}

// writeDefaultPreferences creates the preferences file, listing every project
// without setting 'enabled', so each follows the workspace config until the
// developer chooses otherwise. An existing file is never overwritten, so
// created is false.
func writeDefaultPreferences(fs afero.Fs, path string, names []string) (created bool, err error) {
	exists, err := afero.Exists(fs, path)

	if err != nil || exists {
		return false, err
	}

	projects := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	for _, name := range names {
		projects.Content = append(
			projects.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"},
		)
	}

	doc := &yaml.Node{
		Kind:        yaml.DocumentNode,
		HeadComment: preferencesHeader,
		Content: []*yaml.Node{{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "projects"},
				projects,
			},
		}},
	}

	return true, writePreferencesDocument(fs, path, doc)
}
