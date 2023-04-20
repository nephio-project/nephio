package fnruntime

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

type DownstreamInventory interface {
	AddCondition(name string, resource *corev1.ObjectReference, c *kptv1.Condition)
	AddResource(name string, resource *corev1.ObjectReference, o *fn.KubeObject)
	AddForCondition(name string, c *kptv1.Condition)
	AddForResource(name string, o *fn.KubeObject)
	GetForConditionStatus(name string) bool
	GetReadinessStatus() map[string]*ReadyCtx
}

func NewDownstreamInventory() DownstreamInventory {
	return &downstreamInventory{
		namedResources: map[string]*downstreamResources{},
	}
}

type downstreamInventory struct {
	namedResources map[string]*downstreamResources
}

type downstreamResources struct {
	resources    map[corev1.ObjectReference]*downstreamInventoryCtx
	forObj       *fn.KubeObject
	forCondition kptv1.Condition
}

type downstreamInventoryCtx struct {
	condition *kptv1.Condition
	obj       *fn.KubeObject
}

func (r *downstreamInventory) AddCondition(name string, resource *corev1.ObjectReference, c *kptv1.Condition) {
	if _, ok := r.namedResources[name]; !ok {
		r.namedResources[name] = &downstreamResources{
			resources: map[corev1.ObjectReference]*downstreamInventoryCtx{},
		}
	}
	if r.namedResources[name].resources[*resource] == nil {
		r.namedResources[name].resources[*resource] = &downstreamInventoryCtx{}
	}
	r.namedResources[name].resources[*resource].condition = c
}

func (r *downstreamInventory) AddResource(name string, resource *corev1.ObjectReference, o *fn.KubeObject) {
	if _, ok := r.namedResources[name]; !ok {
		r.namedResources[name] = &downstreamResources{
			resources: map[corev1.ObjectReference]*downstreamInventoryCtx{},
		}
	}
	if r.namedResources[name].resources[*resource] == nil {
		r.namedResources[name].resources[*resource] = &downstreamInventoryCtx{}
	}
	r.namedResources[name].resources[*resource].obj = o
}

func (r *downstreamInventory) AddForCondition(name string, c *kptv1.Condition) {
	if _, ok := r.namedResources[name]; !ok {
		r.namedResources[name] = &downstreamResources{
			resources: map[corev1.ObjectReference]*downstreamInventoryCtx{},
		}
	}
	r.namedResources[name].forCondition = *c
}

func (r *downstreamInventory) AddForResource(name string, o *fn.KubeObject) {
	if _, ok := r.namedResources[name]; !ok {
		r.namedResources[name] = &downstreamResources{
			resources: map[corev1.ObjectReference]*downstreamInventoryCtx{},
		}
	}
	r.namedResources[name].forObj = o
}

func (r *downstreamInventory) GetForConditionStatus(name string) bool {
	c, ok := r.namedResources[name]
	if !ok {
		// this should not happen
		return false
	}
	return c.forCondition.Status == kptv1.ConditionFalse
}

func (r *downstreamInventory) GetReadinessStatus() map[string]*ReadyCtx {
	readyMap := map[string]*ReadyCtx{}
	for name, ref := range r.namedResources {
		readyMap[name] = &ReadyCtx{
			ForCondition: ref.forCondition,
			Objs:         map[corev1.ObjectReference]fn.KubeObject{},
		}
		if ref.forObj != nil {
			readyMap[name].ForObj = ref.forObj
		}

		// if no child resources exist we return empty
		if len(ref.resources) == 0 {
			readyMap[name].Ready = false
			return readyMap
		}

		readyMap[name].Ready = true
		for objRef, invCtx := range ref.resources {
			// not all resources have a condition set e.g. interface
			//
			if invCtx.condition != nil && invCtx.condition.Status == kptv1.ConditionFalse {
				readyMap[name].Ready = false
				break
			}
			// not all resources have an associated object
			if invCtx.obj != nil {
				readyMap[name].Objs[objRef] = *invCtx.obj
			}
		}
	}
	return readyMap
}

type ReadyCtx struct {
	Ready        bool
	Objs         map[corev1.ObjectReference]fn.KubeObject
	ForObj       *fn.KubeObject
	ForCondition kptv1.Condition
}
