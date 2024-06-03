package spirebootstrap

type KubernetesConfig struct {
	APIVersion     string    `yaml:"apiVersion"`
	Kind           string    `yaml:"kind"`
	Clusters       []Cluster `yaml:"clusters"`
	Contexts       []Context `yaml:"contexts"`
	Users          []User    `yaml:"users"`
	CurrentContext string    `yaml:"current-context"`
}

type Cluster struct {
	Name    string        `yaml:"name"`
	Cluster ClusterDetail `yaml:"cluster"`
}

type ClusterDetail struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	Server                   string `yaml:"server"`
}

type Context struct {
	Name    string         `yaml:"name"`
	Context ContextDetails `yaml:"context"`
}

type ContextDetails struct {
	Cluster   string `yaml:"cluster"`
	Namespace string `yaml:"namespace"`
	User      string `yaml:"user"`
}

type User struct {
	Name string     `yaml:"name"`
	User UserDetail `yaml:"user"`
}

type UserDetail struct {
	Token string `yaml:"token"`
}
