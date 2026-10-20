package extension

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownService       = errors.New("unknown service")
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

	// Empty for one of the workspace's hooks
	Project   string
	Hook      string
	Extension string
	Err       error
}

func (err HookFailedError) Error() string {
	where := "the workspace"

	if err.Project != "" {
		where = err.Project
	}

	return fmt.Sprintf("%s's %s hook failed running '%s': %s", where, err.Hook, err.Extension, err.Err)
}

func (err HookFailedError) Unwrap() error {
	return err.Err
}

// RerunCommand is how to run the failed extension again, on its own.
func (err HookFailedError) RerunCommand() string {
	return fmt.Sprintf("orca ext %s %s", err.target(), err.Extension)
}

// RerunHookCommand is how to run the whole hook again.
func (err HookFailedError) RerunHookCommand() string {
	return fmt.Sprintf("orca hooks run %s %s", err.target(), err.Hook)
}

func (err HookFailedError) target() string {
	if err.Project != "" {
		return fmt.Sprintf("-w %s -p %s", err.Workspace, err.Project)
	}

	return fmt.Sprintf("-w %s", err.Workspace)
}
