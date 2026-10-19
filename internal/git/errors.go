package git

import "errors"

var (
	ErrCommandExecutionFailed         = errors.New("command execution failed")
	ErrCouldNotDetermineCurrentBranch = errors.New("could not determine current branch")
	ErrNoBranchesFound                = errors.New("no branches found")
)
