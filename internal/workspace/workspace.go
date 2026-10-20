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
	// Shown next to the name when completing it
	Description string
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

// ProjectType is what a project is for: running a compose stack, or only being
// cloned and set up.
type ProjectType string

const (
	ProjectTypeService ProjectType = "service"
	ProjectTypeLibrary ProjectType = "library"
)

type Project struct {
	Name             string
	Type             ProjectType
	RepositoryConfig ProjectRepositoryConfig

	// Path is where the project is cloned, relative to the directory projects
	// are cloned into. Empty means the project's name.
	Path         string
	ProjectDir   string
	Requires     []string
	Config       ProjectConfig
	IsRegistered bool

	// Disabled projects are left out of workspace-wide commands, unless an
	// enabled project requires them. Set from the preferences file.
	Disabled bool
}

// IsLibrary reports whether the project has no compose stack, so is never run.
func (p Project) IsLibrary() bool {
	return p.Type == ProjectTypeLibrary
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

	// PreferencesPath is the developer's preferences file, which may not exist
	PreferencesPath string

	// UnknownPreferenceProjects are listed in the preferences file, but aren't
	// in the workspace, so are ignored
	UnknownPreferenceProjects []string
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

// EnabledProjects returns the projects that aren't disabled, in the order
// they're configured.
func (ws *Workspace) EnabledProjects() []Project {
	return slices.DeleteFunc(slices.Clone(ws.Projects), func(p Project) bool {
		return p.Disabled
	})
}

// PulledInProject is a disabled project that runs anyway, as RequiredBy
// requires it.
type PulledInProject struct {
	Name       string
	RequiredBy string
}

// RunSet returns the projects that workspace-wide commands run: the enabled
// ones, and every project they require, directly or not, even if it's disabled.
// The disabled projects that were pulled in are returned too, so they can be
// warned about. Libraries are never run, so they're left out, and requirements
// on them are kept, to be trimmed or checked by the caller.
func (ws *Workspace) RunSet() ([]Project, []PulledInProject) {
	included := map[string]bool{}
	pulledIn := []PulledInProject{}
	queue := []Project{}

	for _, p := range ws.EnabledProjects() {
		if p.IsLibrary() {
			continue
		}

		included[p.Name] = true
		queue = append(queue, p)
	}

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		for _, name := range p.Requires {
			if included[name] {
				continue
			}

			required, err := ws.GetProject(name)

			// An unknown requirement is reported when the projects are ordered
			if err != nil || required.IsLibrary() {
				continue
			}

			included[name] = true
			queue = append(queue, *required)
			pulledIn = append(pulledIn, PulledInProject{Name: name, RequiredBy: p.Name})
		}
	}

	projects := slices.DeleteFunc(slices.Clone(ws.Projects), func(p Project) bool {
		return !included[p.Name]
	})

	return projects, pulledIn
}

// TrimRequires returns copies of projects that only require each other, so a
// subset of a workspace's projects can be ordered on its own.
func TrimRequires(projects []Project) []Project {
	names := make([]string, len(projects))

	for i, p := range projects {
		names[i] = p.Name
	}

	trimmed := make([]Project, len(projects))

	for i, p := range projects {
		p.Requires = slices.DeleteFunc(slices.Clone(p.Requires), func(name string) bool {
			return !slices.Contains(names, name)
		})
		trimmed[i] = p
	}

	return trimmed
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
	Type             ProjectType
	RepositoryConfig ProjectRepositoryConfig
	Path             string
}

type UnconfiguredWorkspace struct {
	Name            string
	ConfigPath      string
	PreferencesPath string
	Projects        []UnconfiguredProject
}
