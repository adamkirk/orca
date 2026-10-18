package git

import "fmt"

type ErrCommandExecutionFailed struct {
	Msg string
}

func (err ErrCommandExecutionFailed) Error() string {
	return fmt.Sprintf("command execution failed: %s", err.Msg)
}

type ErrCouldNotDetermineCurrentBranch struct{}

func (err ErrCouldNotDetermineCurrentBranch) Error() string {
	return "could not determine current branch"
}

type ErrNoBranchesFound struct{}

func (err ErrNoBranchesFound) Error() string {
	return "no branches found"
}
