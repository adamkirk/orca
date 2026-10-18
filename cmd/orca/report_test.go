package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/stretchr/testify/assert"
)

type stubPrinter struct {
	lines []string
}

func (p *stubPrinter) Error(msg ...string) {
	p.lines = append(p.lines, msg...)
}

func Test_ReportError(t *testing.T) {
	cause := errors.New("permission denied")
	invalidCfg := workspace.InvalidWorkspaceConfigError{
		Path:     "/code/orca.workspace.yaml",
		Problems: []string{"name is required"},
	}

	tests := []struct {
		name   string
		err    error
		expect []string
	}{
		{
			name:   "without a message, shows the error",
			err:    cause,
			expect: []string{"Error: permission denied"},
		},
		{
			name:   "shows the message, then its cause",
			err:    usererr.Wrap(cause, "Failed to save!"),
			expect: []string{"Failed to save!", "  permission denied"},
		},
		{
			name:   "only shows the most specific message",
			err:    usererr.Wrap(usererr.Wrap(cause, "Failed to checkout branch"), "Failed to checkout"),
			expect: []string{"Failed to checkout branch", "  permission denied"},
		},
		{
			name: "shows the problems with an invalid workspace config",
			err:  usererr.Wrap(fmt.Errorf("loading: %w", invalidCfg), "Failed to load workspace config!"),
			expect: []string{
				"Failed to load workspace config!",
				"Workspace config '/code/orca.workspace.yaml' is invalid:",
				"  - name is required",
			},
		},
		{
			name: "says nothing when the user aborted",
			err:  fmt.Errorf("choosing a branch: %w", tui.ErrUserAbortedExecution),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &stubPrinter{}

			reportError(out, tt.err)

			assert.Equal(t, tt.expect, out.lines)
		})
	}
}
