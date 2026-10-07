package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/spf13/cobra"
)

// Orca may have plugin processes running, which won't exit by themselves when
// orca does. To make sure they're always stopped, nothing should call os.Exit (or
// cobra.CheckErr, which calls it) directly:
//
//   - Command handlers return errors, which are passed back up to run().
//   - Fatal errors go through checkErr or exit, which stop the plugins first.
//   - Signals are caught by handleSignals, which does the same.
//
// This is enforced by Test_NoDirectExits.

// handleErrors wraps a command handler. Handlers report errors to the user
// themselves (usually via the tui), so cobra is told not to print them again.
func handleErrors(f runEHandlerFunc) runEHandlerFunc {
	return func(cmd *cobra.Command, args []string) error {
		err := f(cmd, args)

		if err != nil {
			// Set as a debug level here as it should already be logged earlier
			// in the stack
			slog.Debug("unhandled error", "err", err)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
		}

		return err
	}
}

// exit stops any running plugins and then exits with the given code.
func exit(code int) {
	svcContainer.KillPlugins()
	os.Exit(code)
}

// checkErr is a replacement for cobra.CheckErr, that stops any running plugins
// before exiting.
func checkErr(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, "Error:", err)
	exit(1)
}

// signalExitCode is set once a signal has been received, to the code orca will
// exit with.
var signalExitCode atomic.Int32

// handleSignals stops any running plugins and exits when orca is interrupted or
// terminated, as the default behaviour would kill orca without any cleanup. The
// returned function stops handling signals.
func handleSignals() func() {
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		select {
		case sig := <-sigs:
			slog.Debug("received signal, exiting", "signal", sig)

			code := 1
			if s, ok := sig.(syscall.Signal); ok {
				// Conventional exit code for being killed by a signal
				code = 128 + int(s)
			}

			// Set before stopping the plugins, so that run() knows any errors from
			// running plugin commands are because of this.
			signalExitCode.Store(int32(code))
			exit(code)
		case <-done:
		}
	}()

	return func() {
		signal.Stop(sigs)
		close(done)
	}
}

// The flag getters below only fail if the flag doesn't exist on the command,
// which is a bug rather than a user error, so they panic rather than making every
// handler check for an error. Panics still unwind through run(), so plugins are
// stopped.

func mustGetString(cmd *cobra.Command, name string) string {
	v, err := cmd.Flags().GetString(name)
	mustFlag(name, err)

	return v
}

func mustGetBool(cmd *cobra.Command, name string) bool {
	v, err := cmd.Flags().GetBool(name)
	mustFlag(name, err)

	return v
}

func mustGetInt(cmd *cobra.Command, name string) int {
	v, err := cmd.Flags().GetInt(name)
	mustFlag(name, err)

	return v
}

func mustFlag(name string, err error) {
	if err != nil {
		panic(fmt.Sprintf("reading flag %q: %v", name, err))
	}
}
