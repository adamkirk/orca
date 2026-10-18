package tls_test

import (
	"context"
	"testing"

	"github.com/adamkirk/orca/internal/tls"
	"github.com/adamkirk/orca/internal/workspace"
	tls_mocks "github.com/adamkirk/orca/tests/mocks/tls"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_CertificateManager_Generate(t *testing.T) {
	tests := []struct {
		name         string
		in           tls.GenerateCommand
		expect       error
		configureTui func(*testing.T, *tls_mocks.MockTui)
		workspace    *workspace.Workspace
		reRun        bool
	}{
		{
			name: "success - no certificates",
			in: tls.GenerateCommand{
				Workspace: "test",
			},
			expect: nil,
			configureTui: func(tt *testing.T, tui *tls_mocks.MockTui) {
				tui.EXPECT().Info([]string{"Generating Root certificate if required..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/key.pem"})
				tui.EXPECT().Success([]string{"Key /some/path/key.pem created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/cert.pem"})
				tui.EXPECT().Success([]string{"Certificate /some/path/cert.pem created successfully!"})
			},
			workspace: &workspace.Workspace{
				Projects: []workspace.Project{},
			},
		},
		{
			name: "success - generate 1 certificate",
			in: tls.GenerateCommand{
				Workspace: "test",
			},
			expect: nil,
			configureTui: func(tt *testing.T, tui *tls_mocks.MockTui) {
				tui.EXPECT().Info([]string{"Generating Root certificate if required..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/key.pem"})
				tui.EXPECT().Success([]string{"Key /some/path/key.pem created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/cert.pem"})
				tui.EXPECT().Success([]string{"Certificate /some/path/cert.pem created successfully!"})

				tui.EXPECT().NewLine()
				tui.EXPECT().Info([]string{"Generating certificate for '*.example.com'..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/certs/_.example.com.key"})
				tui.EXPECT().Success([]string{"Key /some/path/certs/_.example.com.key created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/certs/_.example.com.cert"})
				tui.EXPECT().Success([]string{"Certificate /some/path/certs/_.example.com.cert created successfully!"})
			},
			workspace: &workspace.Workspace{
				Projects: []workspace.Project{
					{
						Config: workspace.ProjectConfig{
							TLSCertificates: []string{
								"*.example.com",
							},
						},
					},
				},
			},
		},

		{
			name: "success - re-run",
			in: tls.GenerateCommand{
				Workspace: "test",
			},
			expect: nil,
			reRun:  true,
			configureTui: func(tt *testing.T, tui *tls_mocks.MockTui) {
				tui.EXPECT().Info([]string{"Generating Root certificate if required..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/key.pem"})
				tui.EXPECT().Success([]string{"Key /some/path/key.pem created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/cert.pem"})
				tui.EXPECT().Success([]string{"Certificate /some/path/cert.pem created successfully!"})

				tui.EXPECT().NewLine()
				tui.EXPECT().Info([]string{"Generating certificate for '*.example.com'..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/certs/_.example.com.key"})
				tui.EXPECT().Success([]string{"Key /some/path/certs/_.example.com.key created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/certs/_.example.com.cert"})
				tui.EXPECT().Success([]string{"Certificate /some/path/certs/_.example.com.cert created successfully!"})

				tui.EXPECT().Info([]string{"Key /some/path/key.pem already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Certificate /some/path/cert.pem already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Key /some/path/certs/_.example.com.key already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Certificate /some/path/certs/_.example.com.cert already exists, skipping creation."})
				// tui.EXPECT().Info([]string{"Key /some/path/key.pem already exists, skipping creation."})
			},
			workspace: &workspace.Workspace{
				Projects: []workspace.Project{
					{
						Config: workspace.ProjectConfig{
							TLSCertificates: []string{
								"*.example.com",
							},
						},
					},
				},
			},
		},

		{
			name: "success - re-run with multiple",
			in: tls.GenerateCommand{
				Workspace: "test",
			},
			expect: nil,
			reRun:  true,
			configureTui: func(tt *testing.T, tui *tls_mocks.MockTui) {
				tui.EXPECT().Info([]string{"Generating Root certificate if required..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/key.pem"})
				tui.EXPECT().Success([]string{"Key /some/path/key.pem created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/cert.pem"})
				tui.EXPECT().Success([]string{"Certificate /some/path/cert.pem created successfully!"})

				tui.EXPECT().NewLine()
				tui.EXPECT().Info([]string{"Generating certificate for '*.example.com'..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/certs/_.example.com.key"})
				tui.EXPECT().Success([]string{"Key /some/path/certs/_.example.com.key created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/certs/_.example.com.cert"})
				tui.EXPECT().Success([]string{"Certificate /some/path/certs/_.example.com.cert created successfully!"})

				tui.EXPECT().Info([]string{"Generating certificate for 'blah.test'..."})
				tui.EXPECT().Info([]string{"Creating key: /some/path/certs/blah.test.key"})
				tui.EXPECT().Success([]string{"Key /some/path/certs/blah.test.key created successfully!"})
				tui.EXPECT().Info([]string{"Creating certificate: /some/path/certs/blah.test.cert"})
				tui.EXPECT().Success([]string{"Certificate /some/path/certs/blah.test.cert created successfully!"})

				tui.EXPECT().Info([]string{"Key /some/path/key.pem already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Certificate /some/path/cert.pem already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Key /some/path/certs/_.example.com.key already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Certificate /some/path/certs/_.example.com.cert already exists, skipping creation."})

				tui.EXPECT().Info([]string{"Key /some/path/certs/blah.test.key already exists, skipping creation."})
				tui.EXPECT().Info([]string{"Certificate /some/path/certs/blah.test.cert already exists, skipping creation."})
			},
			workspace: &workspace.Workspace{
				Projects: []workspace.Project{
					{
						Config: workspace.ProjectConfig{
							TLSCertificates: []string{
								"*.example.com",
								"blah.test",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			tui := tls_mocks.NewMockTui(t)
			if test.configureTui != nil {
				test.configureTui(tt, tui)
			}

			fs := afero.NewMemMapFs()

			cm := tls.NewCertificateManager(fs, tui, "/some/path")

			// As resolved by the ExecutionContextMiddleware
			ctx := workspace.WithExecutionContext(context.Background(), workspace.ExecutionContext{Workspace: test.workspace})

			assert.Equal(tt, test.expect, cm.Generate(ctx, test.in))

			if !test.reRun {
				return
			}

			assert.Equal(tt, test.expect, cm.Generate(ctx, test.in))
		})
	}
}
