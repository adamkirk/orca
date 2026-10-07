package plugins

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adamkirk/orca/pkg/plugin"
	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
)

// LoadedPlugin is a running plugin process, along with the commands it provides.
type LoadedPlugin struct {
	Name     string
	Path     string
	Provider plugin.CommandProvider
	Commands []plugin.CommandSpec
}

// ErrKilled is returned when trying to load a plugin after Kill has been called.
var ErrKilled = errors.New("plugin manager has been killed")

type Manager struct {
	logger *slog.Logger
	loaded []LoadedPlugin

	// Guards clients and killed, as Kill may be called from a signal handler
	// while plugins are loading.
	mu       sync.Mutex
	clients  []*goplugin.Client
	killed   bool
	killOnce sync.Once
}

// Load starts each plugin binary and fetches its commands. A plugin that fails to
// start is logged and skipped, so a broken plugin never stops orca from running.
func (m *Manager) Load(paths []string) []LoadedPlugin {
	for _, p := range paths {
		lp, err := m.load(p)

		if errors.Is(err, ErrKilled) {
			break
		}

		if err != nil {
			m.logger.Warn("failed to load plugin", "path", p, "err", err)
			continue
		}

		m.loaded = append(m.loaded, lp)
	}

	return m.loaded
}

func (m *Manager) load(path string) (LoadedPlugin, error) {
	client, err := m.newClient(path)

	if err != nil {
		return LoadedPlugin{}, err
	}

	lp, err := m.connect(client, path)

	if err != nil {
		// Make sure a plugin that failed the handshake isn't left running
		client.Kill()

		return LoadedPlugin{}, err
	}

	return lp, nil
}

// newClient creates the client for a plugin, tracking it before the process is
// started so that Kill can never miss it.
func (m *Manager) newClient(path string) (*goplugin.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.killed {
		return nil, ErrKilled
	}

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  plugin.Handshake,
		Plugins:          plugin.PluginMap,
		Cmd:              exec.Command(path),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Output: os.Stderr,
			Level:  hclogLevel(m.logger),
		}),
	})

	m.clients = append(m.clients, client)

	return client, nil
}

func (m *Manager) connect(client *goplugin.Client, path string) (LoadedPlugin, error) {
	rpcClient, err := client.Client()

	if err != nil {
		return LoadedPlugin{}, err
	}

	raw, err := rpcClient.Dispense(plugin.CommandsPluginName)

	if err != nil {
		return LoadedPlugin{}, err
	}

	provider, ok := raw.(plugin.CommandProvider)

	if !ok {
		return LoadedPlugin{}, fmt.Errorf("plugin does not implement CommandProvider")
	}

	commands, err := provider.Commands()

	if err != nil {
		return LoadedPlugin{}, fmt.Errorf("fetching commands: %w", err)
	}

	return LoadedPlugin{
		Name:     strings.TrimPrefix(filepath.Base(path), BinaryPrefix),
		Path:     path,
		Provider: provider,
		Commands: commands,
	}, nil
}

func (m *Manager) Logger() *slog.Logger {
	return m.logger
}

func (m *Manager) Loaded() []LoadedPlugin {
	return m.loaded
}

// Kill stops all plugin processes, and prevents any more from being started. It
// must be called before orca exits, and is safe to call multiple times or
// concurrently.
func (m *Manager) Kill() {
	// Any concurrent callers wait for the first to finish
	m.killOnce.Do(m.kill)
}

func (m *Manager) kill() {
	m.mu.Lock()
	m.killed = true
	clients := m.clients
	m.mu.Unlock()

	// Each plugin gets a couple of seconds to exit gracefully before being force
	// killed, so stop them in parallel to avoid that adding up.
	var wg sync.WaitGroup

	for _, c := range clients {
		wg.Go(c.Kill)
	}

	wg.Wait()
}

// hclogLevel picks the level for go-plugin's own logging. Its debug and info
// output is lifecycle chatter (process started/exited etc.) on every run, so it's
// only shown when orca is logging at debug, otherwise only warnings and errors.
func hclogLevel(l *slog.Logger) hclog.Level {
	ctx := context.Background()

	switch {
	case l.Enabled(ctx, slog.LevelDebug):
		return hclog.Debug
	case l.Enabled(ctx, slog.LevelWarn):
		return hclog.Warn
	case l.Enabled(ctx, slog.LevelError):
		return hclog.Error
	default:
		return hclog.Off
	}
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}
