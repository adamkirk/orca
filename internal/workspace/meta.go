package workspace

type ProjectMeta struct {
	Name          string
	Path          string
	WorkspaceName string
}

type WorkspaceMeta struct {
	Name     string
	Path     string
	Projects []ProjectMeta
}
