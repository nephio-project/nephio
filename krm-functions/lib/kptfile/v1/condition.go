package v1

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetConditionType(o *corev1.ObjectReference) string {
	var sb strings.Builder
	sb.Reset()
	if o.APIVersion != "" {
		gvk, _ := schema.ParseKindArg(o.APIVersion)
		if gvk != nil {
			sb.WriteString(gvk.Group)
		}
	}
	if o.Kind != "" {
		if sb.String() != "" {
			sb.WriteString(".")
		}
		sb.WriteString(o.Kind)
	}
	if o.Name != "" {
		if sb.String() != "" {
			sb.WriteString(".")
		}
		sb.WriteString(o.Name)
	}
	return sb.String()
}
