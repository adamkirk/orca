package model

import "gopkg.in/yaml.v3"

type LoaderPropertyCondition struct {
	Name  string
	Value any
}

type LoaderCondition struct {
	OS       string
	Arch     string
	Property LoaderPropertyCondition
}

type ExtraComposeFile struct {
	Path string
	When []LoaderCondition
}

type ComposeFiles struct {
	Primary string
	Extras  []ExtraComposeFile
}

type Property struct {
	Name    string
	Type    string
	Default any
}

// Extension is an extension's shared fields. The rest depend on its type, so
// the whole mapping is kept in Spec, to be decoded into the type's own struct
// (CommandSpec, SyncFileSpec...) once the type is known.
type Extension struct {
	Name        string
	Type        string
	Description string
	Spec        yaml.Node `yaml:"-"`
}

func (e *Extension) UnmarshalYAML(node *yaml.Node) error {
	var shared struct {
		Name        string
		Type        string
		Description string
	}

	if err := node.Decode(&shared); err != nil {
		return err
	}

	e.Name = shared.Name
	e.Type = shared.Type
	e.Description = shared.Description
	e.Spec = *node

	return nil
}

// The fields of each extension type. Every field needs a yaml tag, as the tags
// are the keys allowed for the type, and `required:"true"` if it must be set.

type CommandSpec struct {
	Command     string   `yaml:"command" required:"true"`
	Service     string   `yaml:"service"`
	DefaultArgs []string `yaml:"defaultArgs"`
	Chdir       string   `yaml:"chdir"`
}

type SyncFileSpec struct {
	Source string `yaml:"source" required:"true"`
	Target string `yaml:"target" required:"true"`
}

type SymlinkSpec struct {
	Source string `yaml:"source" required:"true"`
	Target string `yaml:"target" required:"true"`
}

type EnvFile struct {
	Path string
}

type ProjectConfig struct {
	ComposeFiles    ComposeFiles `yaml:"composeFiles"`
	EnvFiles        []EnvFile    `yaml:"envFiles"`
	Properties      []Property
	Hosts           []string
	TLSCertificates []string `yaml:"tlsCerts"`
	Extensions      []Extension
}
