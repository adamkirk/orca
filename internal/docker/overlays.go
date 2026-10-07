package docker

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/adamkirk/orca/internal/common"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/spf13/afero"
)

const networkOverlaidLabel = "orca.adamkirk.overlay-enabled/network"
const aliasesOverlaidLabel = "orca.adamkirk.overlay-enabled/aliases"

const tlsInjectCertsLabel = "orca.pantoptescloud.tls/inject-certs"

const defaultAliasTemplate = "{{ .Service }}.{{ .Project }}.{{ .Workspace }}.local"

// networkNamePrefix is prepended to the workspace name to name the shared network,
// so each workspace gets its own network and they can run at the same time.
const networkNamePrefix = "orca-ws-"

func networkName(ws *common.Workspace) string {
	return networkNamePrefix + ws.Name
}

type aliasTemplateVariables struct {
	Service   string
	Project   string
	Workspace string
}

type overlayGenerationContext struct {
	ws       *common.Workspace
	p        *common.Project
	original *types.Project
	new      *types.Project
}

func (ogc *overlayGenerationContext) addLabelToService(key string, value string, s *types.ServiceConfig) {
	if len(s.Labels) == 0 {
		s.Labels = types.Labels{}
	}

	s.Labels[key] = value
}

func (ogc *overlayGenerationContext) addAliasesToServiceNetworkConfig(svcName string, s *types.ServiceNetworkConfig) error {
	tplContent := defaultAliasTemplate

	if ogc.ws.OverlayConfig.Network.AliasPattern != "" {
		tplContent = ogc.ws.OverlayConfig.Network.AliasPattern
	}

	tpl, err := template.New("alias").Parse(tplContent)

	if err != nil {
		return err
	}

	b := bytes.NewBuffer([]byte{})
	tpl.Execute(b, aliasTemplateVariables{
		Service:   svcName,
		Project:   ogc.p.Name,
		Workspace: ogc.ws.Name,
	})

	s.Aliases = []string{
		b.String(),
	}

	return nil
}

func (ogc *overlayGenerationContext) AddRootNetworkConfig() {
	if ogc.new.Networks == nil {
		ogc.new.Networks = types.Networks{}
	}

	labels := types.Labels{
		networkOverlaidLabel: "",
	}

	if ogc.ws.OverlayConfig.Network.CreateIn == ogc.p.Name {
		ogc.new.Networks["orca"] = types.NetworkConfig{
			Name:   networkName(ogc.ws),
			Labels: labels,
		}
	} else {
		ogc.new.Networks["orca"] = types.NetworkConfig{
			Name:     networkName(ogc.ws),
			External: true,
			Labels:   labels,
		}
	}
}

func (ogc *overlayGenerationContext) AddServiceNetworkConfig() error {
	if len(ogc.original.Services) == 0 {
		// Nothing to do if there are no services
		return nil
	}

	if ogc.new.Services == nil {
		ogc.new.Services = types.Services{}
	}

	for name := range ogc.original.Services {
		if _, ok := ogc.new.Services[name]; !ok {
			ogc.new.Services[name] = types.ServiceConfig{}
		}

		tmpSvc, _ := ogc.new.Services[name]
		newSvc := &tmpSvc

		if ogc.new.Services[name].Networks == nil {
			newSvc.Networks = map[string]*types.ServiceNetworkConfig{}
		}

		nw := &types.ServiceNetworkConfig{}
		newSvc.Networks["orca"] = nw

		ogc.addLabelToService(networkOverlaidLabel, "", newSvc)

		if !ogc.ws.OverlayConfig.Network.DisableAliases {
			ogc.addLabelToService(aliasesOverlaidLabel, "", newSvc)
			if err := ogc.addAliasesToServiceNetworkConfig(name, nw); err != nil {
				return err
			}
		}

		ogc.new.Services[name] = *newSvc
	}

	return nil
}

type overlayComposeParser interface {
	Parse(paths []string, envFiles []string) (*types.Project, error)
}

