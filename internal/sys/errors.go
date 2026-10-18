package sys

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

type ErrVersionNotFound struct {
	Version string
}

func (err ErrVersionNotFound) Error() string {
	return fmt.Sprintf("version did not exist: %s", err.Version)
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

type ErrInvalidArchive struct{}

func (err ErrInvalidArchive) Error() string {
	return "invalid archive used"
}

type ErrUnknownTool struct {
	Tool string
}

func (err ErrUnknownTool) Error() string {
	return fmt.Sprintf("unknown tool: %s", err.Tool)
}

type ErrNotImplemented struct {
	Msg string
}

func (err ErrNotImplemented) Error() string {
	return fmt.Sprintf("not implemented: %s", err.Msg)
}
