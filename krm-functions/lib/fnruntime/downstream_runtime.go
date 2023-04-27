package fnruntime

import (
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/lib/kptrl"
	corev1 "k8s.io/api/core/v1"
)

const Wildcard = "wildcard"

type DownstreamRuntimeConfig struct {
	For   DownstreamRuntimeForConfig
	Owns  map[corev1.ObjectReference]struct{}
	Watch map[corev1.ObjectReference]WatchCallbackFn
}

type DownstreamRuntimeForConfig struct {
	ObjectRef  corev1.ObjectReference
	GenerateFn GenerateFn
}

func NewDownstream(rl *fn.ResourceList, c *DownstreamRuntimeConfig) FnRuntime {
	r := &downstreamFnRuntime{
		rl:         kptrl.New(rl),
		inv:        NewDownstreamInventory(),
		cfg:        c,
		owners:     map[string]struct{}{},
		forObjects: map[corev1.ObjectReference]*fn.KubeObject{},
	}
	return r
}

type downstreamFnRuntime struct {
	cfg        *DownstreamRuntimeConfig
	rl         kptrl.ResourceList
	inv        DownstreamInventory
	owners     map[string]struct{}
	forObjects map[corev1.ObjectReference]*fn.KubeObject
}

func (r *downstreamFnRuntime) Run() {
	r.initialize()
	r.update()
}

func (r *downstreamFnRuntime) initialize() {
	// First check if the for resource is wildcard or not;
	// The inventory is populated based on wildcard status
	if r.rl.GetObjects().Len() > 0 {
		// we assume the kpt file is always resource idx 0 in the resourcelist
		o := r.rl.GetObjects()[0]

		kf := kptfilelibv1.NewMutator(o.String())
		var err error
		if _, err = kf.UnMarshal(); err != nil {
			fn.Log("error unmarshal kptfile in initialize")
			r.rl.AddResult(err, o)
		}

		// populate condition inventory
		for _, c := range kf.GetConditions() {
			// based on the ForObj determine if there is work to be done
			if strings.Contains(c.Type, kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef)) {
				if c.Status == kptv1.ConditionFalse {
					if len(r.cfg.Owns) == 0 {
						// this is looking for all resources
						r.inv.AddForCondition(Wildcard, &c)
					} else {
						objRef := kptfilelibv1.GetGVKNFromConditionType(c.Reason)
						r.inv.AddForCondition(objRef.Name, &c)
					}
				}
			}
		}

		// collect all conditions
		for _, c := range kf.GetConditions() {
			objRef := *kptfilelibv1.GetGVKNFromConditionType(c.Type)
			if len(r.cfg.Owns) == 0 {
				// collect all conditions for a wildcard
				r.inv.AddCondition(Wildcard, &objRef, &c)
			} else {
				for ref := range r.cfg.Owns {
					if strings.Contains(c.Type, kptfilelibv1.GetConditionType(&ref)) {
						r.inv.AddCondition(objRef.Name, &objRef, &c)
					}
				}
			}
		}
	}

	// filter the related resources per name in case no wildcard
	// for wildcard we add all resources to the wildcard context
	for _, o := range r.rl.GetObjects() {
		if len(r.cfg.Owns) == 0 {
			if o.GetAPIVersion() == r.cfg.For.ObjectRef.APIVersion && o.GetKind() == r.cfg.For.ObjectRef.Kind {
				r.inv.AddForResource(o.GetName(), o)
			} else {
				r.inv.AddResource(Wildcard, &corev1.ObjectReference{
					APIVersion: o.GetAPIVersion(),
					Kind:       o.GetKind(),
					Name:       o.GetName(),
				}, o)
			}
		} else {
			for objRef := range r.cfg.Owns {
				if o.GetAPIVersion() == objRef.APIVersion && o.GetKind() == objRef.Kind {
					r.inv.AddResource(o.GetName(), &corev1.ObjectReference{
						APIVersion: o.GetAPIVersion(),
						Kind:       o.GetKind(),
						Name:       o.GetName(),
					}, o)
				}
				if o.GetAPIVersion() == r.cfg.For.ObjectRef.APIVersion && o.GetKind() == r.cfg.For.ObjectRef.Kind {
					r.inv.AddForResource(o.GetName(), o)
				}
			}
		}
	}
}

func (r *downstreamFnRuntime) update() {
	kf := kptfilelibv1.NewMutator(r.rl.GetObjects()[0].String())
	var err error
	if _, err = kf.UnMarshal(); err != nil {
		fn.Log("error unmarshal kptfile")
		r.rl.AddResult(err, r.rl.GetObjects()[0])
	}

	for name, readyCtx := range r.inv.GetReadinessStatus() {
		fn.Logf("name: %s condition: %s ready: %t\n", name, readyCtx.ForCondition.Type, readyCtx.Ready)
		if readyCtx.Ready {
			// generate the obj irrespective of current status
			if r.cfg.For.GenerateFn != nil {
				o, err := r.cfg.For.GenerateFn(readyCtx.Objs)
				if err != nil {
					r.rl.AddResult(err, o)
				} else {
					if readyCtx.ForObj != nil {
						r.rl.SetObject(o)
					} else {
						r.rl.AddObject(o)
					}
					// set status to True
					readyCtx.ForCondition.Status = kptv1.ConditionTrue
					kf.SetConditions(readyCtx.ForCondition)
				}
			}
		} else {
			// if obj exists delete it
			if readyCtx.ForObj != nil {
				delete := readyCtx.ForObj.GetAnnotation(FnRuntimeDelete) == "true"
				r.rl.DeleteObject(readyCtx.ForObj)
				if delete {
					kf.DeleteCondition(readyCtx.ForCondition.Type)
				}
			}
		}
	}
}
