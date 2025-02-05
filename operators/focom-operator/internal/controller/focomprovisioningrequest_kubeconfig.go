package controller

import (
	"context"
	"fmt"

	focomv1alpha1 "github.com/dekstroza/focom-operator/api/focom/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

// buildConfigFromKubeconfig parses kubeconfig bytes into a *rest.Config
func buildConfigFromKubeconfig(kc []byte) (*rest.Config, error) {
	// Use the standard client-go approach
	cfg, err := clientcmd.NewClientConfigFromBytes(kc)
	if err != nil {
		return nil, err
	}
	return cfg.ClientConfig()
}

/*
**********************************************

	Building the remote client

**********************************************
*/
func (r *FocomProvisioningRequestReconciler) buildRemoteClient(
	ctx context.Context,
	fpr *focomv1alpha1.FocomProvisioningRequest,
) (client.Client, error) {

	// 1. Get the OCloud resource from fpr.Spec
	var oCloud focomv1alpha1.OCloud
	if err := r.Get(ctx, types.NamespacedName{
		Name:      fpr.Spec.OCloudId,
		Namespace: fpr.Spec.OCloudNamespace,
	}, &oCloud); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("OCloud %s not found in namespace %s", fpr.Spec.OCloudId, fpr.Spec.OCloudNamespace)
		}
		return nil, fmt.Errorf("failed to fetch OCloud: %w", err)
	}

	// The OCloud references a secret for kubeconfig
	secretRefName := oCloud.Spec.O2imsSecret.SecretRef.Name
	secretRefNs := oCloud.Spec.O2imsSecret.SecretRef.Namespace
	if secretRefName == "" || secretRefNs == "" {
		return nil, fmt.Errorf("OCloud %s/%s references an empty secretRef", fpr.Spec.OCloudNamespace, fpr.Spec.OCloudId)
	}

	// 2. Fetch the secret
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: secretRefName, Namespace: secretRefNs}, &secret); err != nil {
		return nil, fmt.Errorf("cannot read secretRef %s/%s: %w", secretRefNs, secretRefName, err)
	}

	kubeconfigData := secret.Data["kubeconfig"]
	if len(kubeconfigData) == 0 {
		return nil, fmt.Errorf("secret %s/%s missing 'kubeconfig' key", secretRefNs, secretRefName)
	}

	// 3. Build REST config from the kubeconfig
	restConfig, err := buildConfigFromKubeconfig(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig from secret %s/%s: %w", secretRefNs, secretRefName, err)
	}

	// 4. Create a new cluster.Client
	remoteCluster, err := cluster.New(restConfig, func(o *cluster.Options) {
		// Use the same scheme as your Reconciler (or a custom one if needed)
		o.Scheme = r.Scheme
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create remote cluster: %w", err)
	}

	return remoteCluster.GetClient(), nil
}
