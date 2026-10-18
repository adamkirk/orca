package sys

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/spf13/afero"
)

const hostCtlVersion = "1.1.4"
const sopsVersion = "3.10.2"

type AvailableTool string

const ToolHostCtl AvailableTool = "hostctl"
const ToolSops AvailableTool = "sops"

var allAvailableTools []AvailableTool = []AvailableTool{
	ToolHostCtl,
	ToolSops,
}

func AllAvailableToolsCsv() string {
	asStrings := make([]string, len(allAvailableTools))

	for i, tool := range allAvailableTools {
		asStrings[i] = string(tool)
	}

	return strings.Join(asStrings, ", ")
}

func (t AvailableTool) IsKnown() bool {
	for _, tool := range allAvailableTools {
		if tool == t {
			return true
		}
	}

	return false
}

func (m *Manager) extractExecutableFromArchive(archive []byte, path string) (io.Reader, error) {
	buf := bytes.NewReader(archive)
	gzr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil, ErrInvalidArchive

		// return any other error
		case err != nil:
			return nil, err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		if header.Name == path {
			return tr, nil
		}
	}
}

func (m *Manager) getExecutableFromArchiveUrl(url string, executableName string) ([]byte, error) {
	archive, err := m.getFileFromUrl(url)

	executable, err := m.extractExecutableFromArchive(archive, executableName)

	if err != nil {
		return nil, err
	}

	return io.ReadAll(executable)
}

func (m *Manager) getFileFromUrl(url string) ([]byte, error) {

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	contents, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	return contents, nil
}

func (m *Manager) getHostctlOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "macOS", nil
	case "linux":
		return "linux", nil
	case "windows":
		return "windows", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOS, os)
	}
}

func (m *Manager) getHostctlArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "64-bit", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedArchitecture, arch)
	}
}

func (m *Manager) installHostCtl() error {
	os, err := m.getHostctlOSForDownload()

	if err != nil {
		return err
	}

	arch, err := m.getHostctlArchForDownload()

	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://github.com/guumaster/hostctl/releases/download/v%s/hostctl_%s_%s_%s.tar.gz",
		hostCtlVersion,
		hostCtlVersion,
		os,
		arch,
	)

	contents, err := m.getExecutableFromArchiveUrl(url, "hostctl")

	if err != nil {
		return err
	}

	if err := m.fs.MkdirAll(m.toolsDir, 0755); err != nil {
		return err
	}

	targetPath := fmt.Sprintf("%s/%s", m.toolsDir, "hostctl")

	exists, err := afero.Exists(m.fs, targetPath)

	if err != nil {
		return err
	}

	if exists {
		if err := m.fs.Remove(targetPath); err != nil {
			return err
		}
	}

	return afero.WriteFile(m.fs, targetPath, contents, 0755)
}

func (m *Manager) getSopsOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "darwin", nil
	case "linux":
		return "linux", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOS, os)
	}
}

func (m *Manager) getSopsArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedArchitecture, arch)
	}
}

func (m *Manager) installSops() error {
	os, err := m.getSopsOSForDownload()

	if err != nil {
		return err
	}

	arch, err := m.getSopsArchForDownload()

	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://github.com/getsops/sops/releases/download/v%s/sops-v%s.%s.%s",
		sopsVersion,
		sopsVersion,
		os,
		arch,
	)

	contents, err := m.getFileFromUrl(url)

	if err != nil {
		return err
	}

	if err := m.fs.MkdirAll(m.toolsDir, 0755); err != nil {
		return err
	}

	targetPath := fmt.Sprintf("%s/%s", m.toolsDir, "sops")

	exists, err := afero.Exists(m.fs, targetPath)

	if err != nil {
		return err
	}

	if exists {
		if err := m.fs.Remove(targetPath); err != nil {
			return err
		}
	}

	return afero.WriteFile(m.fs, targetPath, contents, 0755)
}

type InstallCommand struct {
	bus.Returns[bus.Void]

	Tool AvailableTool
}

func (InstallCommand) MessageName() string {
	return "sys.install"
}

func (m *Manager) Install(_ context.Context, cmd InstallCommand) error {
	if !cmd.Tool.IsKnown() {
		return usererr.Wrap(
			fmt.Errorf("%w: %s", ErrUnknownTool, string(cmd.Tool)),
			fmt.Sprintf("Unknown tool: %s", string(cmd.Tool)),
		)
	}

	var err error
	switch cmd.Tool {
	case ToolHostCtl:
		err = m.installHostCtl()
	case ToolSops:
		err = m.installSops()
	default:
		return usererr.Wrap(
			fmt.Errorf("%w: installation has not been implemented for this tool", ErrNotImplemented),
			"This tool is not supported!",
		)
	}

	if err != nil {
		return usererr.Wrap(err, "Installation failed!")
	}

	m.tui.Success("Installation successful!")

	return nil
}

func (m *Manager) getOrcaOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "Darwin", nil
	case "linux":
		return "Linux", nil
	case "windows":
		return "Windows", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOS, os)
	}
}

func (m *Manager) getOrcaArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "X86_64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedArchitecture, arch)
	}
}
