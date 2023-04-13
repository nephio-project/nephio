package fnruntime

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	corev1 "k8s.io/api/core/v1"
)

const FnRuntimeOwner = "fnruntime.nephio.org/owner"
const FnRuntimeDelete = "fnruntime.nephio.org/delete"

type FnRuntime interface {
	Run()
}

type WatchCallbackFn func(o *fn.KubeObject) error

type PopulateFn func(o *fn.KubeObject) (map[corev1.ObjectReference]*fn.KubeObject, error)

type ConditionFn func() bool

func conditionFnNop() bool {
	return true
}

type GenerateFn func(map[corev1.ObjectReference]fn.KubeObject) (*fn.KubeObject, error)
