package mutator

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/nephio-controller-poc/apis/infra/v1alpha1"
	kptcondsdk "github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	corev1 "k8s.io/api/core/v1"
	"reflect"
)

const UpfDeploymentName string = "upf-deployment"

func Run(rl *fn.ResourceList) (bool, error) {
	nfDeployFn := NewMutatorContext()

	var err error

	nfDeployFn.sdk, err = kptcondsdk.New(
		rl,
		&kptcondsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephiodeployv1alpha1.GroupVersion.Identifier(),
				Kind:       nephiodeployv1alpha1.UPFDeploymentKind,
			},
			Watch: map[corev1.ObjectReference]kptcondsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name(),
				}: nfDeployFn.ClusterContextCallBackFn,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(nephioreqv1alpha1.Capacity{}).Name(),
				}: nfDeployFn.CapacityContextCallBackFn,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: nfDeployFn.InterfaceCallBackFn,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.DataNetworkKind,
				}: nfDeployFn.DnnCallBackFn,
			},
			GenerateResourceFn: nfDeployFn.GenerateResourceFn,
		},
	)

	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}

	nfDeployFn.sdk.Run()
	return true, nil
}

func (h *NfDeployFn) ClusterContextCallBackFn(o *fn.KubeObject) error {
	var cluster infrav1alpha1.ClusterContext
	err := o.As(&cluster)
	if err != nil {
		return err
	}
	h.site = *cluster.Spec.SiteCode
	return nil
}

func (h *NfDeployFn) CapacityContextCallBackFn(o *fn.KubeObject) error {
	var capacity nephioreqv1alpha1.Capacity
	err := o.As(&capacity)
	if err != nil {
		return err
	}

	h.SetCapacity(capacity.Spec.MaxDownlinkThroughput, capacity.Spec.MaxUplinkThroughput)
	return nil
}

func (h *NfDeployFn) InterfaceCallBackFn(o *fn.KubeObject) error {
	var itfce *nephioreqv1alpha1.Interface
	err := o.As(&itfce)
	if err != nil {
		return err
	}

	itfcIPAllocStatus := itfce.Status.IPAllocationStatus
	itfcVlanAllocStatus := itfce.Status.VLANAllocationStatus

	itfcConfig := nephiodeployv1alpha1.InterfaceConfig{
		Name: itfce.Name,
		IPv4: &nephiodeployv1alpha1.IPv4{
			Address: itfcIPAllocStatus.AllocatedPrefix,
			Gateway: &itfcIPAllocStatus.Gateway,
		},
		VLANID: &itfcVlanAllocStatus.AllocatedVlanID,
	}

	h.SetInterfaceConfig(itfcConfig, itfce.Spec.NetworkInstance.Name)
	return nil
}

func (h *NfDeployFn) DnnCallBackFn(o *fn.KubeObject) error {
	var dnnReq nephioreqv1alpha1.DataNetwork
	err := o.As(&dnnReq)
	if err != nil {
		return err
	}

	var pools []nephiodeployv1alpha1.Pool
	// TODO: DNN Status API schema needs change. This should be fixed later.
	pools = append(pools, nephiodeployv1alpha1.Pool{Prefix: dnnReq.Status.IPAllocationStatus.AllocatedPrefix})
	dnn := nephiodeployv1alpha1.DataNetwork{
		Name: &dnnReq.Spec.NetworkInstance.Name,
		Pool: pools,
	}

	h.AddDNNToNetworkInstance(dnn, dnnReq.Spec.NetworkInstance.Name)

	return nil
}

func (h *NfDeployFn) GenerateResourceFn(upfDeploymentObj *fn.KubeObject, _ fn.KubeObjects) (*fn.KubeObject, error) {
	var err error

	// TODO: Replace with NewFromKubeObject
	if upfDeploymentObj == nil {
		upfDeploymentObj = fn.NewEmptyKubeObject()
		err = upfDeploymentObj.SetAPIVersion(nephiodeployv1alpha1.GroupVersion.String())
		if err != nil {
			return nil, err
		}
		err = upfDeploymentObj.SetKind(nephiodeployv1alpha1.UPFDeploymentKind)
		if err != nil {
			return nil, err
		}
	}

	upfDeploymentSpec := &nephiodeployv1alpha1.NFDeploymentSpec{}

	upfDeploymentSpec.Capacity = &nephioreqv1alpha1.CapacitySpec{
		MaxUplinkThroughput:   h.capacityMaxUpLinkThroughPut,
		MaxDownlinkThroughput: h.capacityMaxDownLinkThroughPut,
	}

	if err != nil {
		return nil, err
	}

	for networkInstanceName, itfceConfig := range h.GetInterfaceConfigMap() {
		h.AddInterfaceToNetworkInstance(itfceConfig.Name, networkInstanceName)
	}

	upfDeploymentSpec.Interfaces = h.GetAllInterfaceConfig()
	upfDeploymentSpec.NetworkInstances = h.GetAllNetworkInstance()

	//TODO: Use SetSpec() method
	err = upfDeploymentObj.SetNestedField(upfDeploymentSpec, "spec")

	return upfDeploymentObj, err
}
