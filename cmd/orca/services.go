package main

import (
	"log/slog"
	"os"

	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/config"
	"github.com/adamkirk/orca/internal/controller"
	"github.com/adamkirk/orca/internal/docker"
	"github.com/adamkirk/orca/internal/git"
	"github.com/adamkirk/orca/internal/github"
	"github.com/adamkirk/orca/internal/hostsys"
	"github.com/adamkirk/orca/internal/plugins"
	"github.com/adamkirk/orca/internal/provisioner"
	"github.com/adamkirk/orca/internal/repository"
	"github.com/adamkirk/orca/internal/tls"
	"github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/workspaces"
	"github.com/spf13/afero"
)

type services struct {
	fs afero.Fs

	contextResolver *common.ContextResolver

	config *config.Config

	tui *tui.Tui

	hostSystem      *hostsys.HostSystem
	etcHostsManager *hostsys.EtcHostsManager

	executor *hostsys.Executor

	git *git.Git

	githubClient *github.GithubClient

	workspaceManager *workspaces.Manager

	controller *controller.Controller

	workspaceRepo *repository.WorkspaceRepository

	certificateManager *tls.CertificateManager

	compose                 *docker.Compose
	composeParser           *docker.ComposeParser
	composeOverlayGenerator *docker.ComposeOverlayGenerator

	provisionerRunner *provisioner.Runner

	pluginManager *plugins.Manager
}

func (s *services) GetFs() afero.Fs {
	if s.fs != nil {
		return s.fs
	}

	s.fs = afero.NewOsFs()

	return s.fs
}

func (s *services) GetConfig() *config.Config {
	if s.config != nil {
		return s.config
	}

	s.config = config.NewDefaultConfig(
		s.GetFs(),
		getConfigFilePath(),
	)

	return s.config
}

func (s *services) GetTui() *tui.Tui {
	if s.tui != nil {
		return s.tui
	}

	s.tui = tui.NewTui(
		os.Stdout,
		os.Stderr,
	)

	return s.tui
}

func (s *services) GetHostSystem() *hostsys.HostSystem {
	if s.hostSystem != nil {
		return s.hostSystem
	}

	s.hostSystem = hostsys.NewHostSystem(
		s.GetTui(),
		s.GetFs(),
		s.GetGithubClient(),
		getToolsDir(),
	)

	return s.hostSystem
}

func (s *services) GetExecutor() *hostsys.Executor {
	if s.executor != nil {
		return s.executor
	}

	s.executor = hostsys.NewExecutor()

	return s.executor
}

func (s *services) GetEtcHostsManager() *hostsys.EtcHostsManager {
	if s.etcHostsManager != nil {
		return s.etcHostsManager
	}

	mgr, err := hostsys.NewEtcHostsManager()

	checkErr(err)

	s.etcHostsManager = mgr

	return s.etcHostsManager
}

func (s *services) GetGit() *git.Git {
	if s.git != nil {
		return s.git
	}

	s.git = git.NewGit(
		s.GetExecutor(),
		s.GetTui(),
		s.GetContextResolver(),
	)

	return s.git
}

func (s *services) GetGithubClient() *github.GithubClient {
	if s.githubClient != nil {
		return s.githubClient
	}

	s.githubClient = github.NewGithubClient()

	return s.githubClient
}

func (s *services) GetContextResolver() *common.ContextResolver {
	if s.contextResolver != nil {
		return s.contextResolver
	}

	s.contextResolver = common.NewContextResolver(
		s.GetConfig(),
		s.GetWorkspaceRepository(),
	)

	return s.contextResolver
}

func (s *services) GetWorkspaceManager() *workspaces.Manager {
	if s.workspaceManager != nil {
		return s.workspaceManager
	}

	s.workspaceManager = workspaces.NewManager(
		s.GetFs(),
		s.GetTui(),
		s.GetConfig(),
		s.GetGit(),
		s.GetWorkspaceRepository(),
		s.GetContextResolver(),
	)

	return s.workspaceManager
}

func (s *services) GetController() *controller.Controller {
	if s.controller != nil {
		return s.controller
	}

	s.controller = controller.NewController(
		s.GetConfig(),
		s.GetWorkspaceRepository(),
		s.GetCompose(),
		s.GetTui(),
		s.GetContextResolver(),
		s.GetEtcHostsManager(),
		s.GetProvisionerRunner(),
	)

	return s.controller
}

func (s *services) GetWorkspaceRepository() *repository.WorkspaceRepository {
	if s.workspaceRepo != nil {
		return s.workspaceRepo
	}

	s.workspaceRepo = repository.NewWorkspaceRepository(
		s.GetFs(),
		s.GetConfig(),
	)

	return s.workspaceRepo
}

func (s *services) GetCertificateManager() *tls.CertificateManager {
	if s.certificateManager != nil {
		return s.certificateManager
	}

	s.certificateManager = tls.NewCertificateManager(
		s.GetFs(),
		s.GetWorkspaceRepository(),
		s.GetTui(),
		getTLSDir(),
	)

	return s.certificateManager
}

func (s *services) GetCompose() *docker.Compose {
	if s.compose != nil {
		return s.compose
	}

	s.compose = docker.NewCompose(
		s.GetExecutor(),
		s.GetTui(),
		s.GetComposeOverlayGenerator(),
	)

	return s.compose
}

func (s *services) GetComposeParser() *docker.ComposeParser {
	if s.composeParser != nil {
		return s.composeParser
	}

	s.composeParser = docker.NewComposeParser()

	return s.composeParser
}

func (s *services) GetComposeOverlayGenerator() *docker.ComposeOverlayGenerator {
	if s.composeOverlayGenerator != nil {
		return s.composeOverlayGenerator
	}

	cm := s.GetCertificateManager()

	s.composeOverlayGenerator = docker.NewComposeOverlayGenerator(
		s.GetFs(),
		s.GetComposeParser(),
		getOverlayDir(),
		cm.GetCertsDir(),
	)

	return s.composeOverlayGenerator
}

func (s *services) GetProvisionerRunner() *provisioner.Runner {
	if s.provisionerRunner != nil {
		return s.provisionerRunner
	}

	s.provisionerRunner = provisioner.NewRunner(
		s.GetFs(),
		s.GetTui(),
	)

	return s.provisionerRunner
}

func (s *services) GetPluginManager() *plugins.Manager {
	if s.pluginManager != nil {
		return s.pluginManager
	}

	s.pluginManager = plugins.NewManager(slog.Default())

	return s.pluginManager
}

// KillPlugins stops any running plugin processes. Unlike GetPluginManager it
// won't create the manager, so it's safe to call at any point.
func (s *services) KillPlugins() {
	if s.pluginManager != nil {
		s.pluginManager.Kill()
	}
}
