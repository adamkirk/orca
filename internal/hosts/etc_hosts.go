package hosts

import (
	"fmt"

	"github.com/adamkirk/orca/internal/workspace"
	"github.com/txn2/txeh"
)

type EtcHostsManager struct {
	hostsFile *txeh.Hosts
}

func (ehm *EtcHostsManager) SyncForWorkspace(ws *workspace.Workspace) error {
	hosts := ws.GetUniqueHosts()

	marker := fmt.Sprintf("adamkirk/orca:%s", ws.Name)

	ehm.hostsFile.RemoveByComment(marker)
	// Clean up entries written before the repo moved from panoptescloud/orca
	ehm.hostsFile.RemoveByComment(fmt.Sprintf("panoptescloud/orca:%s", ws.Name))

	ehm.hostsFile.AddHostsWithComment("127.0.0.1", hosts, marker)

	return ehm.hostsFile.Save()
}

func NewEtcHostsManager() (*EtcHostsManager, error) {
	hostsFile, err := txeh.NewHostsDefault()

	if err != nil {
		return nil, err
	}

	return &EtcHostsManager{
		hostsFile: hostsFile,
	}, nil
}
