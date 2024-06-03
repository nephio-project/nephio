/*
Copyright 2023 The Nephio Authors.

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
	"time"

	"github.com/nephio-project/nephio/controllers/pkg/cluster"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	vaultClient "github.com/nephio-project/nephio/controllers/pkg/vault-client"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	vault "github.com/hashicorp/vault/api"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	reconcilerinterface.Register("workloadidentity", &reconciler{})
}

type LoginPayload struct {
	Role string `json:"role"`
	JWT  string `json:"jwt"`
}

type AuthResponse struct {
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c any) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	r.Client = mgr.GetClient()

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("BootstrapSpireController").
		For(&capiv1beta1.Cluster{}).
		Complete(r)
}

type reconciler struct {
	client.Client
}

// r.List --> gets us cluster name list

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cl := &capiv1beta1.Cluster{}
	err := r.Get(ctx, req.NamespacedName, cl)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch Cluster")
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Add your reconciliation logic here
	log.Info("Reconciling Cluster", "cluster", cl.Name)

	// Fetch the ConfigMap from the current cluster
	configMapName := types.NamespacedName{Name: "spire-bundle", Namespace: "spire"}
	configMap := &v1.ConfigMap{}
	err = r.Get(ctx, configMapName, configMap)
	if err != nil {
		log.Error(err, "unable to fetch ConfigMap")
		return reconcile.Result{}, err
	}

	secrets := &v1.SecretList{}
	if err := r.List(ctx, secrets); err != nil {
		msg := "cannot list secrets"
		log.Error(err, msg)
		return ctrl.Result{}, errors.Wrap(err, msg)
	}

	vaultAddress := types.NamespacedName{Name: "vault-config", Namespace: "default"}
	vaultconfigMap := &v1.ConfigMap{}
	err = r.Get(ctx, vaultAddress, vaultconfigMap)
	if err != nil {
		log.Error(err, "unable to fetch ConfigMap")
		return reconcile.Result{}, err
	}

	vaultAddr := vaultconfigMap.Data["VAULT_ADDR"]

	jwtSVID, err := resource.GetJWT(ctx)
	if err != nil {
		log.Error(err, "Unable to get jwtSVID")
	}
	fmt.Println("JWTTT: ", jwtSVID)

	clientToken, err := vaultClient.AuthenticateToVault(vaultAddr, jwtSVID.Marshal(), "dev")
	if err != nil {
		log.Error(err, "Error authenticating to Vault:")
	}

	log.Info("Successfully authenticated to Vault.", "Client token:", clientToken)

	fmt.Println(clientToken)

	config := vault.DefaultConfig()
	config.Address = vaultAddr
	client, err := vault.NewClient(config)
	if err != nil {
		log.Error(err, "Unable to create Vault client:")
	}

	client.SetToken(clientToken)

	for _, secret := range secrets.Items {
		if strings.Contains(secret.GetName(), cl.Name) {
			secret := secret
			fmt.Println("TESTING IFFFFFFF")
			vaultClient.StoreKubeconfig(ctx, secret, client, "/kubeconfigs/"+cl.Name, cl.Name)
		}
	}

	kubeconfig, err := vaultClient.FetchKubeconfig(client, "secret/kubeconfigs/"+cl.Name, cl.Name)
	if err != nil {
		log.Error(err, "Error retrieving secret:")
	}

	decodedKubeConfig, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		fmt.Println("Error decoding base64:", err)
	}

	log.Info("Secret retrieved:", "Secret for cluster", cl.Name)

	Client, err := createK8sClientFromKubeconfig(decodedKubeConfig)

	createK8sSATokenResources(Client)
	if err != nil {
		fmt.Println("Error creating K8s", err)
	}

	kubeconfigCM, err := r.createKubeconfigConfigMap(ctx, Client, cl.Name)
	if err != nil {
		fmt.Println("Error creating K8s kubeconfig configmap", err)
	}

	r.Update(ctx, kubeconfigCM)

	err = updateClusterListConfigMap(Client, cl.Name)
	if err != nil {
		fmt.Println("Cluster list could not be updated...: ", err)
	}

	// Get the spire-server service
	spireService := &v1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "spire-server", Namespace: "spire"}, spireService)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get spire-server service: %v", err)
	}

	// Get the ClusterIP
	clusterIP := spireService.Spec.ClusterIP

	// Get the port
	var port string
	if len(spireService.Spec.Ports) > 0 {
		port = fmt.Sprint(spireService.Spec.Ports[0].Port)
	}

	// Construct the service address
	serviceAddress := fmt.Sprintf("%s:%s", clusterIP, port)

	fmt.Printf("SPIRE Server service address: %s\n", serviceAddress)

	spireAgentCM, err := createSpireAgentConfigMap(Client, "spire-agent", "spire", cl.Name, serviceAddress, port)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get spireAgent ConfigMap: %v", err)
	}

	for _, secret := range secrets.Items {
		if strings.Contains(secret.GetName(), cl.Name) {
			secret := secret // required to prevent gosec warning: G601 (CWE-118): Implicit memory aliasing in for loop
			clusterClient, ok := cluster.Cluster{Client: r.Client}.GetClusterClient(&secret)
			if ok {
				clusterClient, ready, err := clusterClient.GetClusterClient(ctx)
				if err != nil {
					msg := "cannot get clusterClient"
					log.Error(err, msg)
					return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
				}
				if !ready {
					log.Info("cluster not ready")
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}

				remoteNamespace := configMap.Namespace
				ns := &v1.Namespace{}
				if err = clusterClient.Get(ctx, types.NamespacedName{Name: remoteNamespace}, ns); err != nil {
					if resource.IgnoreNotFound(err) != nil {
						msg := fmt.Sprintf("cannot get namespace: %s", remoteNamespace)
						log.Error(err, msg)
						return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
					}
					msg := fmt.Sprintf("namespace: %s, does not exist, retry...", remoteNamespace)
					log.Info(msg)
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}

				newcr := configMap.DeepCopy()

				newcr.ResourceVersion = ""
				newcr.UID = ""
				newcr.Namespace = remoteNamespace

				newAgentConf := spireAgentCM.DeepCopy()
				newAgentConf.ResourceVersion = ""
				newAgentConf.UID = ""
				newAgentConf.Namespace = remoteNamespace
				log.Info("secret info", "secret", newcr.Annotations)
				log.Info("configMap info", "configMap", newAgentConf.Annotations)
				if err := clusterClient.Apply(ctx, newcr); err != nil {
					msg := fmt.Sprintf("cannot apply secret to cluster %s", cl.Name)
					log.Error(err, msg)
					return ctrl.Result{}, errors.Wrap(err, msg)
				}
				if err := clusterClient.Apply(ctx, newAgentConf); err != nil {
					msg := fmt.Sprintf("cannot apply ConfigMap to cluster %s", cl.Name)
					log.Error(err, msg)
					return ctrl.Result{}, errors.Wrap(err, msg)
				}
			}
		}

	}

	return reconcile.Result{}, nil
}

func createK8sClientFromKubeconfig(kubeconfigData []byte) (*kubernetes.Clientset, error) {
	// Load the kubeconfig from the decoded data
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	return clientset, nil
}

func createK8sSATokenResources(clientset *kubernetes.Clientset) error {
	// Create ServiceAccount
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "spire-kubeconfig",
			Namespace: "spire",
		},
	}
	_, err := clientset.CoreV1().ServiceAccounts("spire").Create(context.TODO(), sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %v", err)
	}

	// Create ClusterRole
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-reader",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "nodes"},
				Verbs:     []string{"get"},
			},
		},
	}
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), cr, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ClusterRole: %v", err)
	}

	// Create ClusterRoleBinding for system:auth-delegator
	crbAuthDelegator := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "spire-agent-tokenreview-binding",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "spire-kubeconfig",
				Namespace: "spire",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		},
	}
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crbAuthDelegator, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ClusterRoleBinding (auth-delegator): %v", err)
	}

	// Create ClusterRoleBinding for pod-reader
	crbPodReader := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "spire-agent-pod-reader-binding",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "spire-kubeconfig",
				Namespace: "spire",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "pod-reader",
		},
	}
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crbPodReader, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ClusterRoleBinding (pod-reader): %v", err)
	}

	// Create Secret
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-sa-secret",
			Namespace: "spire",
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": "spire-kubeconfig",
			},
		},
		Type: v1.SecretTypeServiceAccountToken,
	}
	_, err = clientset.CoreV1().Secrets("spire").Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Secret: %v", err)
	}

	return nil
}

func (r *reconciler) createKubeconfigConfigMap(ctx context.Context, clientset *kubernetes.Clientset, clustername string) (*v1.ConfigMap, error) {

	cmName := types.NamespacedName{Name: "kubeconfigs", Namespace: "spire"}
	restrictedKC := &v1.ConfigMap{}
	err := r.Get(ctx, cmName, restrictedKC)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing ConfigMap: %v", err)
	}

	// Retrieve the ServiceAccount token
	secret, err := clientset.CoreV1().Secrets("spire").Get(context.TODO(), "agent-sa-secret", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ServiceAccount token: %v", err)
	}
	token := string(secret.Data["token"])

	// Retrieve the cluster's CA certificate
	configMap, err := clientset.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster CA certificate: %v", err)
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
					User:      "spire-kubeconfig",
				},
			},
		},
		Users: []User{
			{
				Name: "spire-kubeconfig",
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
		return nil, fmt.Errorf("failed to create kubeconfig ConfigMap: %v", err)
	}

	// Generate a unique key for the new kubeconfig
	newConfigKey := fmt.Sprintf("kubeconfig-%s", clustername)

	// Add the new kubeconfig to the existing ConfigMap
	if restrictedKC.Data == nil {
		restrictedKC.Data = make(map[string]string)
	}
	restrictedKC.Data[newConfigKey] = string(yamlData)

	err = r.Update(ctx, restrictedKC)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig ConfigMap: %v", err)
	}

	return restrictedKC, nil
}

func updateClusterListConfigMap(clientset *kubernetes.Clientset, clusterName string) error {

	// Get the ConfigMap
	cm, err := clientset.CoreV1().ConfigMaps("spire").Get(context.TODO(), "clusters", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting ConfigMap: %v", err)
	}

	// Get the clusters.conf data
	clustersConf, ok := cm.Data["clusters.conf"]
	if !ok {
		return fmt.Errorf("clusters.conf not found in ConfigMap")
	}

	// Add new cluster
	newCluster := fmt.Sprintf(`
      "%s" = {
         service_account_allow_list = ["spire:spire-agent"]
         kube_config_file = "/run/spire/kubeconfigs/kubeconfig-%s"
      }`, clusterName, clusterName)

	// Insert the new cluster before the last closing brace
	lastBraceIndex := strings.LastIndex(clustersConf, "}")
	if lastBraceIndex != -1 {
		clustersConf = clustersConf[:lastBraceIndex] + newCluster + clustersConf[lastBraceIndex:]
	} else {
		return fmt.Errorf("invalid clusters.conf format")
	}

	// Update the ConfigMap
	cm.Data["clusters.conf"] = clustersConf

	// Apply the changes
	_, err = clientset.CoreV1().ConfigMaps("spire").Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating ConfigMap: %v", err)
	}

	return nil
}

func createSpireAgentConfigMap(clientset *kubernetes.Clientset, name string, namespace string, cluster string, serverAddress string, serverPort string) (*v1.ConfigMap, error) {
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

	// createdConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	// if err != nil {
	// 	return nil, err
	// }

	return configMap, nil
}
