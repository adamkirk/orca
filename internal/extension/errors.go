package extension

import (
	"errors"
	"fmt"

	"github.com/adamkirk/orca/internal/workspace"
)

var (
	ErrUnknownService       = errors.New("unknown service")
	ErrNoProject            = errors.New("no project")
	ErrNoServices           = errors.New("no services to run the extension in")
	ErrProjectNotCloned     = errors.New("project isn't cloned")
	ErrUnexpectedArgs       = errors.New("extension doesn't take arguments")
	ErrSourceNotFound       = errors.New("source not found")
	ErrTargetExists         = errors.New("target already exists, and isn't the expected symlink")
	ErrSymlinksNotSupported = errors.New("the filesystem doesn't support symlinks")
	ErrTrustDeclined        = errors.New("the extensions weren't trusted")
	ErrTrustNeeded          = errors.New("the extensions need trusting")
)

// HookFailedError is an extension that failed while a hook was running it.
type HookFailedError struct {
	Workspace string

	// The project the extension belongs to, and ran in
	Project   string
	Hook      string
	Extension string
	Err       error
}

// Describe says which extension failed, and in which hook, e.g. "'env', in
// api's clone.post hook".
func (err HookFailedError) Describe() string {
	if err.Hook == workspace.HookInitPost.String() {
		return fmt.Sprintf("%s's '%s', in the workspace's %s hook", err.Project, err.Extension, err.Hook)
	}

	return fmt.Sprintf("'%s', in %s's %s hook", err.Extension, err.Project, err.Hook)
}

func (err HookFailedError) Error() string {
	return fmt.Sprintf("%s failed: %s", err.Describe(), err.Err)
}

func (err HookFailedError) Unwrap() error {
	return err.Err
}

// RerunCommand is how to run the failed extension again, on its own.
func (err HookFailedError) RerunCommand() string {
	return fmt.Sprintf("orca ext -w %s -p %s %s", err.Workspace, err.Project, err.Extension)
}

// RerunHookCommand is how to run the whole hook again. init.post is the
// workspace's, so it doesn't need the project.
func (err HookFailedError) RerunHookCommand() string {
	if err.Hook == workspace.HookInitPost.String() {
		return fmt.Sprintf("orca hooks run -w %s %s", err.Workspace, err.Hook)
	}

	return fmt.Sprintf("orca hooks run -w %s -p %s %s", err.Workspace, err.Project, err.Hook)
}
