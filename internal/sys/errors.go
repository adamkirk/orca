package sys

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidArchive          = errors.New("invalid archive used")
	ErrNotImplemented          = errors.New("not implemented")
	ErrVersionNotFound         = errors.New("version not found")
	ErrUnsupportedOS           = errors.New("unsupported OS")
	ErrUnsupportedArchitecture = errors.New("unsupported architecture")
	ErrUnknownTool             = errors.New("unknown tool")
)

type ToolsNotFoundError struct {
	Errs []ToolNotFoundError
}

func (err ToolsNotFoundError) Error() string {
	tools := make([]string, len(err.Errs))

	for i, e := range err.Errs {
		tools[i] = e.Tool
	}

	return fmt.Sprintf("tools not available on PATH: %s", strings.Join(tools, ", "))
}

type ToolNotFoundError struct {
	Tool string
}

func (err ToolNotFoundError) Error() string {
	return fmt.Sprintf("tool not available on PATH: %s", err.Tool)
}
