/*
Copyright 2025 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
