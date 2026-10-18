package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/config"
	"github.com/adamkirk/orca/internal/controller"
	"github.com/adamkirk/orca/internal/docker"
	"github.com/adamkirk/orca/internal/git"
	"github.com/adamkirk/orca/internal/github"
	"github.com/adamkirk/orca/internal/hosts"
	"github.com/adamkirk/orca/internal/plugins"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/provisioner"
	"github.com/adamkirk/orca/internal/repository"
	"github.com/adamkirk/orca/internal/sys"
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

	hostSystem      *sys.HostSystem
	etcHostsManager *hosts.EtcHostsManager

	executor *process.Executor

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

	bus *bus.Bus

	executionContextMiddleware *common.ExecutionContextMiddleware
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

func (s *services) GetHostSystem() *sys.HostSystem {
	if s.hostSystem != nil {
		return s.hostSystem
	}

	s.hostSystem = sys.NewHostSystem(
		s.GetTui(),
		s.GetFs(),
		s.GetGithubClient(),
		getToolsDir(),
	)

	return s.hostSystem
}

func (s *services) GetExecutor() *process.Executor {
	if s.executor != nil {
		return s.executor
	}

	s.executor = process.NewExecutor()

	return s.executor
}

func (s *services) GetEtcHostsManager() *hosts.EtcHostsManager {
	if s.etcHostsManager != nil {
		return s.etcHostsManager
	}

	mgr, err := hosts.NewEtcHostsManager()

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
	)

	return s.git
}

func (s *services) GetGithubClient() *github.GithubClient {
	if s.githubClient != nil {
		return s.githubClient
	}

	s.githubClient = github.NewGithubClient(getGithubToken())

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
		s.GetGithubClient(),
		s.GetWorkspaceRepository(),
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

// GetBus returns the command bus, with every command handler registered.
func (s *services) GetBus() *bus.Bus {
	if s.bus != nil {
		return s.bus
	}

	b := bus.New()

	b.Use(
		bus.LoggingMiddleware,
		s.GetExecutionContextMiddleware().Handle,
	)

	b.HandleVoid(lazy(s.GetController, (*controller.Controller).Up))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).Down))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).ExecOrRun))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).ExecuteExtension))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).Logs))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).Hosts))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).Provision))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).ShowComposeConfig))
	b.HandleVoid(lazy(s.GetController, (*controller.Controller).ShowComposeCommand))

	b.HandleVoid(lazy(s.GetGit, (*git.Git).Checkout))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).ShowBranches))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).RebaseInteractively))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).Push))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).UndoLastXCommits))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).Logl))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).Status))
	b.HandleVoid(lazy(s.GetGit, (*git.Git).Pull))

	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspaces.Manager).Initialise))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspaces.Manager).Switch))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspaces.Manager).Ls))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspaces.Manager).ShowCurrent))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspaces.Manager).ClearCurrent))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspaces.Manager).Clone))

	b.HandleVoid(lazy(s.GetCertificateManager, (*tls.CertificateManager).Generate))

	b.HandleVoid(lazy(s.GetHostSystem, (*sys.HostSystem).CheckSetup))
	b.HandleVoid(lazy(s.GetHostSystem, (*sys.HostSystem).Install))
	b.HandleVoid(lazy(s.GetHostSystem, (*sys.HostSystem).UpdateSelf))

	s.bus = b

	return s.bus
}

// GetExecutionContextMiddleware returns the middleware that resolves the
// ExecutionContext for each command, with the target registered for every
// command that runs against a workspace or project.
func (s *services) GetExecutionContextMiddleware() *common.ExecutionContextMiddleware {
	if s.executionContextMiddleware != nil {
		return s.executionContextMiddleware
	}

	ec := common.NewExecutionContextMiddleware(
		s.GetContextResolver(),
		s.GetTui(),
	)

	ec.Target(func(cmd controller.UpCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(cmd controller.DownCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(cmd controller.ExecCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(cmd controller.ExecuteExtensionCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(query controller.LogsQuery) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: query.Workspace, Project: query.Project}
	})
	ec.Target(func(cmd controller.ProvisionCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(query controller.ShowComposeConfigQuery) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: query.Workspace, Project: query.Project}
	})
	ec.Target(func(query controller.ShowComposeCommandQuery) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: query.Workspace, Project: query.Project}
	})
	ec.Target(func(cmd controller.HostsCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace}
	})

	// The git commands still work in any repository without a workspace, unless
	// they're acting on all of its projects
	ec.Target(func(cmd git.CheckoutCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, WorkspaceOptional: !cmd.AllProjects}
	})
	ec.Target(func(cmd git.PullCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, WorkspaceOptional: !cmd.AllProjects}
	})
	ec.Target(func(cmd git.PushCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, WorkspaceOptional: !cmd.AllProjects}
	})
	ec.Target(func(query git.StatusQuery) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, WorkspaceOptional: !query.AllProjects}
	})

	ec.Target(func(cmd workspaces.CloneCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.WorkspaceName}
	})
	ec.Target(func(cmd tls.GenerateCommand) common.ExecutionTarget {
		return common.ExecutionTarget{Workspace: cmd.Workspace}
	})

	s.executionContextMiddleware = ec

	return s.executionContextMiddleware
}

// lazy turns a service method into a handler, which only gets (and so builds)
// the service when a message is dispatched to it, rather than when the bus is
// built.
func lazy[S any, M bus.Message[bus.Void]](get func() S, method func(S, context.Context, M) error) func(context.Context, M) error {
	return func(ctx context.Context, msg M) error {
		return method(get(), ctx, msg)
	}
}

// KillPlugins stops any running plugin processes. Unlike GetPluginManager it
// won't create the manager, so it's safe to call at any point.
func (s *services) KillPlugins() {
	if s.pluginManager != nil {
		s.pluginManager.Kill()
	}
}
