package git

import (
	"fmt"
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
)

func (m *Manager) Clone(repoURL string, target string, dir string) error {
	if repoURL == "" || target == "" {
		return fmt.Errorf("%w to 'git.clone': 'repoURL' and 'target' cannot be empty", workspace.ErrInvalidInput)
	}
	return m.exec.Exec(
		"git",
		[]string{
			"clone",
			repoURL,
			target,
		},
		withDir(dir, process.WithHostIO())...,
	)
}
