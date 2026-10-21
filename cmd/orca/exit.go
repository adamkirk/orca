package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/adamkirk/orca/internal/extension"
	"github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
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

// handleErrors wraps a command handler. Handlers return their errors rather
// than printing them, and they're shown to the user here, so cobra is told not
// to print them again.
func handleErrors(f runEHandlerFunc) runEHandlerFunc {
	return func(cmd *cobra.Command, args []string) error {
		err := f(cmd, args)

		if err != nil {
			slog.Debug("command failed", "err", err)

			reportError(svcContainer.GetTui(), err)

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
		}

		return err
	}
}

type errorPrinter interface {
	Error(msg ...string)
}

// reportError shows a failed command's error to the user, once. That's the
// most specific message attached with usererr.Wrap, followed by what caused it,
// or just the error if there's no message.
func reportError(out errorPrinter, err error) {
	// The user chose to stop, so there's nothing to tell them
	if errors.Is(err, tui.ErrUserAbortedExecution) {
		return
	}

	msg, cause, hasMsg := usererr.Message(err)

	if hasMsg {
		out.Error(msg)
	}

	// Config problems can surface from anywhere a workspace is loaded, so
	// they're shown here, in full, rather than in each handler
	if invalidCfg, ok := errors.AsType[workspace.InvalidWorkspaceConfigError](err); ok {
		reportProblems(out, fmt.Sprintf("Workspace config '%s' is invalid:", invalidCfg.Path), invalidCfg.Problems)

		return
	}

	if invalidProject, ok := errors.AsType[workspace.InvalidProjectConfigError](err); ok {
		reportProblems(out, fmt.Sprintf("Project config '%s' is invalid:", invalidProject.Path), invalidProject.Problems)

		return
	}

	if invalidPrefs, ok := errors.AsType[workspace.InvalidPreferencesError](err); ok {
		reportProblems(out, fmt.Sprintf("Preferences file '%s' is invalid:", invalidPrefs.Path), invalidPrefs.Problems)

		return
	}

	if !hasMsg {
		out.Error(fmt.Sprintf("Error: %s", err))
	} else {
		out.Error(fmt.Sprintf("  %s", cause))
	}

	// Whatever the extension said, say where it was running, and how to try
	// it again once it's fixed
	if hookErr, ok := errors.AsType[extension.HookFailedError](err); ok {
		out.Error(fmt.Sprintf("This was %s. Nothing after it ran.", hookErr.Describe()))
		out.Error(fmt.Sprintf("Once it's fixed, re-run it with '%s', or the whole hook with '%s'.", hookErr.RerunCommand(), hookErr.RerunHookCommand()))
	}
}

func reportProblems(out errorPrinter, title string, problems []string) {
	out.Error(title)

	for _, problem := range problems {
		out.Error(fmt.Sprintf("  - %s", problem))
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

	if !completing {
		fmt.Fprintln(os.Stderr, "Error:", err)
		exit(1)
	}

	// Completion never shows errors, it offers nothing instead. Without a
	// directive, shells would offer files.
	fmt.Printf(":%d\n", cobra.ShellCompDirectiveNoFileComp)
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
