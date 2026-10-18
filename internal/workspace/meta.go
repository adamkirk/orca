package workspace

type ProjectRepositoryMeta struct {
	SSH  string
	Self bool
}

type ProjectMeta struct {
	Name          string
	Path          string
	WorkspaceName string
	Repository    ProjectRepositoryMeta
}

type WorkspaceMeta struct {
	Name     string
	Path     string
	Projects []ProjectMeta
}
