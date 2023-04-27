package v1alpha1

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/iputil"
)

var (
	prefixKind          = []string{"spec", "kind"}
	networkInstanceName = []string{"spec", "networkInstance", "name"}
	addressFamily       = []string{"spec", "addressFamily"}
	prefix              = []string{"spec", "prefix"}
	prefixLength        = []string{"spec", "prefixLength"}
	index               = []string{"spec", "index"}
	selectorLabels      = []string{"spec", "selector", "matchLabels"}
	labels              = []string{"spec", "labels"}
	createPrefix        = []string{"spec", "createPrefix"}
	allocatedPrefix     = []string{"status", "prefix"}
	allocatedGateway    = []string{"status", "gateway"}
)

type IPAllocation struct {
	kubeobject.KubeObjectExt[*ipamv1alpha1.IPAllocation]
}

// NewFromKubeObject returns a KubeObjectExt struct
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) (*IPAllocation, error) {
	r, err := kubeobject.NewFromKubeObject[*ipamv1alpha1.IPAllocation](o)
	if err != nil {
		return nil, err
	}
	return &IPAllocation{*r}, nil
}

// NewFromYaml returns a KubeObjectExt struct
// It expects raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (*IPAllocation, error) {
	r, err := kubeobject.NewFromYaml[*ipamv1alpha1.IPAllocation](b)
	if err != nil {
		return nil, err
	}
	return &IPAllocation{*r}, nil
}

// NewFromGoStruct returns a KubeObjectExt struct
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *ipamv1alpha1.IPAllocation) (*IPAllocation, error) {
	r, err := kubeobject.NewFromGoStruct[*ipamv1alpha1.IPAllocation](x)
	if err != nil {
		return nil, err
	}
	return &IPAllocation{*r}, nil
}

func (r *IPAllocation) SetSpec(spec *ipamv1alpha1.IPAllocationSpec) error {
	if spec == nil {
		return nil
	}
	if spec.PrefixKind != "" {
		if err := r.SetPrefixKind(spec.PrefixKind); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("prefixKind is required")
	}
	if spec.NetworkInstance != nil {
		if err := r.SetNetworkInstanceName(string(spec.NetworkInstance.Name)); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("networkInstance is required")
	}

	if spec.AddressFamily != "" {
		if err := r.SetAddressFamily(spec.AddressFamily); err != nil {
			return err
		}
	} else {
		if _, err := r.DeleteAddressFamily(); err != nil {
			return err
		}
	}
	if spec.Prefix != "" {
		if err := r.SetPrefix(spec.Prefix); err != nil {
			return err
		}
	} else {
		if _, err := r.DeletePrefix(); err != nil {
			return err
		}
	}
	if spec.PrefixLength != 0 {
		if err := r.SetPrefixLength(spec.PrefixLength); err != nil {
			return err
		}
	} else {
		if _, err := r.DeletePrefixLength(); err != nil {
			return err
		}
	}
	if spec.Index != 0 {
		if err := r.SetIndex(spec.Index); err != nil {
			return err
		}
	} else {
		if _, err := r.DeleteIndex(); err != nil {
			return err
		}
	}
	if spec.Selector != nil && spec.Selector.MatchLabels != nil {
		if err := r.SetSelectorLabels(spec.Selector.MatchLabels); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("selector matchlabels is required")
	}
	if spec.Labels != nil {
		if err := r.SetSpecLabels(spec.Labels); err != nil {
			return err
		}
	} else {
		if _, err := r.DeleteSpecLabels(); err != nil {
			return err
		}
	}
	if spec.CreatePrefix {
		if err := r.SetCreatePrefix(spec.CreatePrefix); err != nil {
			return err
		}
	} else {
		if _, err := r.DeleteCreatePrefix(); err != nil {
			return err
		}
	}

	return nil
}

func (r *IPAllocation) GetNestedString(fields ...string) string {
	s, ok, err := r.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *IPAllocation) GetNestedInt(fields ...string) int {
	s, ok, err := r.NestedInt(fields...)
	if err != nil {
		return 0
	}
	if !ok {
		return 0
	}
	return s
}

func (r *IPAllocation) GetNestedBool(fields ...string) bool {
	s, ok, err := r.NestedBool(fields...)
	if err != nil {
		return false
	}
	if !ok {
		return false
	}
	return s
}

func (r *IPAllocation) GetNestedStringMap(fields ...string) map[string]string {
	s, ok, err := r.NestedStringMap(fields...)
	if err != nil {
		return map[string]string{}
	}
	if !ok {
		return map[string]string{}
	}
	return s
}

// GetPrefixKind returns the prefixKind from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *IPAllocation) GetPrefixKind() ipamv1alpha1.PrefixKind {
	return ipamv1alpha1.PrefixKind(r.GetNestedString(prefixKind...))
}

// GetNetworkInstanceName returns the name of the networkInstance from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *IPAllocation) GetNetworkInstanceName() string {
	return r.GetNestedString(networkInstanceName...)
}

// GetNetworkInstanceName returns the name of the networkInstance from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *IPAllocation) GetAddressFamily() iputil.AddressFamily {
	return iputil.AddressFamily(r.GetNestedString(addressFamily...))
}

// GetPrefix returns the prefix from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *IPAllocation) GetPrefix() string {
	return r.GetNestedString(prefix...)
}

