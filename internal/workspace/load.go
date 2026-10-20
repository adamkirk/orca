package workspace

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/adamkirk/orca/internal/workspace/internal/model"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type userConfig interface {
	GetWorkspaceMeta(name string) (WorkspaceMeta, error)
	GetProjectMeta(wsName string, projectName string) (ProjectMeta, error)
}

func convertLoaderCondition(e model.LoaderCondition) LoaderCondition {
	return LoaderCondition{
		OS:   e.OS,
		Arch: e.Arch,
		Property: LoaderPropertyCondition{
			Name:  e.Property.Name,
			Value: e.Property.Value,
		},
	}
}

func convertLoaderConditions(cfgConds []model.LoaderCondition) []LoaderCondition {
	conds := make([]LoaderCondition, len(cfgConds))
	for i, c := range cfgConds {
		conds[i] = convertLoaderCondition(c)
	}

	return conds
}

func convertExtraComposeFile(e model.ExtraComposeFile) ExtraComposeFile {
	return ExtraComposeFile{
		Path: e.Path,
		When: convertLoaderConditions(e.When),
	}
}

func convertExtraComposeFiles(files []model.ExtraComposeFile) []ExtraComposeFile {
	extras := make([]ExtraComposeFile, len(files))

	for i, e := range files {
		extras[i] = convertExtraComposeFile(e)
	}

	return extras
}

func convertComposeFiles(cfg model.ComposeFiles) ComposeFiles {
	return ComposeFiles{
		Primary: cfg.Primary,
		Extras:  convertExtraComposeFiles(cfg.Extras),
	}
}

func convertProperty(p model.Property) Property {
	return Property{
		Name:    p.Name,
		Type:    p.Type,
		Default: p.Default,
	}
}

func convertProperties(cfgProps []model.Property) []Property {
	props := make([]Property, len(cfgProps))

	for i, p := range cfgProps {
		props[i] = convertProperty(p)
	}

	return props
}

func convertExtension(e model.Extension) Extension {
	return Extension{
		Name:        e.Name,
		Chdir:       e.Chdir,
		Command:     e.Command,
		Service:     e.Service,
		DefaultArgs: e.DefaultArgs,
		Description: e.Description,
	}
}

func convertExtensions(cfgExts []model.Extension) []Extension {
	exts := make([]Extension, len(cfgExts))

	for i, e := range cfgExts {
		exts[i] = convertExtension(e)
	}

	return exts
}

func convertEnvFile(e model.EnvFile) EnvFile {
	return EnvFile{
		Path: e.Path,
	}
}

func convertEnvFiles(cfgEnvFiles []model.EnvFile) []EnvFile {
	envFiles := make([]EnvFile, len(cfgEnvFiles))

	for i, e := range cfgEnvFiles {
		envFiles[i] = convertEnvFile(e)
	}

	return envFiles
}

func convertProvisioner(e model.Provisioner) Provisioner {
	return Provisioner{
		ExampleFile: &ExampleFileProvisioner{
			Src:    e.ExampleFile.Src,
			Target: e.ExampleFile.Target,
			Type:   ExampleFileProvisionerType(e.ExampleFile.Type),
		},
	}
}

func convertProvisioners(cfgProvisioners []model.Provisioner) []Provisioner {
	provisioners := make([]Provisioner, len(cfgProvisioners))
	for i, p := range cfgProvisioners {
		provisioners[i] = convertProvisioner(p)
	}

	return provisioners
}

func convertRepository(r model.WorkspaceProjectRepository) ProjectRepositoryConfig {
	return ProjectRepositoryConfig{
		SSH:    r.SSH,
		Self:   r.Self,
		Branch: r.Branch,
	}
}

func buildProject(wsPCfg model.WorkspaceProjectConfig, meta ProjectMeta, pCfg model.ProjectConfig) Project {
	return Project{
		Name:             wsPCfg.Name,
		RepositoryConfig: convertRepository(wsPCfg.Repository),
		IsRegistered:     true,
		ProjectDir:       meta.Path,
		Requires:         wsPCfg.Requires,
		Disabled:         disabledByDefault(wsPCfg),
		Config: ProjectConfig{
			ComposeFiles:    convertComposeFiles(pCfg.ComposeFiles),
			Properties:      convertProperties(pCfg.Properties),
			Hosts:           pCfg.Hosts,
			TLSCertificates: pCfg.TLSCertificates,
			Extensions:      convertExtensions(pCfg.Extensions),
			EnvFiles:        convertEnvFiles(pCfg.EnvFiles),
			Provisioners:    convertProvisioners(pCfg.Provisioners),
		},
	}
}

type Repository struct {
	fs         afero.Fs
	userConfig userConfig
}

func (wcr *Repository) loadProject(projectMeta ProjectMeta) (*model.ProjectConfig, error) {
	path := fmt.Sprintf("%s/%s", strings.TrimSuffix(projectMeta.Path, "/"), DefaultProjectFileName)

	found, err := afero.Exists(wcr.fs, path)

	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("%w: %s", ErrProjectConfigNotFound, path)
	}

	contents, err := afero.ReadFile(wcr.fs, path)

	if err != nil {
		return nil, err
	}

	cfg := &model.ProjectConfig{}

	if err := yaml.Unmarshal(contents, cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrConfigIsNotValidYAML, path, err)
	}

	return cfg, nil
}

func (wcr *Repository) loadConfigFromPath(path string) (*model.WorkspaceConfig, error) {
	found, err := afero.Exists(wcr.fs, path)

	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("%w: %s", ErrWorkspaceConfigNotFound, path)
	}

	contents, err := afero.ReadFile(wcr.fs, path)

	if err != nil {
		return nil, err
	}

	return parseWorkspaceConfig(contents, path)
}

