package model

type WorkspaceProjectRepository struct {
	SSH string

	// Self signifies that this project is actually the same repo as the workspace
	// itself.
	Self bool
}

type WorkspaceProjectConfig struct {
	Name       string
	Repository WorkspaceProjectRepository
	Path       string
	Requires   []string
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

type WorkspaceConfig struct {
	Name     string
	Projects []WorkspaceProjectConfig
	Overlays Overlays
}
