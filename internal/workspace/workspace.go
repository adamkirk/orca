package workspace

import (
	"fmt"
	"slices"
)

type ExampleFileProvisionerType string

const ExampleFileProvisionerTypeDotenv ExampleFileProvisionerType = "dotenv"

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

type Extension struct {
	Name        string
	Chdir       string
	Command     string
	Service     string
	DefaultArgs []string
}

type EnvFile struct {
	Path string
}

type ExampleFileProvisioner struct {
	Src    string
	Target string
	Type   ExampleFileProvisionerType
}

type Provisioner struct {
	ExampleFile *ExampleFileProvisioner
}

type ProjectConfig struct {
	ComposeFiles    ComposeFiles
	Provisioners    []Provisioner
	EnvFiles        []EnvFile
	Properties      []Property
	Hosts           []string
	TLSCertificates []string
	Extensions      []Extension
}

type ProjectRepositoryConfig struct {
	SSH  string
	Self bool

	// Branch is checked out when the project is first cloned, it's never used
	// to switch the branch of an existing clone.
	Branch string
}

type Project struct {
	Name             string
	RepositoryConfig ProjectRepositoryConfig
	ProjectDir       string
	Requires         []string
	Config           ProjectConfig
	IsRegistered     bool
}

func (p Project) GetKey() string {
	return p.Name
}

func (p Project) GetParents() []string {
	return p.Requires
}

func (p Project) GetChildren() []string {
	return []string{}
}

func (p Project) EnvFilePaths() []string {
	paths := make([]string, len(p.Config.EnvFiles))
	for i, e := range p.Config.EnvFiles {
		paths[i] = e.Path
	}

	return paths
}

func (p Project) FindExtension(name string) (Extension, error) {
	for _, ext := range p.Config.Extensions {
		if ext.Name == name {
			return ext, nil
		}
	}

	return Extension{}, fmt.Errorf("%w: %s", ErrUnknownExtension, name)
}

type NetworkOverlayConfig struct {
	Enabled        bool
	CreateIn       string
	DisableAliases bool
	AliasPattern   string
}

type OverlayConfig struct {
	Network NetworkOverlayConfig
}

type Workspace struct {
	Name          string
	ConfigPath    string
	Projects      []Project
	OverlayConfig OverlayConfig `yaml:"overlays"`
}

func (ws *Workspace) GetProject(name string) (*Project, error) {
	i := slices.IndexFunc(ws.Projects, func(p Project) bool {
		return p.Name == name
	})

	if i == -1 {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProject, name)
	}

	return &ws.Projects[i], nil
}

func (ws *Workspace) GetUniqueTLSCertificates() []string {
	certs := []string{}

	for _, p := range ws.Projects {
		for _, c := range p.Config.TLSCertificates {
			if !slices.Contains(certs, c) {
				certs = append(certs, c)
			}
		}
	}

	return certs
}

func (ws *Workspace) GetUniqueHosts() []string {
	hosts := []string{}

	for _, p := range ws.Projects {
		for _, c := range p.Config.Hosts {
			if !slices.Contains(hosts, c) {
				hosts = append(hosts, c)
			}
		}
	}

	return hosts
}

// --- unconfigured version

type UnconfiguredProject struct {
	Name             string
	RepositoryConfig ProjectRepositoryConfig
}

type UnconfiguredWorkspace struct {
	Name       string
	ConfigPath string
	Projects   []UnconfiguredProject
}
