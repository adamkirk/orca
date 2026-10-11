package common

import (
	"fmt"
	"strings"
)

type ErrToolsNotFoundOnSystem struct {
	Errs []ErrToolNotFoundOnSystem
}

func (err ErrToolsNotFoundOnSystem) Error() string {
	tools := make([]string, len(err.Errs))

	for i, e := range err.Errs {
		tools[i] = e.Tool
	}

	return fmt.Sprintf("tools not available on PATH: %s", strings.Join(tools, ", "))
}

type ErrToolNotFoundOnSystem struct {
	Tool string
}

func (err ErrToolNotFoundOnSystem) Error() string {
	return fmt.Sprintf("tool not available on PATH: %s", err.Tool)
}

type ErrInvalidStrategy struct {
	Msg string
}

func (err ErrInvalidStrategy) Error() string {
	return fmt.Sprintf("invalid strategy: %s", err.Msg)
}

type ErrInvalidExecutionContext struct {
	Msg string
}

func (err ErrInvalidExecutionContext) Error() string {
	return fmt.Sprintf("invalid execution context: %s", err.Msg)
}

type ErrCommandExecutionFailed struct {
	Msg string
}

func (err ErrCommandExecutionFailed) Error() string {
	return fmt.Sprintf("command execution failed: %s", err.Msg)
}

type ErrUserAbortedExecution struct{}

func (err ErrUserAbortedExecution) Error() string {
	return "user aborted execution"
}

type ErrCouldNotDetermineCurrentBranch struct{}

func (err ErrCouldNotDetermineCurrentBranch) Error() string {
	return "could not determine current branch"
}

type ErrNoBranchesFound struct{}

func (err ErrNoBranchesFound) Error() string {
	return "no branches found"
}

type ErrInvalidSemVerValue struct {
	Value string
}

func (err ErrInvalidSemVerValue) Error() string {
	return fmt.Sprintf("invalid semver value given: %s", err.Value)
}

type ErrVersionNotFound struct {
	Version string
}

func (err ErrVersionNotFound) Error() string {
	return fmt.Sprintf("version did not exist: %s", err.Version)
}

type ErrUnexpectedApiError struct {
	Msg string
}

func (err ErrUnexpectedApiError) Error() string {
	return fmt.Sprintf("unexpected API error: %s", err.Msg)
}

type ErrRateLimitedByApi struct {
	Api string
}

func (err ErrRateLimitedByApi) Error() string {
	return fmt.Sprintf("rate limit exceeded for '%s'", err.Api)
}

type ErrUnsupportedOS struct {
	OS string
}

func (err ErrUnsupportedOS) Error() string {
	return fmt.Sprintf("unsupported OS: %s", err.OS)
}

type ErrUnsupportedArchitecture struct {
	Arch string
}

func (err ErrUnsupportedArchitecture) Error() string {
	return fmt.Sprintf("unsupported architecture: %s", err.Arch)
}

type ErrUnsupportedProvisioner struct {
	Message string
}

func (err ErrUnsupportedProvisioner) Error() string {
	return fmt.Sprintf("unsupported provisioner: %s", err.Message)
}

type ErrInvalidArchive struct{}

func (err ErrInvalidArchive) Error() string {
	return "invalid archive used"
}

type ErrUnknownWorkspace struct {
	Name string
}

func (err ErrUnknownWorkspace) Error() string {
	return fmt.Sprintf("unknown workspace: %s", err.Name)
}

type ErrCouldNotDetermineWorkspace struct {
	Message string
}

func (err ErrCouldNotDetermineWorkspace) Error() string {
	return fmt.Sprintf("could not determine workspace: %s", err.Message)
}

type ErrUnknownProject struct {
	Name string
}

func (err ErrUnknownProject) Error() string {
	return fmt.Sprintf("unknown project: %s", err.Name)
}

type ErrUnknownService struct {
	Name string
}

func (err ErrUnknownService) Error() string {
	return fmt.Sprintf("unknown service: %s", err.Name)
}

type ErrUnknownTool struct {
	Tool string
}

func (err ErrUnknownTool) Error() string {
	return fmt.Sprintf("unknown tool: %s", err.Tool)
}

type ErrArgumentRequired struct {
	Name  string
	Index int
}

func (err ErrArgumentRequired) Error() string {
	return fmt.Sprintf("the '%s' argument must be supplied at position %d", err.Name, err.Index)
}

type ErrTooManyArguments struct {
	Expected int
}

func (err ErrTooManyArguments) Error() string {
	return fmt.Sprintf("too many arguments supplied, expected only %d", err.Expected)
}

type ErrInvalidInput struct {
	To  string
	Msg string
}

func (err ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input to '%s': %s", err.To, err.Msg)
}

type ErrNotImplemented struct {
	Msg string
}

func (err ErrNotImplemented) Error() string {
	return fmt.Sprintf("not implemented: %s", err.Msg)
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

type ErrRepositoryAlreadyCloned struct {
	RepoURL string
}

func (err ErrRepositoryAlreadyCloned) Error() string {
	return fmt.Sprintf("repository already cloned: %s", err.RepoURL)
}

type ErrInvalidProjectDependency struct {
	ProjectName        string
	InvalidRequirement string
}

func (err ErrInvalidProjectDependency) Error() string {
	return fmt.Sprintf("project '%s' requires unknown project: '%s'", err.ProjectName, err.InvalidRequirement)
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

type ErrInvalidValueForOverlayModifier struct {
	Label   string
	Message string
	Service string
}

func (err ErrInvalidValueForOverlayModifier) Error() string {
	svc := ""

	if err.Service != "" {
		svc = fmt.Sprintf("(service=%s) ", err.Service)
	}

	return fmt.Sprintf("invalid value for label '%s' modifier in compose file %s: %s", err.Label, svc, err.Message)
}

type ErrUnknownExtension struct {
	Name string
}

func (err ErrUnknownExtension) Error() string {
	return fmt.Sprintf("extension '%s' not found in project", err.Name)
}
