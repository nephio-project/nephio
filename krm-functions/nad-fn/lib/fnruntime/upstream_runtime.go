package fnruntime

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/lib/kptrl"
	corev1 "k8s.io/api/core/v1"
)

type UpstreamRuntimeConfig struct {
	For         UpstreamRuntimeForConfig
	Owns        map[corev1.ObjectReference]UpstreamRuntimeConfigOperation
	Watch       map[corev1.ObjectReference]WatchCallbackFn
	ConditionFn ConditionFn
}

type UpstreamRuntimeForConfig struct {
	ObjectRef  corev1.ObjectReference
	PopulateFn PopulateFn
}

type UpstreamRuntimeConfigOperation string

const (
	UpstreamRuntimeConfigOperationDefault         UpstreamRuntimeConfigOperation = "default"
	UpstreamRuntimeConfigOperationConditionOnly   UpstreamRuntimeConfigOperation = "conditionOnly"
	UpstreamRuntimeConfigOperationConditionGlobal UpstreamRuntimeConfigOperation = "conditionGlobal"
)

func NewUpstream(rl *fn.ResourceList, c *UpstreamRuntimeConfig) FnRuntime {
	r := &upstreamFnRuntime{
		cfg:         c,
		inv:         NewUpstreamInventory(),
		rl:          kptrl.New(rl),
		conditionFn: conditionFnNop,
	}
	if r.cfg.ConditionFn != nil {
		r.conditionFn = r.cfg.ConditionFn
	}

	return r
}

type upstreamFnRuntime struct {
	cfg         *UpstreamRuntimeConfig
	inv         UpstreamInventory
	rl          kptrl.ResourceList
	conditionFn ConditionFn
}

func (r *upstreamFnRuntime) Run() {
	r.initialize()
	r.populate()
	r.update()
}

// initialize updates the inventory based on the interested resources
// kptfile conditions
// own and watch ressources from the config
func (r *upstreamFnRuntime) initialize() {
	for _, o := range r.rl.GetObjects() {
		if o.GetAPIVersion() == kptv1.KptFileGVK().GroupVersion().String() && o.GetKind() == kptv1.KptFileName {
			kf := kptfilelibv1.NewMutator(o.String())
			var err error
			if _, err = kf.UnMarshal(); err != nil {
				fn.Log("error unmarshal kptfile in initialize")
				r.rl.AddResult(err, o)
			}

			// populate condition inventory as existing conditions in the package
			for objRef := range r.cfg.Owns {
				for _, c := range kf.GetConditions() {
					if strings.Contains(c.Type, kptfilelibv1.GetConditionType(&objRef)) &&
						strings.Contains(c.Reason, kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef)) {
						r.inv.AddExistingCondition(kptfilelibv1.GetGVKNFromConditionType(c.Type), &c)
					}
				}
			}
		}

		// populate the inventory with own resources as an exisiting resource in the package
		for objRef := range r.cfg.Owns {
			if o.GetAPIVersion() == objRef.APIVersion && o.GetKind() == objRef.Kind &&
				o.GetAnnotation(FnRuntimeOwner) == kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef) {

				r.inv.AddExistingResource(&corev1.ObjectReference{
					APIVersion: objRef.APIVersion,
					Kind:       objRef.Kind,
					Name:       o.GetName(),
				}, o)
			}
		}

		// callback provides a means to provide the fn information
		// on resources they are interested in and can make decisions on this
		for objRef, watchCallbackFn := range r.cfg.Watch {
			if o.GetAPIVersion() == objRef.APIVersion &&
				o.GetKind() == objRef.Kind {
				// provide watch resource
				if err := watchCallbackFn(o); err != nil {
					r.rl.AddResult(err, o)
				}
			}
		}
	}
}

// populate populates the inventory with resources based on the For resource data
func (r *upstreamFnRuntime) populate() {
	// the condition Fn allows to control the behavior if the populate needs to be executed
	if r.conditionFn() {
		for _, o := range r.rl.GetObjects() {
			if o.GetAPIVersion() == r.cfg.For.ObjectRef.APIVersion && o.GetKind() == r.cfg.For.ObjectRef.Kind {
				if r.cfg.For.PopulateFn != nil {
					// call the external fn, which has the knowledge on how to populate the resources
					res, err := r.cfg.For.PopulateFn(o)
					if err != nil {
						r.rl.AddResult(err, o)
					} else {
						for objRef, newObj := range res {
							newObj.SetAnnotation(FnRuntimeOwner, kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef))
							r.inv.AddNewResource(&corev1.ObjectReference{
								APIVersion: objRef.APIVersion,
								Kind:       objRef.Kind,
								Name:       o.GetName(),
							}, newObj)
						}
					}
				}
			}
		}
	}
}

