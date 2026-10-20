package tui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Confirm(t *testing.T) {
	tests := []struct {
		answer string
		want   bool
	}{
		{"y\n", true},
		{"Yes\n", true},
		{"  y  \n", true},
		{"n\n", false},
		{"\n", false},
		{"sure\n", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.answer, func(t *testing.T) {
			var out bytes.Buffer
			tui := &Tui{in: bufio.NewReader(strings.NewReader(tt.answer)), std: &out, interactive: true}

			got, err := tui.Confirm("Trust these and continue?")

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, "Trust these and continue? [y/N] ", out.String())
		})
	}
}

func Test_Confirm_NotATerminal(t *testing.T) {
	var out bytes.Buffer

	_, err := NewTui(strings.NewReader("y\n"), &out, &out).Confirm("Trust these and continue?")

	assert.ErrorIs(t, err, ErrNotInteractive)
	assert.Empty(t, out.String(), "nothing is asked")
}

func Test_Indented(t *testing.T) {
	var out bytes.Buffer
	w := NewTui(strings.NewReader(""), &out, &out).Indented("  ")

	for _, chunk := range []string{"added 3 packages", "\nfound 0 vuln", "erabilities\n", "\n", "done"} {
		_, err := w.Write([]byte(chunk))
		require.NoError(t, err)
	}

	assert.Equal(t, "  added 3 packages\n  found 0 vulnerabilities\n\n  done", out.String())
}
