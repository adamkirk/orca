package git

import (
	"github.com/adamkirk/orca/internal/common"
	"github.com/adamkirk/orca/internal/process"
)

func (g *Git) Clone(repoURL string, target string, dir string) error {
	if repoURL == "" || target == "" {
		return common.ErrInvalidInput{
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
