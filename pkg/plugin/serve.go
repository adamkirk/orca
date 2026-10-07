package plugin

import (
	"os"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
)

// orphanCheckInterval is how often the plugin checks whether orca is still
// running. It's the longest a plugin can outlive orca.
const orphanCheckInterval = 250 * time.Millisecond

// Serve starts the plugin's gRPC server and blocks until orca disconnects. It
// should be the only thing a plugin's main function does.
func Serve(impl CommandProvider) {
	go exitWhenOrphaned(os.Getppid(), os.Getppid, orphanCheckInterval, func() { os.Exit(1) })

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			CommandsPluginName: &CommandProviderPlugin{Impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}

// exitWhenOrphaned calls exit if orca dies without stopping the plugin, which
// happens when orca is killed in a way it can't handle (e.g. SIGKILL, or SIGPIPE
// from `orca ... | head`). Orca normally stops plugins itself, this is a fallback.
//
// When a process's parent dies, it's re-parented (to init or a subreaper), so its
// parent pid changes. On Windows the parent pid never changes, so this is a no-op.
func exitWhenOrphaned(originalPpid int, getppid func() int, interval time.Duration, exit func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if getppid() != originalPpid {
			exit()
			return
		}
	}
}
