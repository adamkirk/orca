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
	"github.com/adamkirk/orca/internal/common"
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

func (hs *HostSystem) extractExecutableFromArchive(archive []byte, path string) (io.Reader, error) {
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
			return nil, common.ErrInvalidArchive{}

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

func (hs *HostSystem) getExecutableFromArchiveUrl(url string, executableName string) ([]byte, error) {
	archive, err := hs.getFileFromUrl(url)

	executable, err := hs.extractExecutableFromArchive(archive, executableName)

	if err != nil {
		return nil, err
	}

	return io.ReadAll(executable)
}

func (hs *HostSystem) getFileFromUrl(url string) ([]byte, error) {

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

func (hs *HostSystem) getHostctlOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "macOS", nil
	case "linux":
		return "linux", nil
	case "windows":
		return "windows", nil
	default:
		return "", common.ErrUnsupportedOS{
			OS: os,
		}
	}
}

func (hs *HostSystem) getHostctlArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "64-bit", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", common.ErrUnsupportedArchitecture{
			Arch: arch,
		}
	}
}

func (hs *HostSystem) installHostCtl() error {
	os, err := hs.getHostctlOSForDownload()

	if err != nil {
		return err
	}

	arch, err := hs.getHostctlArchForDownload()

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

	contents, err := hs.getExecutableFromArchiveUrl(url, "hostctl")

	if err != nil {
		return err
	}

	if err := hs.fs.MkdirAll(hs.toolsDir, 0755); err != nil {
		return err
	}

	targetPath := fmt.Sprintf("%s/%s", hs.toolsDir, "hostctl")

	exists, err := afero.Exists(hs.fs, targetPath)

	if err != nil {
		return err
	}

	if exists {
		if err := hs.fs.Remove(targetPath); err != nil {
			return err
		}
	}

	return afero.WriteFile(hs.fs, targetPath, contents, 0755)
}

func (hs *HostSystem) getSopsOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "darwin", nil
	case "linux":
		return "linux", nil
	default:
		return "", common.ErrUnsupportedOS{
			OS: os,
		}
	}
}

func (hs *HostSystem) getSopsArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", common.ErrUnsupportedArchitecture{
			Arch: arch,
		}
	}
}

func (hs *HostSystem) installSops() error {
	os, err := hs.getSopsOSForDownload()

	if err != nil {
		return err
	}

	arch, err := hs.getSopsArchForDownload()

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

	contents, err := hs.getFileFromUrl(url)

	if err != nil {
		return err
	}

	if err := hs.fs.MkdirAll(hs.toolsDir, 0755); err != nil {
		return err
	}

	targetPath := fmt.Sprintf("%s/%s", hs.toolsDir, "sops")

	exists, err := afero.Exists(hs.fs, targetPath)

	if err != nil {
		return err
	}

	if exists {
		if err := hs.fs.Remove(targetPath); err != nil {
			return err
		}
	}

	return afero.WriteFile(hs.fs, targetPath, contents, 0755)
}

type InstallCommand struct {
	bus.Returns[bus.Void]

	Tool AvailableTool
}

func (InstallCommand) MessageName() string {
	return "sys.install"
}

func (hs *HostSystem) Install(_ context.Context, cmd InstallCommand) error {
	if !cmd.Tool.IsKnown() {
		return hs.tui.RecordIfError(fmt.Sprintf("Unknown tool: %s", string(cmd.Tool)), common.ErrUnknownTool{
			Tool: string(cmd.Tool),
		})
	}

	var err error
	switch cmd.Tool {
	case ToolHostCtl:
		err = hs.installHostCtl()
	case ToolSops:
		err = hs.installSops()
	default:
		return hs.tui.RecordIfError("This tool is not supported!", common.ErrNotImplemented{
			Msg: "installation has not been implemented for this tool",
		})
	}

	if err != nil {
		hs.tui.Error("Installation failed!")
		return err
	}

	hs.tui.Success("Installation successful!")

	return nil
}

func (hs *HostSystem) getOrcaOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "Darwin", nil
	case "linux":
		return "Linux", nil
	case "windows":
		return "Windows", nil
	default:
		return "", common.ErrUnsupportedOS{
			OS: os,
		}
	}
}

func (hs *HostSystem) getOrcaArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "X86_64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", common.ErrUnsupportedArchitecture{
			Arch: arch,
		}
	}
}
