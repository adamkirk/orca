package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/config"
	"github.com/adamkirk/orca/internal/docker"
	"github.com/adamkirk/orca/internal/extension"
	"github.com/adamkirk/orca/internal/git"
	"github.com/adamkirk/orca/internal/github"
	"github.com/adamkirk/orca/internal/hosts"
	"github.com/adamkirk/orca/internal/plugins"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/stack"
	"github.com/adamkirk/orca/internal/sys"
	"github.com/adamkirk/orca/internal/tls"
	"github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
)

type services struct {
	fs afero.Fs

	contextResolver *workspace.Resolver

	config *config.Config

	tui *tui.Tui

	sys             *sys.Manager
	etcHostsManager *hosts.EtcHostsManager

	executor *process.Executor

	git *git.Manager

	githubClient *github.GithubClient

	workspaceManager *workspace.Manager

	stack *stack.Manager

	extension  *extension.Manager
	trustStore *extension.TrustStore

	hosts *hosts.Manager

	workspaceRepo *workspace.Repository

	tls *tls.Manager

	compose                 *docker.Compose
	composeParser           *docker.ComposeParser
	composeOverlayGenerator *docker.ComposeOverlayGenerator

	pluginManager      *plugins.Manager
	pluginCommandCache *plugins.CommandCache

	bus *bus.Bus

	executionContextMiddleware *workspace.ExecutionContextMiddleware
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
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)

	return s.tui
}

func (s *services) GetSys() *sys.Manager {
	if s.sys != nil {
		return s.sys
	}

	s.sys = sys.NewManager(
		s.GetTui(),
		s.GetGithubClient(),
	)

	return s.sys
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

func (s *services) GetGit() *git.Manager {
	if s.git != nil {
		return s.git
	}

	s.git = git.NewManager(
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

func (s *services) GetContextResolver() *workspace.Resolver {
	if s.contextResolver != nil {
		return s.contextResolver
	}

	s.contextResolver = workspace.NewResolver(
		s.GetConfig(),
		s.GetWorkspaceRepository(),
	)

	return s.contextResolver
}

func (s *services) GetWorkspaceManager() *workspace.Manager {
	if s.workspaceManager != nil {
		return s.workspaceManager
	}

	s.workspaceManager = workspace.NewManager(
		s.GetFs(),
		s.GetTui(),
		s.GetConfig(),
		s.GetGit(),
		s.GetGithubClient(),
		s.GetWorkspaceRepository(),
		s.GetCompose(),
		s.GetExtension(),
	)

	return s.workspaceManager
}

func (s *services) GetStack() *stack.Manager {
	if s.stack != nil {
		return s.stack
	}

	s.stack = stack.NewManager(s.GetCompose(), s.GetTui())

	return s.stack
}

func (s *services) GetExtension() *extension.Manager {
	if s.extension != nil {
		return s.extension
	}

	s.extension = extension.NewManager(s.GetFs(), s.GetCompose(), s.GetExecutor(), s.GetTui(), s.GetTrustStore())

	return s.extension
}

func (s *services) GetTrustStore() *extension.TrustStore {
	if s.trustStore != nil {
		return s.trustStore
	}

	s.trustStore = extension.NewTrustStore(s.GetFs(), getTrustFilePath())

	return s.trustStore
}

func (s *services) GetHosts() *hosts.Manager {
	if s.hosts != nil {
		return s.hosts
	}

	s.hosts = hosts.NewManager(
		s.GetEtcHostsManager(),
		s.GetTui(),
	)

	return s.hosts
}

func (s *services) GetWorkspaceRepository() *workspace.Repository {
	if s.workspaceRepo != nil {
		return s.workspaceRepo
	}

	s.workspaceRepo = workspace.NewRepository(
		s.GetFs(),
		s.GetConfig(),
	)

	return s.workspaceRepo
}

func (s *services) GetTLS() *tls.Manager {
	if s.tls != nil {
		return s.tls
	}

	s.tls = tls.NewManager(
		s.GetFs(),
		s.GetTui(),
		getTLSDir(),
	)

	return s.tls
}

func (s *services) GetCompose() *docker.Compose {
	if s.compose != nil {
		return s.compose
	}

	s.compose = docker.NewCompose(
		s.GetExecutor(),
		s.GetTui(),
		s.GetComposeOverlayGenerator(),
		s.GetComposeParser(),
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

	cm := s.GetTLS()

	s.composeOverlayGenerator = docker.NewComposeOverlayGenerator(
		s.GetFs(),
		s.GetComposeParser(),
		getOverlayDir(),
		cm.GetCertsDir(),
	)

	return s.composeOverlayGenerator
}

func (s *services) GetPluginManager() *plugins.Manager {
	if s.pluginManager != nil {
		return s.pluginManager
	}

	s.pluginManager = plugins.NewManager(slog.Default())

	return s.pluginManager
}

func (s *services) GetPluginCommandCache() *plugins.CommandCache {
	if s.pluginCommandCache != nil {
		return s.pluginCommandCache
	}

	s.pluginCommandCache = plugins.NewCommandCache(s.GetFs(), getPluginCommandCachePath())

	return s.pluginCommandCache
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

	b.HandleVoid(lazy(s.GetStack, (*stack.Manager).Up))
	b.HandleVoid(lazy(s.GetStack, (*stack.Manager).Down))
	b.HandleVoid(lazy(s.GetStack, (*stack.Manager).ExecOrRun))
	b.HandleVoid(lazy(s.GetExtension, (*extension.Manager).Run))
	b.HandleVoid(lazy(s.GetStack, (*stack.Manager).Logs))
	b.HandleVoid(lazy(s.GetHosts, (*hosts.Manager).Sync))
	b.HandleVoid(lazy(s.GetStack, (*stack.Manager).ShowComposeConfig))
	b.HandleVoid(lazy(s.GetStack, (*stack.Manager).ShowComposeCommand))
	b.Handle(lazyResult(s.GetStack, (*stack.Manager).ServiceChoices))
	b.Handle(lazyResult(s.GetExtension, (*extension.Manager).Choices))
	b.HandleVoid(lazy(s.GetExtension, (*extension.Manager).RunHook))
	b.Handle(lazyResult(s.GetExtension, (*extension.Manager).HookChoices))

	b.HandleVoid(lazy(s.GetGit, (*git.Manager).Checkout))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).ShowBranches))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).RebaseInteractively))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).Push))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).UndoLastXCommits))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).Logl))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).Status))
	b.HandleVoid(lazy(s.GetGit, (*git.Manager).Pull))
	b.Handle(lazyResult(s.GetGit, (*git.Manager).BranchChoices))

	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Initialise))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Switch))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Ls))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).ShowCurrent))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).ClearCurrent))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Clone))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Projects))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Enable))
	b.HandleVoid(lazy(s.GetWorkspaceManager, (*workspace.Manager).Disable))
	b.Handle(lazyResult(s.GetWorkspaceManager, (*workspace.Manager).WorkspaceChoices))
	b.Handle(lazyResult(s.GetWorkspaceManager, (*workspace.Manager).ProjectChoices))

	b.HandleVoid(lazy(s.GetTLS, (*tls.Manager).Generate))

	b.HandleVoid(lazy(s.GetSys, (*sys.Manager).CheckSetup))
	b.HandleVoid(lazy(s.GetSys, (*sys.Manager).UpdateSelf))

	s.bus = b

	return s.bus
}

