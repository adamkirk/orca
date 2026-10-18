package git

import (
	"github.com/adamkirk/orca/internal/process"
	"github.com/adamkirk/orca/internal/workspace"
)

func (g *Git) Clone(repoURL string, target string, dir string) error {
	if repoURL == "" || target == "" {
		return workspace.ErrInvalidInput{
			To:  "git.clone",
			Msg: "'repoURL' and 'target' cannot be empty",
		}
	}
	return g.exec.Exec(
		"git",
		[]string{
			"clone",
			repoURL,
			target,
		},
		withDir(dir, process.WithHostIO())...,
	)
}
