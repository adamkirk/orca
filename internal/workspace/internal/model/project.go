package model

type LoaderPropertyCondition struct {
	Name  string
	Value any
}

type LoaderCondition struct {
	OS       string
	Arch     string
	Property LoaderPropertyCondition
}

type ExtraComposeFile struct {
	Path string
	When []LoaderCondition
}

type ComposeFiles struct {
	Primary string
	Extras  []ExtraComposeFile
}

type Property struct {
	Name    string
	Type    string
	Default any
}

type Extension struct {
	Name        string
	Chdir       string
	Command     string
	Service     string
	DefaultArgs []string `yaml:"defaultArgs"`
}

type EnvFile struct {
	Path string
}
type ExampleFileProvisioner struct {
	Src    string
	Target string
	Type   string
}

type Provisioner struct {
	ExampleFile *ExampleFileProvisioner `yaml:"exampleFile"`
}

type ProjectConfig struct {
	ComposeFiles    ComposeFiles `yaml:"composeFiles"`
	EnvFiles        []EnvFile    `yaml:"envFiles"`
	Properties      []Property
	Hosts           []string
	TLSCertificates []string `yaml:"tlsCerts"`
	Extensions      []Extension
	Provisioners    []Provisioner `yaml:"provisioners"`
}