// GetExecutionContextMiddleware returns the middleware that resolves the
// ExecutionContext for each command, with the target registered for every
// command that runs against a workspace or project.
func (s *services) GetExecutionContextMiddleware() *workspace.ExecutionContextMiddleware {
	if s.executionContextMiddleware != nil {
		return s.executionContextMiddleware
	}

	ec := workspace.NewExecutionContextMiddleware(s.GetContextResolver())

	ec.Target(func(cmd stack.UpCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, ServicesOnly: true}
	})
	ec.Target(func(cmd stack.DownCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, ServicesOnly: true}
	})
	ec.Target(func(cmd stack.ExecCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, ServicesOnly: true}
	})
	ec.Target(func(cmd extension.RunCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(query stack.LogsQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, ServicesOnly: true}
	})
	ec.Target(func(query stack.ShowComposeConfigQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, ServicesOnly: true}
	})
	ec.Target(func(query stack.ShowComposeCommandQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, ServicesOnly: true}
	})
	ec.Target(func(query stack.ServiceChoicesQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, ServicesOnly: true}
	})
	ec.Target(func(query extension.ChoicesQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project}
	})
	ec.Target(func(cmd extension.RunHookCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project}
	})
	ec.Target(func(query extension.HookChoicesQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project}
	})
	ec.Target(func(cmd hosts.SyncCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace}
	})

	// The git commands still work in any repository without a workspace, unless
	// they're acting on all of its projects
	ec.Target(func(cmd git.CheckoutCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, WorkspaceOptional: !cmd.AllProjects}
	})
	ec.Target(func(cmd git.PullCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, WorkspaceOptional: !cmd.AllProjects}
	})
	ec.Target(func(cmd git.PushCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace, Project: cmd.Project, WorkspaceOptional: !cmd.AllProjects}
	})
	ec.Target(func(query git.BranchChoicesQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, WorkspaceOptional: true}
	})
	ec.Target(func(query git.StatusQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace, Project: query.Project, WorkspaceOptional: !query.AllProjects}
	})

	ec.Target(func(cmd workspace.CloneCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.WorkspaceName}
	})
	ec.Target(func(query workspace.ProjectsQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace}
	})
	ec.Target(func(cmd workspace.EnableCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace}
	})
	ec.Target(func(cmd workspace.DisableCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace}
	})
	ec.Target(func(cmd tls.GenerateCommand) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: cmd.Workspace}
	})
	ec.Target(func(query workspace.ProjectChoicesQuery) workspace.ExecutionTarget {
		return workspace.ExecutionTarget{Workspace: query.Workspace}
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

// lazyResult is lazy, for messages that return a result.
func lazyResult[S any, M bus.Message[R], R any](get func() S, method func(S, context.Context, M) (R, error)) func(context.Context, M) (R, error) {
	return func(ctx context.Context, msg M) (R, error) {
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
