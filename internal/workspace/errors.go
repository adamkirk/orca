package workspace

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidExecutionContext     = errors.New("invalid execution context")
	ErrCouldNotDetermineWorkspace  = errors.New("could not determine workspace")
	ErrUnknownWorkspace            = errors.New("unknown workspace")
	ErrUnknownProject              = errors.New("unknown project")
	ErrInvalidInput                = errors.New("invalid input")
	ErrFileNotFound                = errors.New("file not found")
	ErrWorkspaceAlreadyExists      = errors.New("workspace already exists")
	ErrDirectoryAlreadyExists      = errors.New("directory already exists")
	ErrDirectoryDoesNotMatchOrigin = errors.New("directory does not contain a repo with the expected origin")
	ErrProjectConfigNotFound       = errors.New("project config not found")
	ErrWorkspaceConfigNotFound     = errors.New("workspace config not found")
	ErrConfigIsNotValidYAML        = errors.New("config is not valid yaml")
	ErrUnknownExtension            = errors.New("unknown extension")
	ErrBranchNotFound              = errors.New("branch not found")
	ErrLibraryProject              = errors.New("library project")
	ErrCircularCloneOrder          = errors.New("projects require, or are inside, each other in a circle")
	ErrEnclosingProjectNotCloned   = errors.New("the project it's inside isn't cloned")
)

type InvalidWorkspaceConfigError struct {
	Path     string
	Problems []string
}

func (err InvalidWorkspaceConfigError) Error() string {
	return fmt.Sprintf("workspace config at '%s' is invalid: %s", err.Path, strings.Join(err.Problems, "; "))
}

type InvalidProjectConfigError struct {
	Path     string
	Problems []string
}

func (err InvalidProjectConfigError) Error() string {
	return fmt.Sprintf("project config at '%s' is invalid: %s", err.Path, strings.Join(err.Problems, "; "))
}

type InvalidPreferencesError struct {
	Path     string
	Problems []string
}

func (err InvalidPreferencesError) Error() string {
	return fmt.Sprintf("preferences at '%s' are invalid: %s", err.Path, strings.Join(err.Problems, "; "))
}
