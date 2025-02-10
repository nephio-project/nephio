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
package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	focomv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/focom/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	ORAN_PROV_REQ_CRD_PATH = "../../test-data/"
	ORAN_PROV_REQ_CRD_FILE = "o2ims.provisioning.oran.org_provisioningrequests.yaml"
)

// TestIntegrationBuildRemoteClient_CreateProvisioningRequest tests building a remote client
// that attempts discovery on a real ephemeral EnvTest server.
func TestIntegrationBuildRemoteClientCreateProvisioningRequest(t *testing.T) {

	// get the o2ims provrequest CRD
    err := fetchCRDFromRepo()
	require.NoError(t, err, "failed to fetch o2ims.provrequest CRD")

	// 1. Start an EnvTest environment for the "remote" cluster
	remoteEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			// Path(s) to the CRDs for your "remote" cluster
			"../../config/crd/bases",
			ORAN_PROV_REQ_CRD_PATH,
		},
	}

	cfgRemote, err := remoteEnv.Start()
	require.NoError(t, err, "failed to start remote envtest")
	defer func() {
		err := remoteEnv.Stop()
		require.NoError(t, err, "failed to stop remote envtest")
	}()

	// 2. Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(cfgRemote)
	require.NoError(t, err, "failed to create Kubernetes clientset")

	// 3. Create a ClusterRoleBinding for the system:anonymous user
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.Background(), &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "allow-anonymous-access",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: "system:anonymous",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin", // Grant full access (or use a custom role with specific permissions)
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create ClusterRoleBinding for system:anonymous")

	apiextensionsClient, err := apiextensionsclient.NewForConfig(cfgRemote)
	require.NoError(t, err, "failed to create API extensions clientset")
	crdList, err := apiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err, "failed to list CRDs")
	fmt.Println("CRDs installed in remote envtest:")
	for _, crd := range crdList.Items {
		fmt.Println(crd.Name)
	}

	// If your remote CRD is installed as a Go type, also register it to remoteEnv.Scheme here if needed
	// e.g. _ = provisioningv1alpha1.AddToScheme(remoteEnv.Scheme)

	// 2. Create a "kubeconfig" from the EnvTest config
	remoteKubeconfig, err := kubeconfigFromEnvTestConfig(cfgRemote)
	require.NoError(t, err, "failed to build kubeconfig from envtest")

	// 3. We'll set up a local fake client that references that Kubeconfig
	localScheme := scheme.Scheme
	_ = focomv1alpha1.AddToScheme(localScheme)

	localClient := fake.NewClientBuilder().
		WithScheme(localScheme).
		WithObjects(
			&focomv1alpha1.OCloud{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ocloud-a",
					Namespace: "default",
				},
				Spec: focomv1alpha1.OCloudSpec{
					O2imsSecret: focomv1alpha1.O2imsSecret{
						SecretRef: focomv1alpha1.SecretRef{
							Name:      "ocloud-cred-a",
							Namespace: "default",
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ocloud-cred-a",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"kubeconfig": remoteKubeconfig,
				},
			},
		).Build()

	// 4. Build the Reconciler
	r := &FocomProvisioningRequestReconciler{
		Client: localClient,
		Scheme: localScheme,
	}

	// 5. Create a local FPR referencing OCloud
	fpr := &focomv1alpha1.FocomProvisioningRequest{
		Spec: focomv1alpha1.FocomProvisioningRequestSpec{
			OCloudId:        "ocloud-a",
			OCloudNamespace: "default",
			TemplateName:    "red-hat-cluster-template",
			TemplateVersion: "1.0.0",
		},
	}

	// 6. Now call buildRemoteClient -> it will attempt discovery on EnvTest
	remoteCl, buildErr := r.buildRemoteClient(context.Background(), fpr)
	require.NoError(t, buildErr)

	// 7. Verify we can create a resource in the remote env
	// If the remote CRD is installed, let's create an unstructured "ProvisioningRequest"
	remoteObj := &unstructuredv1.Unstructured{}
	remoteObj.SetAPIVersion("o2ims.provisioning.oran.org/v1alpha1")
	remoteObj.SetKind("ProvisioningRequest")
	remoteObj.SetName("test-remote")
	// ... set more fields if needed
	err = remoteCl.Create(context.Background(), remoteObj)
	require.NoError(t, err, "should succeed creating the resource in remote envtest cluster")

	removeCRDsFromLocal()
	require.NoError(t, err, "failed to clen up afetr test")
}

// kubeconfigFromEnvTestConfig builds a minimal kubeconfig from an EnvTest *rest.Config
func kubeconfigFromEnvTestConfig(cfg *rest.Config) ([]byte, error) {
	clusterName := "envtest-remote"
	contextName := "default"
	userName := "default-user"

	caData := cfg.CAData
	if len(caData) == 0 && cfg.CAFile != "" {
		raw, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caData = raw
	}

	apiCfg := clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server:                   cfg.Host,
				CertificateAuthorityData: caData,
				InsecureSkipTLSVerify:    cfg.Insecure,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			userName: {
				Token: cfg.BearerToken,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		CurrentContext: contextName,
	}
	return clientcmd.Write(apiCfg)
}

// A function to pull CRDs from another repo (e.g., a remote GitHub repository)
func fetchCRDFromRepo() error {
    // Get the CRD from github
    cmd := exec.Command("curl", "-o", ORAN_PROV_REQ_CRD_PATH + ORAN_PROV_REQ_CRD_FILE,
	"https://raw.githubusercontent.com/nephio-project/api/main/config/crd/bases/" + ORAN_PROV_REQ_CRD_FILE)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to fetch CRD from GitHub: %w", err)
    }
    return nil
}

func removeCRDsFromLocal() error {
    cmd := exec.Command("rm", "-f", ORAN_PROV_REQ_CRD_PATH + ORAN_PROV_REQ_CRD_FILE)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to remove CRD from local: %w", err)
    }
    return nil
}