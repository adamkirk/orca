package tui

type ErrUserAbortedExecution struct{}

func (err ErrUserAbortedExecution) Error() string {
	return "user aborted execution"
}