type ComposeOverlayGenerator struct {
	fs          afero.Fs
	parser      overlayComposeParser
	overlayDir  string
	tlsCertsDir string
}

func emptyOverlay() *types.Project {
	return &types.Project{}
}
func (cog *ComposeOverlayGenerator) getOverlayPath(ws *common.Workspace, p *common.Project) string {
	return fmt.Sprintf("%s/%s/%s.yaml", cog.overlayDir, ws.Name, p.Name)
}

func (cog *ComposeOverlayGenerator) addNetwork(ctx *overlayGenerationContext) error {
	ctx.AddRootNetworkConfig()

	if err := ctx.AddServiceNetworkConfig(); err != nil {
		return err
	}

	return nil
}

func validateBindMountTarget(path string) error {
	switch path {
	case "":
		return errors.New("cannot be empty")
	case "/":
		return errors.New("cannot be root directory (/)")
	default:
		return nil
	}
}

func (cog *ComposeOverlayGenerator) addTLSInjectionOverlay(ctx *overlayGenerationContext, s string, path string) error {
	svc := ctx.new.Services[s]

	// Ensure it has a trailing slash
	// TODO: Consider validation for this, almost anything could be a valid path, right?
	path = fmt.Sprintf("%s/", strings.TrimSuffix(path, "/"))

	if err := validateBindMountTarget(path); err != nil {
		return common.ErrInvalidValueForOverlayModifier{
			Service: s,
			Message: err.Error(),
			Label:   tlsInjectCertsLabel,
		}
	}

	svc.Volumes = append(svc.Volumes, types.ServiceVolumeConfig{
		Type:   types.VolumeTypeBind,
		Source: fmt.Sprintf("%s/", strings.TrimSuffix(cog.tlsCertsDir, "/")),
		Target: path,
	})

	ctx.new.Services[s] = svc

	return nil
}

func (cog *ComposeOverlayGenerator) addTLSOverlays(ctx *overlayGenerationContext) error {
	for k, s := range ctx.original.Services {
		if v, ok := s.Labels[tlsInjectCertsLabel]; ok {
			if err := cog.addTLSInjectionOverlay(ctx, k, v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cog *ComposeOverlayGenerator) buildOverlay(ctx *overlayGenerationContext) error {
	if ctx.ws.OverlayConfig.Network.Enabled {
		if err := cog.addNetwork(ctx); err != nil {
			return err
		}
	}

	if err := cog.addTLSOverlays(ctx); err != nil {
		return err
	}

	return nil
}

func (cog *ComposeOverlayGenerator) CreateOrRetrieve(ws *common.Workspace, p *common.Project) (string, error) {
	composeProject, err := cog.parser.Parse(
		[]string{
			fmt.Sprintf("%s/%s", p.ProjectDir, p.Config.ComposeFiles.Primary),
		},
		p.EnvFilePaths(),
	)

	if err != nil {
		return "", err
	}

	ctx := &overlayGenerationContext{
		ws:       ws,
		p:        p,
		original: composeProject,
		new:      emptyOverlay(),
	}

	if err := cog.buildOverlay(ctx); err != nil {

		return "", err
	}

	newFile, err := ctx.new.MarshalYAML()

	if err != nil {

		return "", err
	}

	overlayPath := cog.getOverlayPath(ws, p)
	overlayDir := filepath.Dir(overlayPath)

	if err := cog.fs.MkdirAll(overlayDir, 0700); err != nil {
		return "", err
	}

	if err := afero.WriteFile(cog.fs, overlayPath, newFile, 0655); err != nil {
		return "", err
	}

	return overlayPath, nil
}

func NewComposeOverlayGenerator(fs afero.Fs, parser overlayComposeParser, overlayDir string, tlsCertsDir string) *ComposeOverlayGenerator {
	return &ComposeOverlayGenerator{
		fs:          fs,
		parser:      parser,
		overlayDir:  overlayDir,
		tlsCertsDir: tlsCertsDir,
	}
}
