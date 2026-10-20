package extension

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
)

// target is where an extension runs: in a project, or in the workspace's
// directory when project is nil.
type target struct {
	ws      *workspace.Workspace
	project *workspace.Project
}

// dir is the directory the extension's paths are relative to, and its
// commands run in.
func (t target) dir() string {
	if t.project != nil {
		return t.project.ProjectDir
	}

	return t.ws.Dir()
}

// execute runs the extension in the target, with args for a command. What it
// shows, and a host command's output, start with indent, so they can be shown
// under the hook that's running them.
func (m *Manager) execute(t target, ext workspace.Extension, args []string, indent string) error {
	if t.project != nil && !t.project.IsRegistered {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrProjectNotCloned, t.project.Name),
			fmt.Sprintf("%s isn't cloned; clone it with 'orca ws clone -p %s'.", t.project.Name, t.project.Name),
		)
	}

	switch spec := ext.Spec.(type) {
	case workspace.CommandSpec:
		return m.executeCommand(t, spec, args, indent)
	case workspace.SyncFileSpec:
		if err := noArgs(ext, args); err != nil {
			return err
		}

		return m.syncFile(t, spec, indent)
	case workspace.SymlinkSpec:
		if err := noArgs(ext, args); err != nil {
			return err
		}

		return m.symlink(t, spec, indent)
	default:
		panic(fmt.Sprintf("unknown extension spec %T", spec))
	}
}

func noArgs(ext workspace.Extension, args []string) error {
	if len(args) == 0 {
		return nil
	}

	return usererr.Wrap(
		fmt.Errorf("%w: %s", ErrUnexpectedArgs, ext.Name),
		fmt.Sprintf("'%s' is a %s extension, so it doesn't take arguments!", ext.Name, ext.Spec.Type()),
	)
}

// commandArgs is the command split on spaces, followed by the args, or its
// default args when there are none.
func commandArgs(spec workspace.CommandSpec, args []string) []string {
	cmdArgs := strings.Split(spec.Command, " ")

	if len(args) > 0 {
		return append(cmdArgs, args...)
	}

	return append(cmdArgs, spec.DefaultArgs...)
}

func (m *Manager) executeCommand(t target, spec workspace.CommandSpec, args []string, indent string) error {
	cmdArgs := commandArgs(spec, args)

	if spec.Service == "" {
		output := process.WithHostIO()

		if indent != "" {
			output = process.WithOutput(m.tui.Indented(indent))
		}

		return m.executor.Exec(cmdArgs[0], cmdArgs[1:], output, process.ChdirOpt(inDir(t.dir(), spec.Chdir)))
	}

	if t.project == nil || t.project.IsLibrary() {
		return usererr.Wrap(
			fmt.Errorf("%w: service '%s'", ErrNoServices, spec.Service),
			fmt.Sprintf("This extension runs in the '%s' service, so it has to run in a project with a compose stack; pass one with -p.", spec.Service),
		)
	}

	exists, err := m.compose.DoesSvcExist(t.ws, t.project, spec.Service)

	if err != nil {
		return err
	}

	if !exists {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrUnknownService, spec.Service),
			fmt.Sprintf("%s has no service '%s' to run the extension in!", t.project.Name, spec.Service),
		)
	}

	running, err := m.compose.IsSvcRunning(t.ws, t.project, spec.Service)

	if err != nil {
		return err
	}

	// Run by hand, it's interactive. From a hook, it's shown under the hook,
	// and doesn't need a terminal.
	var output io.Writer

	if indent != "" {
		output = m.tui.Indented(indent)
	}

	if !running {
		return m.compose.Run(t.ws, t.project, spec.Service, cmdArgs, output)
	}

	return m.compose.Exec(t.ws, t.project, spec.Service, cmdArgs, output)
}

// inDir resolves path against dir, unless it's already absolute.
func inDir(dir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(dir, path)
}

// syncFile creates the target from the source, if it doesn't exist. An existing
// target is never changed.
func (m *Manager) syncFile(t target, spec workspace.SyncFileSpec, indent string) error {
	source, err := renderSource(t.ws, spec.Source)

	if err != nil {
		return err
	}

	sourcePath := inDir(t.dir(), source)
	targetPath := inDir(t.dir(), spec.Target)

	targetExists, err := afero.Exists(m.fs, targetPath)

	if err != nil {
		return err
	}

	if targetExists {
		m.tui.Info(indent + fmt.Sprintf("%s already exists, leaving it as it is.", spec.Target))
		return nil
	}

	info, err := m.fs.Stat(sourcePath)

	if errors.Is(err, fs.ErrNotExist) {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrSourceNotFound, sourcePath),
			fmt.Sprintf("Can't create %s, as %s doesn't exist!", spec.Target, sourcePath),
		)
	}

	if err != nil {
		return err
	}

	contents, err := afero.ReadFile(m.fs, sourcePath)

	if err != nil {
		return err
	}

	if err := m.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	if err := afero.WriteFile(m.fs, targetPath, contents, info.Mode().Perm()); err != nil {
		return err
	}

	m.tui.Info(indent + fmt.Sprintf("Created %s from %s.", spec.Target, source))

	return nil
}

// symlink creates the target as an absolute symlink to the source. A target
// that's already that symlink is left alone, anything else is an error, rather
// than being replaced.
func (m *Manager) symlink(t target, spec workspace.SymlinkSpec, indent string) error {
	linker, canLink := m.fs.(afero.Linker)
	reader, canRead := m.fs.(afero.LinkReader)
	lstater, canLstat := m.fs.(afero.Lstater)

	if !canLink || !canRead || !canLstat {
		return ErrSymlinksNotSupported
	}

	source, err := renderSource(t.ws, spec.Source)

	if err != nil {
		return err
	}

	sourcePath := inDir(t.dir(), source)
	targetPath := inDir(t.dir(), spec.Target)

	info, _, err := lstater.LstatIfPossible(targetPath)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		// Created below
	case err != nil:
		return err
	case info.Mode()&fs.ModeSymlink != 0:
		existing, err := reader.ReadlinkIfPossible(targetPath)

		if err != nil {
			return err
		}

		if filepath.Clean(existing) == filepath.Clean(sourcePath) {
			m.tui.Info(indent + fmt.Sprintf("%s already links to %s.", spec.Target, sourcePath))
			return nil
		}

		fallthrough
	default:
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrTargetExists, targetPath),
			fmt.Sprintf("%s already exists, and isn't a link to %s, so it was left alone. Remove it, and run the extension again.", targetPath, sourcePath),
		)
	}

	if err := m.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	if err := linker.SymlinkIfPossible(sourcePath, targetPath); err != nil {
		return err
	}

	m.tui.Info(indent + fmt.Sprintf("Linked %s to %s.", spec.Target, sourcePath))

	return nil
}
