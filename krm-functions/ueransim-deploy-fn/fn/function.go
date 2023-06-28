/*
Copyright 2023 Nephio.

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

package fn

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/iputil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const NetworksAnnotation = "k8s.v1.cni.cncf.io/networks"

type deployFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
}

func Run(rl *fn.ResourceList) (bool, error) {
	myFn := deployFn{}
	var err error
	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: appsv1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(appsv1.Deployment{}).Name(),
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: condkptsdk.ChildInitial,
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPClaimKind,
				}: condkptsdk.ChildInitial,
				{
					APIVersion: corev1.SchemeGroupVersion.Identifier(),
					Kind:       reflect.TypeOf(corev1.ConfigMap{}).Name(),
				}: condkptsdk.ChildLocal,
				{
					APIVersion: corev1.SchemeGroupVersion.Identifier(),
					Kind:       reflect.TypeOf(corev1.Service{}).Name(),
				}: condkptsdk.ChildLocal,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: myFn.WorkloadClusterCallbackFn,
			},
			PopulateOwnResourcesFn: myFn.desiredOwnedResourceList,
			UpdateResourceFn:       myFn.updateResource,
			Root:                   true,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return myFn.sdk.Run()
}

func (f *deployFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
	var err error

	if f.workloadCluster != nil {
		return fmt.Errorf("multiple WorkloadCluster objects found in the kpt package")
	}
	f.workloadCluster, err = ko.KubeObjectToStruct[infrav1alpha1.WorkloadCluster](o)
	if err != nil {
		return err
	}

	// validate check the specifics of the spec, like mandatory fields
	return f.workloadCluster.Spec.Validate()
}

// desiredOwnedResourceList returns with the list of all child KubeObjects
// belonging to the parent Interface "for object"
func (f *deployFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}

	//fn.Logf("object: %s %s %s %s\n", o.GetAPIVersion(), o.GetKind(), o.GetName(), o.GetNamespace())
	return fn.KubeObjects{}, nil
}

func (f *deployFn) updateResource(deployObj *fn.KubeObject, objs fn.KubeObjects) (fn.KubeObjects, error) {
	if deployObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}

	ipClaimObjs := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPClaimGroupVersionKind))
	amfs, err := getAmfAddresses(ipClaimObjs)
	if err != nil {
		return nil, err
	}

	itfceObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.InterfaceGroupVersionKind))
	if len(itfceObjs) == 0 {
		// if no interface object present we can return now
		return fn.KubeObjects{deployObj}, nil
	}

	resources := fn.KubeObjects{}

	// update the deploy fn with the annotations
	deployKoE, err := ko.NewFromKubeObject[appsv1.Deployment](deployObj)
	if err != nil {
		return nil, err
	}

	deploy, err := deployKoE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	nadString, err := getNetworkAttachmentDefinitionObjects(deploy.Name, itfceObjs)
	if err != nil {
		return nil, err
	}
	if err := deployObj.SetAnnotation(NetworksAnnotation, nadString); err != nil {
		return nil, err	
	}
	resources = append(resources, &deployKoE.KubeObject)

	// add configmap with the additional information
	configuration, err := getConfiguration(itfceObjs, amfs)
	if err != nil {
		return nil, err
	}
	//fn.Logf("config: %s\n", configuration)

	for _, volume := range deploy.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil {
			cmKo, err := buildConfigMapKubeObject(metav1.ObjectMeta{
				Name:      deploy.Spec.Template.Spec.Volumes[0].ConfigMap.Name,
				Namespace: deploy.Namespace,
				Labels:    getLabels(),
			}, "gnb-config.yaml", configuration)
			if err != nil {
				return nil, err
			}
			resources = append(resources, cmKo)
		}
	}
	// add service with the additional information
	serviceKo, err := buildServiceKubeObject(metav1.ObjectMeta{
		Name:      "gnb-service",
		Namespace: deploy.Namespace,
		Labels:    getLabels(),
	}, corev1.ServiceSpec{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{
				Name:       "gnb-ue",
				Protocol:   corev1.ProtocolUDP,
				Port:       4997,
				TargetPort: intstr.IntOrString{IntVal: 4097},
			},
		},
		Selector: getSelectorLabels(),
	})
	if err != nil {
		return nil, err
	}
	resources = append(resources, serviceKo)

	return resources, nil

}

func getAmfAddresses(ipClaimObjs fn.KubeObjects) ([]string, error) {
	amfAddresses := []string{}
	for _, o := range ipClaimObjs {
		ipClaimKOE, err := ko.NewFromKubeObject[ipamv1alpha1.IPClaim](o)
		if err != nil {
			return nil, err
		}

		ipClaim, err := ipClaimKOE.GetGoStruct()
		if err != nil {
			return nil, err
		}

		if ipClaim.Status.Prefix != nil {
			pi, err := iputil.New(*ipClaim.Status.Prefix)
			if err != nil {
				return nil, err
			}
			amfAddresses = append(amfAddresses, pi.Addr().String())
		}
	}
	return amfAddresses, nil
}

func getConfiguration(itfceObjs fn.KubeObjects, amfs []string) (string, error) {
	var templateValues configurationTemplateValues
	templateValues.AMF = amfs
	for _, o := range itfceObjs {
		itfcKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
		if err != nil {
			return "", err
		}

		itfce, err := itfcKOE.GetGoStruct()
		if err != nil {
			return "", err
		}
		switch itfce.Name {
		case "n2", "n3":
			ipAddr, err := getIPAddress(itfce)
			if err != nil {
				return "", err
			}
			if itfce.Name == "n2" {
				templateValues.N2 = ipAddr
			} else {
				templateValues.N3 = ipAddr
			}
		}
	}

	if templateValues.N2 == "" || templateValues.N3 == "" {
		return "", fmt.Errorf("cannot render config, expecting n2, got %s and n3, got %s", templateValues.N2, templateValues.N3)
	}

	return renderConfigurationTemplate(templateValues)
}

func getIPAddress(itfce *nephioreqv1alpha1.Interface) (string, error) {
	if len(itfce.Status.IPClaimStatus) == 0 {
		return "", fmt.Errorf("no ip status provided in interface: %s", itfce.Name)
	}
	// we takes 1 address right now, as this is related to ueransim config
	// TBD how to deal with ipv4 and ipv6, etc
	if itfce.Status.IPClaimStatus[0].Prefix == nil {
		return "", fmt.Errorf("no ip address provided in interface: %s", itfce.Name)
	}
	pi, err := iputil.New(*itfce.Status.IPClaimStatus[0].Prefix)
	if err != nil {
		return "", err
	}
	return pi.Addr().String(), nil
}

func getNetworkAttachmentDefinitionObjects(prefix string, itfceObjs fn.KubeObjects) (string, error) {
	var networksJson []string

	for _, o := range itfceObjs {
		itfcKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
		if err != nil {
			return "", err
		}

		itfce, err := itfcKOE.GetGoStruct()
		if err != nil {
			return "", err
		}

		for _, ipStatus := range itfce.Status.IPClaimStatus {
			if ipStatus.Gateway != nil && ipStatus.Prefix != nil {
				networksJson = append(networksJson, fmt.Sprintf(` {
  "name": %q,
  "interface": %q,
  "ips": [%q],
  "gateways": [%q]
}`,
					createNetworkAttachmentDefintiionName(prefix, itfce.Name),
					itfce.Name,
					*ipStatus.Prefix,
					*ipStatus.Gateway))
			}
		}
	}
	return "[\n" + strings.Join(networksJson, ",\n") + "\n]", nil
}

func createNetworkAttachmentDefintiionName(prefix, suffix string) string {
	return fmt.Sprintf("%s-%s", prefix, suffix)
}

func buildConfigMapKubeObject(meta metav1.ObjectMeta, key, value string) (*fn.KubeObject, error) {
	o := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(corev1.ConfigMap{}).Name(),
		},
		ObjectMeta: meta,
		Data: map[string]string{
			key: value,
		},
	}
	return fn.NewFromTypedObject(o)
}

func buildServiceKubeObject(meta metav1.ObjectMeta, spec corev1.ServiceSpec) (*fn.KubeObject, error) {
	o := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(corev1.Service{}).Name(),
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
	return fn.NewFromTypedObject(o)
}

func getSelectorLabels() map[string]string {
	return map[string]string{
		"app":       "ueransim",
		"component": "gnb",
	}
}

func getLabels() map[string]string {
	l := map[string]string{
		"app.kubernetes.io/version": "v3.2.6",
	}
	for k, v := range getSelectorLabels() {
		l[k] = v
	}
	return l
}
