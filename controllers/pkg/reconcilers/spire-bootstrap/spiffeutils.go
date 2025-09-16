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

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createSpireAgentConfigMap(name string, namespace string, cluster string, serverAddress string, serverPort string) (*v1.ConfigMap, error) {
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

func getServiceExternalIP(service *v1.Service) (string, error) {

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

func (r *reconciler) createKubeconfigConfigMap(ctx context.Context, clientset *kubernetes.Clientset, clustername string) (*v1.ConfigMap, error) {
	log := log.FromContext(ctx)

	log.Info("Creating Kubeconfig ConfigMap for the cluster", "clusterName", clustername)

	cmName := types.NamespacedName{Name: "kubeconfigs", Namespace: "spire"}
	restrictedKC := &v1.ConfigMap{}
	err := r.Get(ctx, cmName, restrictedKC)
	if err != nil {
		msg := "failed to get existing ConfigMap"
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}

	// Retrieve the ServiceAccount token
	secret, err := clientset.CoreV1().Secrets("spire").Get(ctx, "agent-sa-secret", metav1.GetOptions{})
	if err != nil {
		msg := "failed to get Service Account token"
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}
	token := string(secret.Data["token"])

	// Retrieve the cluster's CA certificate
	configMap, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		msg := "failed to get cluster CA"
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}
	caCert := configMap.Data["ca.crt"]
	caCertEncoded := strings.TrimSpace(base64.StdEncoding.EncodeToString([]byte(caCert)))

	config := KubernetesConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{
				Name: clustername,
				Cluster: ClusterDetail{
					CertificateAuthorityData: caCertEncoded,
					Server:                   clientset.RESTClient().Get().URL().String(),
				},
			},
		},
		Contexts: []Context{
			{
				Name: "spire-kubeconfig@" + clustername,
				Context: ContextDetails{
					Cluster:   clustername,
					Namespace: "spire",
					User:      spirekubeconfig,
				},
			},
		},
		Users: []User{
			{
				Name: spirekubeconfig,
				User: UserDetail{
					Token: token,
				},
			},
		},
		CurrentContext: "spire-kubeconfig@" + clustername,
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		msg := "failed to create kubeconfig CM"
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}

	// Generate a unique key for the new kubeconfig
	newConfigKey := fmt.Sprintf("kubeconfig-%s", clustername)

	// Add the new kubeconfig to the existing ConfigMap
	if restrictedKC.Data == nil {
		restrictedKC.Data = make(map[string]string)
	}
	restrictedKC.Data[newConfigKey] = string(yamlData)

	log.Info("Kubeconfig added to the ConfigMap successfully", "clusterName", clustername)

	return restrictedKC, nil
}

func (r *reconciler) updateClusterListConfigMap(ctx context.Context, clusterName string) error {
	log := log.FromContext(ctx)

	log.Info("Updating Cluster List...", "ClusterName", clusterName)

	// Get the ConfigMap
	cm := &v1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "spire",
		Name:      "clusters",
	}, cm); err != nil {
		msg := "failed to create kubeconfig CM"
		log.Error(err, msg)
		return errors.Wrap(err, msg)
	}

	// Get the clusters.conf data
	clustersConf, ok := cm.Data["clusters.conf"]
	if !ok {
		// Initialize with basic structure if not exists
		clustersConf = "clusters = {\n}"
	}

	// Remove any initial whitespace if present
	clustersConf = strings.TrimPrefix(clustersConf, "|")
	clustersConf = strings.TrimSpace(clustersConf)

	// Check if cluster already exists
	if strings.Contains(clustersConf, fmt.Sprintf(`"%s"`, clusterName)) {
		return nil
	}

	// Add new cluster with proper indentation
	newCluster := fmt.Sprintf(`      "%s" = {
        service_account_allow_list = ["spire:spire-agent"]
        kube_config_file = "/run/spire/kubeconfigs/kubeconfig-%s"
      }`, clusterName, clusterName)

	// Insert the new cluster before the last closing brace
	lastBraceIndex := strings.LastIndex(clustersConf, "}")
	if lastBraceIndex != -1 {
		clustersConf = clustersConf[:lastBraceIndex] + newCluster + "\n" + clustersConf[lastBraceIndex:]
	} else {
		return fmt.Errorf("invalid clusters.conf format")
	}

	// Format the final content with pipe operator and proper indentation
	formattedConf := "|\n    " + strings.ReplaceAll(clustersConf, "\n", "\n    ", )

	// Update the ConfigMap
	cm.Data = map[string]string{
		"clusters.conf": formattedConf,
	}

	// Apply the changes
	if err := r.Update(ctx, cm); err != nil {
		msg := "error updating Cluster List ConfigMap"
		log.Error(err, msg)
		return errors.Wrap(err, msg)
	}

	log.Info("Cluster added to the Cluster List", "clusterName", clusterName)

	return nil
}
