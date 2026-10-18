package workspace

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidExecutionContext    = errors.New("invalid execution context")
	ErrCouldNotDetermineWorkspace = errors.New("could not determine workspace")
)

type ErrUnknownWorkspace struct {
	Name string
}

func (err ErrUnknownWorkspace) Error() string {
	return fmt.Sprintf("unknown workspace: %s", err.Name)
}

type ErrUnknownProject struct {
	Name string
}

func (err ErrUnknownProject) Error() string {
	return fmt.Sprintf("unknown project: %s", err.Name)
}

type ErrInvalidInput struct {
	To  string
	Msg string
}

func (err ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input to '%s': %s", err.To, err.Msg)
}

type ErrFileNotFound struct {
	Path string
}

func (err ErrFileNotFound) Error() string {
	return fmt.Sprintf("file not found: %s", err.Path)
}

type ErrWorkspaceAlreadyExists struct {
	Name string
}

func (err ErrWorkspaceAlreadyExists) Error() string {
	return fmt.Sprintf("workspace already exists: %s", err.Name)
}

type ErrDirectoryAlreadyExists struct {
	Path string
}

func (err ErrDirectoryAlreadyExists) Error() string {
	return fmt.Sprintf("directory already exists: %s", err.Path)
}

type ErrDirectoryDoesNotMatchOrigin struct {
	Path   string
	Origin string
}

func (err ErrDirectoryDoesNotMatchOrigin) Error() string {
	return fmt.Sprintf("directory '%s' does not contain repo with origin: %s", err.Path, err.Origin)
}

type ErrProjectConfigNotFound struct {
	Path string
}

func (err ErrProjectConfigNotFound) Error() string {
	return fmt.Sprintf("project config expected at '%s' was not found", err.Path)
}

type ErrWorkspaceConfigNotFound struct {
	Path string
}

func (err ErrWorkspaceConfigNotFound) Error() string {
	return fmt.Sprintf("workspace config expected at '%s' was not found", err.Path)
}

type ErrConfigIsNotValidYAML struct {
	Path       string
	ParseError string
}

func (err ErrConfigIsNotValidYAML) Error() string {
	return fmt.Sprintf("config at '%s' was not valid yaml: %s", err.Path, err.ParseError)
}

type ErrInvalidWorkspaceConfig struct {
	Path     string
	Problems []string
}

func (err ErrInvalidWorkspaceConfig) Error() string {
	return fmt.Sprintf("workspace config at '%s' is invalid: %s", err.Path, strings.Join(err.Problems, "; "))
}

type ErrUnknownExtension struct {
	Name string
}

func (err ErrUnknownExtension) Error() string {
	return fmt.Sprintf("extension '%s' not found in project", err.Name)
}
