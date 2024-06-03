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

package resource

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateSpireAgentConfigMap(name string, namespace string, cluster string, serverAddress string, serverPort string) (*v1.ConfigMap, error) {
	configMapData := map[string]string{
		"agent.conf": `
agent {
  data_dir = "/run/spire"
  log_level = "DEBUG"
  server_address = "` + serverAddress + `"
  server_port = "` + serverPort + `"
  socket_path = "/run/spire/sockets/spire-agent.sock"
  trust_bundle_path = "/run/spire/bundle/bundle.crt"
  trust_domain = "example.org"
}

plugins {
  NodeAttestor "k8s_psat" {
    plugin_data {
      cluster = "` + cluster + `"
    }
  }

  KeyManager "memory" {
    plugin_data {
    }
  }

  WorkloadAttestor "k8s" {
    plugin_data {
      skip_kubelet_verification = true
    }
  }
}
`,
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: configMapData,
	}

	return configMap, nil
}

func GetServiceExternalIP(service *v1.Service) (string, error) {

	// Check LoadBalancer ingress
	if len(service.Status.LoadBalancer.Ingress) > 0 {
		if service.Status.LoadBalancer.Ingress[0].IP != "" {
			return service.Status.LoadBalancer.Ingress[0].IP, nil
		}
		if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
			return service.Status.LoadBalancer.Ingress[0].Hostname, nil
		}
	}

	// Check external IPs
	if len(service.Spec.ExternalIPs) > 0 {
		return service.Spec.ExternalIPs[0], nil
	}

	return "", nil
}