// GetPrefixLength returns the prefixlength from the spec
// if an error occurs or the attribute is not present 0 is returned
func (r *IPAllocation) GetPrefixLength() uint8 {
	return uint8(r.GetNestedInt(prefixLength...))
}

// GetIndex returns the index from the spec
// if an error occurs or the attribute is not present 0 is returned
func (r *IPAllocation) GetIndex() uint32 {
	return uint32(r.GetNestedInt(index...))
}

// GetSelectorLabels returns the selector Labels from the spec
// if an error occurs or the attribute is not present an empty map[string]string is returned
func (r *IPAllocation) GetSelectorLabels() map[string]string {
	return r.GetNestedStringMap(selectorLabels...)
}

// GetSpecLabels returns the labelsfrom the spec
// if an error occurs or the attribute is not present an empty map[string]string is returned
func (r *IPAllocation) GetSpecLabels() map[string]string {
	return r.GetNestedStringMap(labels...)
}

// GetCreatePrefix returns the create prefix from the spec
// if an error occurs or the attribute is not present false is returned
func (r *IPAllocation) GetCreatePrefix() bool {
	return r.GetNestedBool(createPrefix...)
}

// GetAllocatedPrefix returns the prefix from the status
// if an error occurs or the attribute is not present an empty string is returned
func (r *IPAllocation) GetAllocatedPrefix() string {
	return r.GetNestedString(allocatedPrefix...)
}

// GetAllocatedGateway returns the gateway from the status
// if an error occurs or the attribute is not present an empty string is returned
func (r *IPAllocation) GetAllocatedGateway() string {
	return r.GetNestedString(allocatedGateway...)
}

// SetPrefixKind sets the prefixKind in the spec
func (r *IPAllocation) SetPrefixKind(s ipamv1alpha1.PrefixKind) error {
	return r.SetNestedString(string(s), prefixKind...)
}

// SetNetworkInstanceName sets the name of the networkInstance in the spec
func (r *IPAllocation) SetNetworkInstanceName(s string) error {
	return r.SetNestedString(string(s), networkInstanceName...)
}

// SetAddressFamily sets the address family in the spec
func (r *IPAllocation) SetAddressFamily(s iputil.AddressFamily) error {
	return r.SetNestedString(string(s), addressFamily...)
}

// SetPrefix sets the prefix in the spec
func (r *IPAllocation) SetPrefix(s string) error {
	if _, err := iputil.New(s); err != nil {
		return err
	}
	return r.SetNestedString(string(s), addressFamily...)
}

// SetPrefixLength sets the prefix length in the spec
func (r *IPAllocation) SetPrefixLength(s uint8) error {
	return r.SetNestedInt(int(s), prefixLength...)
}

// SetIndex sets the index in the spec
func (r *IPAllocation) SetIndex(s uint32) error {
	return r.SetNestedInt(int(s), index...)
}

// SetSelectorLabels sets the selector matchLabels in the spec
func (r *IPAllocation) SetSelectorLabels(s map[string]string) error {
	return r.SetNestedStringMap(s, selectorLabels...)
}

// SetSpecLabels sets the labels in the spec
func (r *IPAllocation) SetSpecLabels(s map[string]string) error {
	return r.SetNestedStringMap(s, labels...)
}

// SetCreatePrefix sets the create prefix in the spec
func (r *IPAllocation) SetCreatePrefix(s bool) error {
	return r.SetNestedBool(s, createPrefix...)
}

// SetAllocatedPrefix sets the allocated prefix in the status
func (r *IPAllocation) SetAllocatedPrefix(s string) error {
	if _, err := iputil.New(s); err != nil {
		return err
	}
	return r.SetNestedString(s, createPrefix...)
}

// SetAllocatedGateway sets the allocated gateway in the status
func (r *IPAllocation) SetAllocatedGateway(s string) error {
	if _, err := iputil.New(s); err != nil {
		return err
	}
	return r.SetNestedString(s, createPrefix...)
}

// DeleteAddressFamily deletes the address family from the spec
func (r *IPAllocation) DeleteAddressFamily() (bool, error) {
	return r.RemoveNestedField(addressFamily...)
}

// DeletePrefix deletes the prefix from the spec
func (r *IPAllocation) DeletePrefix() (bool, error) {
	return r.RemoveNestedField(prefix...)
}

// DeletePrefixLength deletes the prefix length from the spec
func (r *IPAllocation) DeletePrefixLength() (bool, error) {
	return r.RemoveNestedField(prefixLength...)
}

// DeleteIndex deletes the index from the spec
func (r *IPAllocation) DeleteIndex() (bool, error) {
	return r.RemoveNestedField(index...)
}

// DeleteSpecLabels deletes the labels from the spec
func (r *IPAllocation) DeleteSpecLabels() (bool, error) {
	return r.RemoveNestedField(labels...)
}

// DeleteCreatePrefix deletes the create prefix from the spec
func (r *IPAllocation) DeleteCreatePrefix() (bool, error) {
	return r.RemoveNestedField(createPrefix...)
}

// DeleteAllocatedPrefix deletes the allocated prefix from the status
func (r *IPAllocation) DeleteAllocatedPrefix() (bool, error) {
	return r.RemoveNestedField(allocatedPrefix...)
}

// DeleteAllocatedGateway deletes the allocated gateway from the status
func (r *IPAllocation) DeleteAllocatedGateway() (bool, error) {
	return r.RemoveNestedField(allocatedGateway...)
}