func parseWorkspaceConfig(contents []byte, path string) (*model.WorkspaceConfig, error) {
	cfg := &model.WorkspaceConfig{}

	if err := yaml.Unmarshal(contents, cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrConfigIsNotValidYAML, path, err)
	}

	if err := validateWorkspaceConfig(cfg, path); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateWorkspaceConfig checks the required fields of a workspace config are
// present.
func validateWorkspaceConfig(cfg *model.WorkspaceConfig, path string) error {
	problems := []string{}

	if cfg.Name == "" {
		problems = append(problems, "'name' is required")
	}

	for i, p := range cfg.Projects {
		label := fmt.Sprintf("project '%s'", p.Name)

		if p.Name == "" {
			label = fmt.Sprintf("project at index %d", i)
			problems = append(problems, fmt.Sprintf("%s: 'name' is required", label))
		}

		hasSSH := p.Repository.SSH != ""

		if hasSSH == p.Repository.Self {
			problems = append(problems, fmt.Sprintf("%s: exactly one of 'repository.ssh' or 'repository.self' is required", label))
		}

		if p.Repository.Self && p.Repository.Branch != "" {
			problems = append(problems, fmt.Sprintf("%s: 'repository.branch' can't be combined with 'repository.self'", label))
		}
	}

	if len(problems) > 0 {
		return InvalidWorkspaceConfigError{
			Path:     path,
			Problems: problems,
		}
	}

	return nil
}

func buildUnconfiguredWorkspace(ws *model.WorkspaceConfig, path string) *UnconfiguredWorkspace {
	projects := make([]UnconfiguredProject, len(ws.Projects))

	for i, p := range ws.Projects {
		projects[i] = UnconfiguredProject{
			Name:             p.Name,
			RepositoryConfig: convertRepository(p.Repository),
		}
	}

	return &UnconfiguredWorkspace{
		Name:            ws.Name,
		ConfigPath:      path,
		PreferencesPath: preferencesPath(path, ws.Preferences.Path),
		Projects:        projects,
	}
}

func (wcr *Repository) LoadUnconfiguredWorkspace(path string) (*UnconfiguredWorkspace, error) {
	ws, err := wcr.loadConfigFromPath(path)

	if err != nil {
		return nil, err
	}

	return buildUnconfiguredWorkspace(ws, path), nil
}

// ParseUnconfiguredWorkspace loads a workspace from raw config contents, e.g.
// when fetched from a remote. The path is only used for reporting.
func (wcr *Repository) ParseUnconfiguredWorkspace(contents []byte, path string) (*UnconfiguredWorkspace, error) {
	ws, err := parseWorkspaceConfig(contents, path)

	if err != nil {
		return nil, err
	}

	return buildUnconfiguredWorkspace(ws, path), nil
}

func (wcr *Repository) Load(name string) (*Workspace, error) {
	wsMeta, err := wcr.userConfig.GetWorkspaceMeta(name)
	if err != nil {
		return nil, err
	}

	cfg, err := wcr.loadConfigFromPath(wsMeta.Path)

	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		Name:       cfg.Name,
		ConfigPath: wsMeta.Path,
		Projects:   make([]Project, len(cfg.Projects)),
		OverlayConfig: OverlayConfig{
			Network: NetworkOverlayConfig{
				Enabled:        cfg.Overlays.Network.Enabled,
				CreateIn:       cfg.Overlays.Network.CreateIn,
				DisableAliases: cfg.Overlays.Network.DisableAliases,
				AliasPattern:   cfg.Overlays.Network.AliasPattern,
			},
		},
	}

	for i, pCfg := range cfg.Projects {
		projectMeta, err := wcr.userConfig.GetProjectMeta(wsMeta.Name, pCfg.Name)

		if err != nil {
			if errors.Is(err, ErrUnknownProject) {
				ws.Projects[i] = Project{
					Name:             pCfg.Name,
					RepositoryConfig: convertRepository(pCfg.Repository),
					IsRegistered:     false,
					ProjectDir:       "",
					Requires:         pCfg.Requires,
					Disabled:         disabledByDefault(pCfg),
					Config:           ProjectConfig{},
				}
				continue
			}

			return nil, err
		}

		p, err := wcr.loadProject(projectMeta)

		if err != nil {
			return nil, err
		}

		ws.Projects[i] = buildProject(pCfg, projectMeta, *p)
	}

	ws.PreferencesPath = preferencesPath(wsMeta.Path, cfg.Preferences.Path)

	prefs, err := loadPreferences(wcr.fs, ws.PreferencesPath)

	if err != nil {
		return nil, err
	}

	applyPreferences(ws, prefs)

	return ws, nil
}

// disabledByDefault reports whether the workspace config disables the project,
// before the developer's preferences are applied.
func disabledByDefault(cfg model.WorkspaceProjectConfig) bool {
	return cfg.Enabled != nil && !*cfg.Enabled
}

// applyPreferences enables or disables the projects the preferences set,
// overriding the workspace config, and records any it lists that aren't in the
// workspace.
func applyPreferences(ws *Workspace, prefs *model.Preferences) {
	for name, pp := range prefs.Projects {
		p, err := ws.GetProject(name)

		if err != nil {
			ws.UnknownPreferenceProjects = append(ws.UnknownPreferenceProjects, name)
			continue
		}

		if pp.Enabled != nil {
			p.Disabled = !*pp.Enabled
		}
	}

	slices.Sort(ws.UnknownPreferenceProjects)
}

func NewRepository(fs afero.Fs, userConfig userConfig) *Repository {
	return &Repository{
		fs:         fs,
		userConfig: userConfig,
	}
}
