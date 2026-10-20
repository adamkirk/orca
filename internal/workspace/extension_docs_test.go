package workspace

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_ExtensionTypesAreDocumented keeps the extensions reference in step with
// the types: each needs its own section, listing every property it takes.
func Test_ExtensionTypesAreDocumented(t *testing.T) {
	contents, err := os.ReadFile("../../docs/configuration/extensions.md")
	require.NoError(t, err)

	// Each type's section starts with a "### `type`" heading
	sections := map[string]string{}

	for _, section := range strings.Split(string(contents), "\n### ")[1:] {
		heading, body, _ := strings.Cut(section, "\n")
		sections[strings.Trim(heading, "` ")] = body
	}

	for name, kind := range extensionKinds {
		t.Run(string(name), func(t *testing.T) {
			body, ok := sections[string(name)]

			require.True(t, ok, "docs/configuration/extensions.md needs a '### `%s`' section", name)

			for _, field := range kind.fields {
				assert.Contains(t, body, fmt.Sprintf("| `%s` |", field), "the '%s' section doesn't document '%s'", name, field)
			}
		})
	}
}
