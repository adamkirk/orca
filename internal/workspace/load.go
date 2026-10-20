package workspace

import (
	"errors"
	"fmt"
	"path/filepath"
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

func convertRepository(r model.WorkspaceProjectRepository) ProjectRepositoryConfig {
	return ProjectRepositoryConfig{
		SSH:    r.SSH,
		Self:   r.Self,
		Branch: r.Branch,
	}
}

// projectType is the project's type, which is a service unless it says
// otherwise.
func projectType(cfg model.WorkspaceProjectConfig) ProjectType {
	if cfg.Type == "" {
		return ProjectTypeService
	}

	return ProjectType(cfg.Type)
}

func buildProject(wsPCfg model.WorkspaceProjectConfig, meta ProjectMeta, pCfg model.ProjectConfig, exts []Extension, hooks map[Hook][]HookEntry) Project {
	return Project{
		Name:             wsPCfg.Name,
		Type:             projectType(wsPCfg),
		RepositoryConfig: convertRepository(wsPCfg.Repository),
		Path:             wsPCfg.Path,
		IsRegistered:     true,
		ProjectDir:       meta.Path,
		Requires:         wsPCfg.Requires,
		Disabled:         disabledByDefault(wsPCfg),
		Config: ProjectConfig{
			ComposeFiles:    convertComposeFiles(pCfg.ComposeFiles),
			Properties:      convertProperties(pCfg.Properties),
			Hosts:           pCfg.Hosts,
			TLSCertificates: pCfg.TLSCertificates,
			Extensions:      exts,
			Hooks:           hooks,
			EnvFiles:        convertEnvFiles(pCfg.EnvFiles),
		},
	}
}

type Repository struct {
	fs         afero.Fs
	userConfig userConfig
}

func projectConfigPath(projectMeta ProjectMeta) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(projectMeta.Path, "/"), DefaultProjectFileName)
}

func (wcr *Repository) loadProject(projectMeta ProjectMeta) (*model.ProjectConfig, error) {
	path := projectConfigPath(projectMeta)

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
// present, and its extensions are valid.
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

		if t := ProjectType(p.Type); p.Type != "" && t != ProjectTypeService && t != ProjectTypeLibrary {
			problems = append(problems, fmt.Sprintf("%s: 'type' must be '%s' or '%s', not '%s'", label, ProjectTypeService, ProjectTypeLibrary, p.Type))
		}

		if p.Path != "" {
			if p.Repository.Self {
				problems = append(problems, fmt.Sprintf("%s: 'path' can't be combined with 'repository.self'", label))
			} else if !filepath.IsLocal(p.Path) {
				problems = append(problems, fmt.Sprintf("%s: 'path' must be relative, and stay inside the directory projects are cloned into, not '%s'", label, p.Path))
			}
		}
	}

	exts, extProblems := buildExtensions(cfg.Extensions)
	problems = append(problems, extProblems...)

	_, hookProblems := buildHooks(cfg.Hooks, workspaceHooksConfig(exts))
	problems = append(problems, hookProblems...)

	if len(problems) > 0 {
		return InvalidWorkspaceConfigError{
			Path:     path,
			Problems: problems,
		}
	}

	return nil
}

// workspaceHooksConfig is what the workspace config's hooks can use.
func workspaceHooksConfig(exts []Extension) hooksConfig {
	return hooksConfig{
		label:     "the workspace config",
		supported: WorkspaceHooks,
		scope:     HookScopeWorkspace,
		own:       exts,
		workspace: exts,
	}
}

// buildProjectConfig decodes a project's extensions and hooks, and checks its
// config as a whole: a library can only have what applies to a project with no
// compose stack. The hooks can reference the workspace's extensions too.
func buildProjectConfig(cfg *model.ProjectConfig, meta ProjectMeta, projectType ProjectType, wsExts []Extension) ([]Extension, map[Hook][]HookEntry, error) {
	exts, problems := buildExtensions(cfg.Extensions)
	hc := hooksConfig{
		label:     "a project config",
		supported: ProjectHooks,
		scope:     HookScopeProject,
		own:       exts,
		workspace: wsExts,
	}

	if projectType == ProjectTypeLibrary {
		problems = append(problems, libraryProblems(cfg, exts)...)
		hc.label = "a library project, as it's never started"
		hc.supported = libraryHooks
	}

	hooks, hookProblems := buildHooks(cfg.Hooks, hc)
	problems = append(problems, hookProblems...)

	if len(problems) > 0 {
		return nil, nil, InvalidProjectConfigError{
			Path:     projectConfigPath(meta),
			Problems: problems,
		}
	}

	return exts, hooks, nil
}

// libraryProblems lists what's in a library's project config that only applies
// to a project with a compose stack.
func libraryProblems(cfg *model.ProjectConfig, exts []Extension) []string {
	problems := []string{}

	for _, key := range []struct {
		name string
		set  bool
	}{
		{"composeFiles", cfg.ComposeFiles.Primary != "" || len(cfg.ComposeFiles.Extras) > 0},
		{"envFiles", len(cfg.EnvFiles) > 0},
		{"hosts", len(cfg.Hosts) > 0},
		{"tlsCerts", len(cfg.TLSCertificates) > 0},
	} {
		if key.set {
			problems = append(problems, fmt.Sprintf("'%s' can't be used in a library project", key.name))
		}
	}

	for _, ext := range exts {
		if cmd, ok := ext.Spec.(CommandSpec); ok && cmd.Service != "" {
			problems = append(problems, fmt.Sprintf("extension '%s': 'service' can't be used in a library project, as it has no services", ext.Name))
		}
	}

	return problems
}

func buildUnconfiguredWorkspace(ws *model.WorkspaceConfig, path string) *UnconfiguredWorkspace {
	projects := make([]UnconfiguredProject, len(ws.Projects))

	for i, p := range ws.Projects {
		projects[i] = UnconfiguredProject{
			Name:             p.Name,
			Type:             projectType(p),
			RepositoryConfig: convertRepository(p.Repository),
			Path:             p.Path,
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

	// They were checked when the config was parsed
	exts, _ := buildExtensions(cfg.Extensions)
	hooks, _ := buildHooks(cfg.Hooks, workspaceHooksConfig(exts))

	ws := &Workspace{
		Name:       cfg.Name,
		ConfigPath: wsMeta.Path,
		Projects:   make([]Project, len(cfg.Projects)),
		Extensions: exts,
		Hooks:      hooks,
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
					Type:             projectType(pCfg),
					RepositoryConfig: convertRepository(pCfg.Repository),
					Path:             pCfg.Path,
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

		// A library doesn't need any project config
		if errors.Is(err, ErrProjectConfigNotFound) && projectType(pCfg) == ProjectTypeLibrary {
			p, err = &model.ProjectConfig{}, nil
		}

		if err != nil {
			return nil, err
		}

		pExts, hooks, err := buildProjectConfig(p, projectMeta, projectType(pCfg), exts)

		if err != nil {
			return nil, err
		}

		ws.Projects[i] = buildProject(pCfg, projectMeta, *p, pExts, hooks)
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
