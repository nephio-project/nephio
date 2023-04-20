package mutatordownstream

import (
	"fmt"
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/nephio-controller-poc/apis/infra/v1alpha1"
	clusterctxtlibv1alpha1 "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/clustercontext/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/lib/fnruntime"
	nadlibv1 "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/nad/v1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/ipam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mutatorCtx struct {
	fnruntime       fnruntime.FnRuntime
	masterInterface string
	cniType         string
}

func Run(rl *fn.ResourceList) (bool, error) {
	m := mutatorCtx{}

	m.fnruntime = fnruntime.NewDownstream(
		rl,
		&fnruntime.DownstreamRuntimeConfig{
			For: fnruntime.DownstreamRuntimeForConfig{
				ObjectRef: corev1.ObjectReference{
					APIVersion: nadv1.SchemeGroupVersion.Identifier(),
					Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
				},
				GenerateFn: m.generateNadFn,
			},
			Owns: map[corev1.ObjectReference]struct{}{
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPAllocationKind,
				}: {},
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: {},
			},
			Watch: map[corev1.ObjectReference]fnruntime.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name(),
				}: m.ClusterContextCallbackFn,
			},
		},
	)
	m.fnruntime.Run()
	return true, nil
}

func (r *mutatorCtx) ClusterContextCallbackFn(o *fn.KubeObject) error {
	fmt.Println("IN ClusterContextCallbackFn")

	if o.GetAPIVersion() == infrav1alpha1.SchemeBuilder.GroupVersion.Identifier() && o.GetKind() == reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name() {
		clusterContext := clusterctxtlibv1alpha1.NewMutator(o.String())
		cluster, err := clusterContext.UnMarshal()
		if err != nil {
			return err
		}
		r.masterInterface = cluster.Spec.CNIConfig.MasterInterface
		r.cniType = cluster.Spec.CNIConfig.CNIType
	}
	return nil
}

func (r *mutatorCtx) generateNadFn(resources map[corev1.ObjectReference]fn.KubeObject) (*fn.KubeObject, error) {

	// loop throough resource get ip, vlan and masterInterface and generate a nad
	fmt.Println("IN generate NAD")
	meta := metav1.ObjectMeta{
		Name: "dummyName",
	}
	fn.Log("size of resource", len(resources))
	for i, o := range resources {
		meta.Name = o.GetName()
		fn.Log("KEY", i)
		fn.Log("Value", resources[i])
	}

	return nadlibv1.NewGenerator(meta, nadlibv1.NetworkAttachmentDefinitionSpec{}).ParseKubeObject()
}
