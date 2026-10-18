package hostsys

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

func (self AvailableTool) IsKnown() bool {
	for _, tool := range allAvailableTools {
		if tool == self {
			return true
		}
	}

	return false
}

func (self *HostSystem) extractExecutableFromArchive(archive []byte, path string) (io.Reader, error) {
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

func (self *HostSystem) getExecutableFromArchiveUrl(url string, executableName string) ([]byte, error) {
	archive, err := self.getFileFromUrl(url)

	executable, err := self.extractExecutableFromArchive(archive, executableName)

	if err != nil {
		return nil, err
	}

	return io.ReadAll(executable)
}

func (self *HostSystem) getFileFromUrl(url string) ([]byte, error) {

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

func (self *HostSystem) getHostctlOSForDownload() (string, error) {
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

func (self *HostSystem) getHostctlArchForDownload() (string, error) {
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

func (self *HostSystem) installHostCtl() error {
	os, err := self.getHostctlOSForDownload()

	if err != nil {
		return err
	}

	arch, err := self.getHostctlArchForDownload()

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

	contents, err := self.getExecutableFromArchiveUrl(url, "hostctl")

	if err != nil {
		return err
	}

	if err := self.fs.MkdirAll(self.toolsDir, 0755); err != nil {
		return err
	}

	targetPath := fmt.Sprintf("%s/%s", self.toolsDir, "hostctl")

	exists, err := afero.Exists(self.fs, targetPath)

	if err != nil {
		return err
	}

	if exists {
		if err := self.fs.Remove(targetPath); err != nil {
			return err
		}
	}

	return afero.WriteFile(self.fs, targetPath, contents, 0755)
}

func (self *HostSystem) getSopsOSForDownload() (string, error) {
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

func (self *HostSystem) getSopsArchForDownload() (string, error) {
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

func (self *HostSystem) installSops() error {
	os, err := self.getSopsOSForDownload()

	if err != nil {
		return err
	}

	arch, err := self.getSopsArchForDownload()

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

	contents, err := self.getFileFromUrl(url)

	if err != nil {
		return err
	}

	if err := self.fs.MkdirAll(self.toolsDir, 0755); err != nil {
		return err
	}

	targetPath := fmt.Sprintf("%s/%s", self.toolsDir, "sops")

	exists, err := afero.Exists(self.fs, targetPath)

	if err != nil {
		return err
	}

	if exists {
		if err := self.fs.Remove(targetPath); err != nil {
			return err
		}
	}

	return afero.WriteFile(self.fs, targetPath, contents, 0755)
}

type InstallCommand struct {
	bus.Returns[bus.Void]

	Tool AvailableTool
}

func (InstallCommand) MessageName() string {
	return "sys.install"
}

func (self *HostSystem) Install(_ context.Context, cmd InstallCommand) error {
	if !cmd.Tool.IsKnown() {
		return self.tui.RecordIfError(fmt.Sprintf("Unknown tool: %s", string(cmd.Tool)), common.ErrUnknownTool{
			Tool: string(cmd.Tool),
		})
	}

	var err error
	switch cmd.Tool {
	case ToolHostCtl:
		err = self.installHostCtl()
	case ToolSops:
		err = self.installSops()
	default:
		return self.tui.RecordIfError("This tool is not supported!", common.ErrNotImplemented{
			Msg: "installation has not been implemented for this tool",
		})
	}

	if err != nil {
		self.tui.Error("Installation failed!")
		return err
	}

	self.tui.Success("Installation successful!")

	return nil
}

func (self *HostSystem) getOrcaOSForDownload() (string, error) {
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

func (self *HostSystem) getOrcaArchForDownload() (string, error) {
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
