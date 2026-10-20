package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

const (
	colourRed   = "#ff0000"
	colourGreen = "#00ff00"
)

type Tui struct {
	in  *bufio.Reader
	std io.Writer
	err io.Writer

	// Whether in is a terminal, so questions can be asked
	interactive bool

	errStyle     lipgloss.Style
	successStyle lipgloss.Style
}

func (t *Tui) NewLine() {
	fmt.Fprintln(t.std, "")
}

func (t *Tui) Error(msg ...string) {
	fmt.Fprintln(t.err, t.errStyle.Render(msg...))
}

func (t *Tui) Success(msg ...string) {
	fmt.Fprintln(t.std, t.successStyle.Render(msg...))
}

func (t *Tui) Info(msgs ...string) {
	for _, msg := range msgs {
		fmt.Fprintln(t.std, msg)
	}
}

// Confirm asks a yes or no question, which is no unless the answer is y or yes.
// It fails with ErrNotInteractive when there's no terminal to ask in, rather
// than waiting for an answer that will never come.
func (t *Tui) Confirm(question string) (bool, error) {
	if !t.interactive {
		return false, ErrNotInteractive
	}

	fmt.Fprintf(t.std, "%s [y/N] ", question)

	answer, err := t.in.ReadString('\n')

	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading the answer: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (t *Tui) Table(header []string, rows [][]string) {
	var (
		headerStyle = lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)
		cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	)

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		}).
		Headers(header...).
		Rows(rows...)

	fmt.Fprintln(t.std, tbl.Render())
}

// NewTui writes to std and err, and asks questions on in, if it's a terminal.
func NewTui(in io.Reader, std io.Writer, err io.Writer) *Tui {
	return &Tui{
		in:           bufio.NewReader(in),
		interactive:  isTerminal(in),
		std:          std,
		err:          err,
		errStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color(colourRed)),
		successStyle: lipgloss.NewStyle().Foreground(lipgloss.Color(colourGreen)),
	}
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)

	return ok && term.IsTerminal(int(f.Fd()))
}
