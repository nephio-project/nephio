WatchCallbackFn
# Conditional kpt fn sdk

## Introduction

The `condkptsdk` is an sdk that aims to simplify building functions/controllers that are part of a `Condition choreography` also called a `Condition` dance. In the "`Condition` choreography/dance" a set of independent actors together reach a certain outcome. 
- The actors are functions/controllers that act on a kpt package.
- A role is performed by a particular instance of a function/controller that act on a particular KRM resource (what we call the `for` resource).
- The conditional dance is a staged execution of the various actors/roles. (implemented using the `kpt` pipeline).
- The outcome is a KRM resources/set of KRM resource specialized within a kpt package.

The `condkptsdk` performs 3 main tasks:
- filtering the resources a particular function/controller acts upon (implemented through for/own/watch filter attributes in the sdk)
- lifecycle (CRUD) operation on behalf of the function/controllers
- readiness gates on behalf of the functions/controllers

The `condkptsdk` is implemented using [golang](https://go.dev) for now and is implemented on top of the [kpt fn sdk](https://github.com/GoogleContainerTools/kpt-functions-sdk)

## Design

Given the `condkptsdk` has a lot of similarities with the controller runtime, the design of the `condkptsdk` is centered around the following resource types (also called filters):
- `for`: the KRM resource the function/controller acts on
- `Owns`: the child KRM resources derived from the `for` resource also called children 
- `Watch`: additional resources the function/controller uses for its execution

Each of these resource types act as a filter within the SDK, such that the function/controller does not have to be bothered with all the other resources in the kpt package.

A resource filter is identified through the Group/Version/Kind definitions of the KRM model expressed using `apiVersion` and `Kind`

example

```golang
For: corev1.ObjectReference{
    APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
    Kind:       nephioreqv1alpha1.InterfaceKind,
}
```

### For KRM resource

Each function or controller has to implement a single `for` KRM resource. A `kpt` package can have multiple KRM resource instances matching the `for` filter. We call each instance of the KRM resource matching the `for` filter a `forKRMInstance`.

example `for` resource filter

```golang
For: corev1.ObjectReference{
    APIVersion: "example.com/v1alpha1",
    Kind:       "A",
}
```

Lets assume the kpt package contains

```yaml
apiVersion: example.com/v1alpha1
kind: A
metadata:
  name: a1

apiVersion: example.com/v1alpha1
kind: A
metadata:
  name: a2

apiVersion: example.com/v1alpha1
kind: A
metadata:
  name: a3
```

This results in 3 `for` KRMInstances:

```yaml
example.com/v1alpha1.A.a1
example.com/v1alpha1.A.a2
example.com/v1alpha1.A.a3
```

### Owns KRM resource

The `Owns` resource filter identifies which KRM resources are children of the `for` resource instance. You could also say these are created or lifecycled as a result of the parent resource (the `for` resource in this case)
The sdk defines different types of own resources:
- childRemote: 
    - the current/parent fn/controller defines the spec attributes of the child resource, but another child function/controller takes care of the actuation that are related to this KRM resource. Like updating the status and are deriving other child resources acting as a parent.
    - A remote function will act upon this KRM through a `for` filter and will update the status. 
    - The deletion is taken care of by the remote resource by acting on the delete annotation set by the sdk.
    - An example use case is e.g. the interface-fn that needs an IP. The interface-fn is the parent that creates an IPAllocation on which a downstream function/controller acts and fills out the IP Allocation
- childRemoteCondition: 
    - the current/parent fn/controller defines the KRM header attributes of the child resource, but another function/controller takes care of the spec and or status that are related to this KRM resource. 
    - A remote function will act upon this KRM through a `for` filter and will update the status
    - The deletion is take care of by the remote resource by acting on the delete annotation set by the sdk
    - The typical usage pattern is when the parent has insufficient information to define the full spec. E.g. a NAD needs an IP and VLAN for it to be specified fully. So rather than building a half baked CRD the system generates a condition for the child NAD fn/controller to act upon and it will create the Spec within the child function/controller.
- childLocal: 
    - the fn/controller defines the spec locally within the fn/controller. Its condition is always true as the actor is the fn itslef
- childInitial:
    - the package can contain already owned reference in the resource to indicate its dependency. As such the SDK takes this into account when updating conditions. 

The fn/controller implementation is triggered using the `PopulateOwnResourcesFn` callback.
Each of the `own` instances are lifecycled by the sdk within the context of the `for` instance. As such if we have 3 `for` instance the sdk calls the `PopulateOwnResourcesFn` callback 3 times

### Watch KRM resource

The `Watch` resource filter identifies KRM resources that the function/controller uses as additional information to define its outcome
There are 2 types of `watch` resource filters:
- global one:
    - they are triggered through the `WatchCallbackFn` and can influence the readiness gate.
    - e.g. WorkloadCluster
- instance based:
    - they are relevant within the context of a `for` instance
    - e.g. IP, VLAN within the NAD function/controller

###  WatchCallbackFn

The `WatchCallbackFn` provides the `watched KubeObject instance` to the fn/controller. The function/controller uses the KubeObject for contextual information as extra metadata when the `PopulateOwnResourcesFn` or `GenerateResourceFn` are called. On top when processing the callback the fn can return an error, which is used by the sdk to determine readiness within the sdk

The `WatchCallbackFn` is called for each global watches resource.

signature of the watchCallbackFn:

```golang
type WatchCallbackFn func(*fn.KubeObject) error
```

If the fn/controller is dependent on a global resource the fn/controller MUST implement the `WatchCallbackFn`.

### PopulateOwnResourcesFn

The `PopulateOwnResourcesFn` provides the `for KubeObject instance` to the fn/controller. The function/controller uses the `for KubeObject` + optionally the contextual information provided through the `WatchCallbackFn` and returns a list of child KRM resources as `KubeObject`. These child resource are defined by the fn/controller based on the content of the `for` KRM resource instance + the metadata.

The `PopulateOwnResourcesFn` is called for each `for` KRM resource instance.

The sdk will handle from here on the lifecycle of the child resources based on the `OwnType`. The sdk uses the `ownerReference` implemented through an annotation to identify child resource instances belonging to a for instance 

signature of the populateOwnResourcesFn:

```golang
type PopulateOwnResourcesFn func(*fn.KubeObject) (fn.KubeObjects, error)
```

Any fn/controller that has own resources MUST implement the `PopulateOwnResourcesFn`.

### UpdateResourceFn

The `UpdateResourceFn` provides:
- the `for` object as a first parameter (f the object does not exist a nil pointer is provided)
- the `watch` and `own` instance resource are provided as a list

The `UpdateResourceFn` function updates the status based on the resource information it is presented. E.g. NAD uses IP and VLAN and interface KRM to generate the KRM.

signature of the UpdateResourceFn:

```golang
type UpdateResourceFn func(*fn.KubeObject, fn.KubeObjects) (*fn.KubeObject, error)
```

Any fn/controller MUST implement the `UpdateResourceFn`.

### sdk phases

The SDK operates in phases when being executed within a fn/controller
- firstly the sdk builds up an inventory based on the filters
- Secondly the global watch callbacks are called. The fn/controller implementing these callback use the data for 2 things:
    1. uses attributes of the KRM for further processing later on
    2. provide feedback on readiness through the error
- Afterwards if the readiness gate is true the `PopulateOwnResourcesFn` is called if defined by the fn/controller.
    - the resources returned are used as 
        - child resource within the `for` instance
        - the sdk performs a diff and brings the actual state in line with the desired state
- In the final phase when readiness is determined, the sdk executes the `GenerateResourceFn`. The result is the final step and the data is added/updated in the resourceList of the kpt package

Each function/controller has to implement `UpdateResourceFn`. Only the functions/controller having own resource have to implement `PopulateOwnResourcesFn`.

### pipeline stages

Right now the kpt pipeline is used to execute the conditional dance

### example

#### example with owns

```golang
type itfceFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
}

func Run(rl *fn.ResourceList) (bool, error) {
	myFn := itfceFn{}
	var err error
	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
				Kind:       nephioreqv1alpha1.InterfaceKind,
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: nadv1.SchemeGroupVersion.Identifier(),
					Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
				}: condkptsdk.ChildRemoteCondition,
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPAllocationKind,
				}: condkptsdk.ChildRemote,
				{
					APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
					Kind:       vlanv1alpha1.VLANAllocationKind,
				}: condkptsdk.ChildRemote,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: myFn.WorkloadClusterCallbackFn,
			},
			PopulateOwnResourcesFn: myFn.desiredOwnedResourceList,
			UpdateResourceFn:       myFn.updateItfceResource,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return myFn.sdk.Run()
}

// WorkloadClusterCallbackFn provides a callback for the workload cluster
// resources in the resourceList
func (f *itfceFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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
func (f *itfceFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}
	// resources contain the list of child resources
	// belonging to the parent object
	resources := fn.KubeObjects{}

	itfceKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
	if err != nil {
		return nil, err
	}

	itfce, err := itfceKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	// Nothing to be done in case the interface is attached to
	// the default pod network since this is all handled in the
	// k8s cluster via the CNI.
	if itfce.Spec.NetworkInstance.Name == defaultPODNetwork {
		return fn.KubeObjects{}, nil
	}

	// meta is the generic object meta attached to all derived child objects
	meta := metav1.ObjectMeta{
		Name: o.GetName(),
	}
	// When the CNIType is not set this is a loopback interface
	if itfce.Spec.CNIType != "" {
		if !f.IsCNITypePresent(itfce.Spec.CNIType) {
			return nil, fmt.Errorf("cniType not supported in workload cluster; workload cluster CNI(s): %v, interface cniType requested: %s", f.workloadCluster.Spec.CNIs, itfce.Spec.CNIType)
		}
		// add IP allocation of type network
		o, err := f.getIPAllocation(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindNetwork)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)

		if itfce.Spec.AttachmentType == nephioreqv1alpha1.AttachmentTypeVLAN {
			// add VLAN allocation
			o, err := f.getVLANAllocation(meta)
			if err != nil {
				return nil, err
			}
			resources = append(resources, o)
		}

		// allocate nad
		o, err = f.getNAD(meta)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)
	} else {
		// add IP allocation of type loopback
		o, err := f.getIPAllocation(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindLoopback)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)
	}
	return resources, nil
}

func (f *itfceFn) updateItfceResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}
	itfceKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](forObj)
	if err != nil {
		return nil, err
	}
	itfce, err := itfceKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	ipallocs := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPAllocationGroupVersionKind))
	for _, ipalloc := range ipallocs {
		if ipalloc.GetName() == forObj.GetName() {
			alloc, err := ko.NewFromKubeObject[ipamv1alpha1.IPAllocation](ipalloc)
			if err != nil {
				return nil, err
			}
			allocGoStruct, err := alloc.GetGoStruct()
			if err != nil {
				return nil, err
			}
			itfce.Status.IPAllocationStatus = &allocGoStruct.Status
		}
	}
	vlanallocs := objs.Where(fn.IsGroupVersionKind(vlanv1alpha1.VLANAllocationGroupVersionKind))
	for _, vlanalloc := range vlanallocs {
		if vlanalloc.GetName() == forObj.GetName() {
			alloc, err := vlanlibv1alpha1.NewFromKubeObject(vlanalloc)
			if err != nil {
				return nil, err
			}
			allocGoStruct, err := alloc.GetGoStruct()
			if err != nil {
				return nil, err
			}
			itfce.Status.VLANAllocationStatus = &allocGoStruct.Status
		}
	}
	// set the status
	err = itfceKOE.SetStatus(itfce)
	return &itfceKOE.KubeObject, err
}

```

### example without owns

```golang
type nadFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
}

func Run(rl *fn.ResourceList) (bool, error) {
	myFn := nadFn{}
	var err error
	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nadv1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: myFn.WorkloadClusterCallbackFn,
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPAllocationKind,
				}: nil,
				{
					APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
					Kind:       vlanv1alpha1.VLANAllocationKind,
				}: nil,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: nil,
			},
			PopulateOwnResourcesFn: nil,
			UpdateResourceFn:       myFn.updateResourceFn,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return myFn.sdk.Run()
}

// WorkloadClusterCallbackFn provides a callback for the workload cluster
// resources in the resourceList
func (f *nadFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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

func (f *nadFn) updateResourceFn(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}
	ipAllocationObjs := objs.Where(fn.IsGroupVersionKind(schema.GroupVersionKind(ipamv1alpha1.IPAllocationGroupVersionKind)))
	vlanAllocationObjs := objs.Where(fn.IsGroupVersionKind(schema.GroupVersionKind(vlanv1alpha1.VLANAllocationGroupVersionKind)))
	interfaceObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.InterfaceGroupVersionKind))

	// verify all needed objects exist
	if interfaceObjs.Len() == 0 {
		return nil, fmt.Errorf("expected %s object to generate the nad", nephioreqv1alpha1.InterfaceKind)
	}
	if ipAllocationObjs.Len() == 0 && vlanAllocationObjs.Len() == 0 {
		return nil, fmt.Errorf("expected one of %s or %s objects to generate the nad", ipamv1alpha1.IPAllocationKind, vlanv1alpha1.VLANAllocationKind)
	}

	// generate an empty nad struct
	nad, err := nadlibv1.NewFromGoStruct(&nadv1.NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nadv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
		},
		ObjectMeta: metav1.ObjectMeta{Name: objs[0].GetName()},
	})
	if err != nil {
		return nil, err
	}

	if ipAllocationObjs.Len() == 0 && vlanAllocationObjs.Len() != 0 {
		nad.CniSpecType = nadlibv1.VlanAllocOnly
	}
	if nad.CniSpecType != nadlibv1.VlanAllocOnly {
		for _, itfce := range interfaceObjs {
			i, err := interfacelibv1alpha1.NewFromKubeObject(itfce)
			if err != nil {
				return nil, err
			}
			interfaceGoStruct, err := i.GetGoStruct()
			if err != nil {
				return nil, err
			}

			if !f.IsCNITypePresent(interfaceGoStruct.Spec.CNIType) {
				return nil, fmt.Errorf("cniType not supported in workload cluster; workload cluster CNI(s): %v, interface cniType requested: %s", f.workloadCluster.Spec.CNIs, interfaceGoStruct.Spec.CNIType)
			}

			if err := nad.SetCNIType(string(interfaceGoStruct.Spec.CNIType)); err != nil {
				return nil, err
			}
			err = nad.SetNadMaster(*f.workloadCluster.Spec.MasterInterface) // since we validated the workload cluster before it is safe to do this
			if err != nil {
				return nil, err
			}
		}

		for _, ipAllocation := range ipAllocationObjs {
			alloc, err := ipalloclibv1alpha1.NewFromKubeObject(ipAllocation)
			if err != nil {
				return nil, err
			}
			allocGoStruct, err := alloc.GetGoStruct()
			if err != nil {
				return nil, err
			}
			address := ""
			gateway := ""
			if allocGoStruct.Status.Prefix != nil {
				address = *allocGoStruct.Status.Prefix
			}
			if allocGoStruct.Status.Gateway != nil {
				gateway = *allocGoStruct.Status.Gateway
			}
			err = nad.SetIpamAddress([]nadlibv1.Addresses{{
				Address: address,
				Gateway: gateway,
			}})
			if err != nil {
				return nil, err
			}
		}
	}

	for _, vlanAllocation := range vlanAllocationObjs {
		vlanID, _, _ := vlanAllocation.NestedInt([]string{"status", "vlanID"}...)
		err = nad.SetVlan(vlanID)
		if err != nil {
			return nil, err
		}
	}

	return &nad.K.KubeObject, nil
}

func (f *nadFn) IsCNITypePresent(itfceCNIType nephioreqv1alpha1.CNIType) bool {
	for _, cni := range f.workloadCluster.Spec.CNIs {
		if nephioreqv1alpha1.CNIType(cni) == itfceCNIType {
			return true
		}
	}
	return false
}
```

## see also

- [golang](https://go.dev)
- [configuration as data](https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/06-config-as-data.md)
- [cad concepts](https://kpt.dev/book/02-concepts/)
- [kpt](https://kpt.dev/book/02-concepts/03-functions)
- [porch](https://kpt.dev/guides/porch-user-guide)
- [kpt fn sdk](https://github.com/GoogleContainerTools/kpt-functions-sdk)
