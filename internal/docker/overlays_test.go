package docker

import (
	"testing"

	"github.com/adamkirk/orca/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddRootNetworkConfig(t *testing.T) {
	tests := []struct {
		name           string
		project        string
		expectExternal bool
	}{
		{
			name:           "createIn project creates the network",
			project:        "local-env",
			expectExternal: false,
		},
		{
			name:           "other projects join it as an external network",
			project:        "api",
			expectExternal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &overlayGenerationContext{
				ws: &common.Workspace{
					Name: "acme",
					OverlayConfig: common.OverlayConfig{
						Network: common.NetworkOverlayConfig{
							Enabled:  true,
							CreateIn: "local-env",
						},
					},
				},
				p:   &common.Project{Name: tt.project},
				new: emptyOverlay(),
			}

			ctx.AddRootNetworkConfig()

			nw, ok := ctx.new.Networks["orca"]
			require.True(t, ok)

			// Named per workspace, so multiple workspaces can run at once
			assert.Equal(t, "orca-ws-acme", nw.Name)
			assert.Equal(t, tt.expectExternal, bool(nw.External))
		})
	}
}
