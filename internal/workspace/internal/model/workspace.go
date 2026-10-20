package model

type WorkspaceProjectRepository struct {
	SSH string

	// Self signifies that this project is actually the same repo as the workspace
	// itself.
	Self bool

	// Branch is checked out when the project is first cloned, instead of the
	// remote's default branch.
	Branch string
}

type WorkspaceProjectConfig struct {
	Name       string
	Type       string
	Repository WorkspaceProjectRepository

	// Where to clone the project, relative to the directory projects are
	// cloned into
	Path     string
	Requires []string

	// Whether the project is enabled by default, nil means it is. Each
	// developer's preferences file can override it.
	Enabled *bool
}

type NetworkOverlay struct {
	Enabled        bool
	CreateIn       string `yaml:"createIn"`
	DisableAliases bool   `yaml:"disableAliases"`
	AliasPattern   string `yaml:"aliasPattern"`
}

type Overlays struct {
	Network NetworkOverlay
}

type PreferencesConfig struct {
	// Where the preferences file is, relative to the workspace config
	Path string
}

type WorkspaceConfig struct {
	Name        string
	Projects    []WorkspaceProjectConfig
	Overlays    Overlays
	Preferences PreferencesConfig
	Extensions  []Extension
}
