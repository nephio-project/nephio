package kptrl

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type ResourceList interface {
	AddResult(err error, obj *fn.KubeObject)
	GetObjects() fn.KubeObjects
	SetObject(obj *fn.KubeObject)
	SetObjectWithDeleteTimestamp(obj *fn.KubeObject)
	AddObject(obj *fn.KubeObject)
	DeleteObject(obj *fn.KubeObject)
}

func New(rl *fn.ResourceList) ResourceList {
	return &resourceList{
		rl: rl,
	}
}

type resourceList struct {
	rl *fn.ResourceList
}

func (r *resourceList) AddResult(err error, obj *fn.KubeObject) {
	r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, obj))
}

func (r *resourceList) GetObjects() fn.KubeObjects {
	return r.rl.Items
}

func (r *resourceList) SetObject(obj *fn.KubeObject) {
	exists := false
	for idx, o := range r.rl.Items {
		if o.GetAPIVersion() == obj.GetAPIVersion() && o.GetKind() == obj.GetKind() && o.GetName() == obj.GetName() {
			r.rl.Items[idx] = obj
			exists = true
			break
		}
	}
	if !exists {
		r.AddObject(obj)
	}
}

func (r *resourceList) SetObjectWithDeleteTimestamp(obj *fn.KubeObject) {
	for idx, o := range r.rl.Items {
		if o.GetAPIVersion() == obj.GetAPIVersion() && o.GetKind() == obj.GetKind() && o.GetName() == obj.GetName() {
			u := &unstructured.Unstructured{}
			yaml.Unmarshal([]byte(obj.String()), u)
			t := metav1.Now()
			u.SetDeletionTimestamp(&t)

			r.rl.Items[idx] = obj
			break
		}
	}
}

func (r *resourceList) AddObject(obj *fn.KubeObject) {
	r.rl.Items = append(r.rl.Items, obj)
}

func (r *resourceList) DeleteObject(obj *fn.KubeObject) {
	for idx, o := range r.rl.Items {
		if o.GetAPIVersion() == obj.GetAPIVersion() && o.GetKind() == obj.GetKind() && o.GetName() == obj.GetName() {
			r.rl.Items = append(r.rl.Items[:idx], r.rl.Items[idx+1:]...)
		}
	}
}