// update call the diff on inventory to find out the actions to take to make align the package
// based on the latest data.
func (r *upstreamFnRuntime) update() {
	// kptfile
	kf := kptfilelibv1.NewMutator(r.rl.GetObjects()[0].String())
	var err error
	if _, err = kf.UnMarshal(); err != nil {
		fn.Log("error unmarshal kptfile")
		r.rl.AddResult(err, r.rl.GetObjects()[0])
	}

	// perform a diff
	diff, err := r.inv.Diff()
	if err != nil {
		r.rl.AddResult(err, r.rl.GetObjects()[0])
	}

	if !r.conditionFn() {
		// set deletion timestamp on all resources
		for _, obj := range diff.DeleteObjs {
			fn.Logf("create set condition: %s\n", kptfilelibv1.GetConditionType(&obj.Ref))
			// set condition
			kf.SetConditions(kptv1.Condition{
				Type:    kptfilelibv1.GetConditionType(&obj.Ref),
				Status:  kptv1.ConditionFalse,
				Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef), obj.Obj.GetName()),
				Message: "cluster context has no site id",
			})
			// update the release timestamp
			obj.Obj.SetAnnotation(FnRuntimeDelete, "true")
			r.rl.SetObject(&obj.Obj)
		}
		return
	} else {
		for _, obj := range diff.CreateConditions {
			fn.Logf("create condition: %s\n", kptfilelibv1.GetConditionType(&obj.Ref))
			// create condition again
			kf.SetConditions(kptv1.Condition{
				Type:    kptfilelibv1.GetConditionType(&obj.Ref),
				Status:  kptv1.ConditionFalse,
				Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef), obj.Obj.GetName()),
				Message: "create condition again as it was deleted",
			})
		}
		for _, obj := range diff.DeleteConditions {
			fn.Logf("delete condition: %s\n", kptfilelibv1.GetConditionType(&obj.Ref))
			// delete condition
			kf.DeleteCondition(kptfilelibv1.GetConditionType(&obj.Ref))
		}
		for _, obj := range diff.CreateObjs {
			fn.Logf("create set condition: %s\n", kptfilelibv1.GetConditionType(&obj.Ref))
			// create condition - add resource to resource list
			kf.SetConditions(kptv1.Condition{
				Type:    kptfilelibv1.GetConditionType(&obj.Ref),
				Status:  kptv1.ConditionFalse,
				Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef), obj.Obj.GetName()),
				Message: "create new resource",
			})

			if r.cfg.Owns[corev1.ObjectReference{APIVersion: obj.Ref.APIVersion, Kind: obj.Ref.Kind}] == UpstreamRuntimeConfigOperationDefault {
				//TODO: wait for latest lib
				r.rl.AddObject(&obj.Obj)
			}
		}
		for _, obj := range diff.UpdateObjs {
			fn.Logf("update set condition: %s\n", kptfilelibv1.GetConditionType(&obj.Ref))
			// update condition - add resource to resource list
			kf.SetConditions(kptv1.Condition{
				Type:    kptfilelibv1.GetConditionType(&obj.Ref),
				Status:  kptv1.ConditionFalse,
				Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef), obj.Obj.GetName()),
				Message: "update existing resource",
			})
			if r.cfg.Owns[corev1.ObjectReference{APIVersion: obj.Ref.APIVersion, Kind: obj.Ref.Kind}] == UpstreamRuntimeConfigOperationDefault {
				r.rl.SetObject(&obj.Obj)
			}
		}
		for _, obj := range diff.DeleteObjs {
			fn.Logf("update set condition: %s\n", kptfilelibv1.GetConditionType(&obj.Ref))
			// create condition - add resource to resource list
			kf.SetConditions(kptv1.Condition{
				Type:    kptfilelibv1.GetConditionType(&obj.Ref),
				Status:  kptv1.ConditionFalse,
				Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For.ObjectRef), obj.Obj.GetName()),
				Message: "delete existing resource",
			})
			// update resource to resoucelist with delete Timestamp set
			obj.Obj.SetAnnotation(FnRuntimeDelete, "true")
			r.rl.SetObject(&obj.Obj)
		}
	}

	kptfile, err := kf.ParseKubeObject()
	if err != nil {
		fn.Log(err)
		r.rl.AddResult(err, r.rl.GetObjects()[0])
	}
	r.rl.SetObject(kptfile)
}
