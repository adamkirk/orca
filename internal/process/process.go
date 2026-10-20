package process

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

type Executor struct {
}

type ExecOpt func(*exec.Cmd) error

func ChdirOpt(dir string) ExecOpt {
	return func(cmd *exec.Cmd) error {
		cmd.Dir = dir

		return nil
	}
}

func WithHostIO() ExecOpt {
	return func(cmd *exec.Cmd) error {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		return nil
	}
}

// WithOutput sends the command's output, and its errors, to w, while it still
// reads the host's input.
func WithOutput(w io.Writer) ExecOpt {
	return func(cmd *exec.Cmd) error {
		cmd.Stdout = w
		cmd.Stderr = w
		cmd.Stdin = os.Stdin

		return nil
	}
}

func WithStdout() (ExecOpt, *bytes.Buffer) {
	var stdout bytes.Buffer
	ptr := &stdout
	return func(cmd *exec.Cmd) error {
		cmd.Stdout = ptr

		return nil
	}, ptr
}

func WithStderr() (ExecOpt, *bytes.Buffer) {
	var stderr bytes.Buffer
	ptr := &stderr
	return func(cmd *exec.Cmd) error {
		cmd.Stderr = ptr

		return nil
	}, ptr
}

func (e *Executor) Exec(cmdName string, args []string, opts ...ExecOpt) (err error) {
	cmd := exec.Command(cmdName, args...)

	for _, opt := range opts {
		opt(cmd)
	}

	err = cmd.Run()

	return err
}

func NewExecutor() *Executor {
	return &Executor{}
}
