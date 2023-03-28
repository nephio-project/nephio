package v1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestGetConditionType(t *testing.T) {
	type object struct {
		apiVersion string
		kind       string
		name       string
		dummy      string
	}

	tests := []struct {
		input object
		want  string
	}{
		{
			input: object{
				apiVersion: "a.a.a",
				kind:       "b",
				name:       "c",
			},
			want: "a.b.c",
		},
		{
			input: object{
				kind: "b",
				name: "c",
			},
			want: "b.c",
		},
		{
			input: object{
				apiVersion: "a.a",
				kind:       "b",
				name:       "c",
			},
			want: "b.c",
		},
		{
			input: object{
				name: "c",
			},
			want: "c",
		},
	}

	for _, tt := range tests {
		got := GetConditionType(&corev1.ObjectReference{
			APIVersion: tt.input.apiVersion,
			Kind:       tt.input.kind,
			Name:       tt.input.name,
			Namespace:  tt.input.dummy,
		})
		if got != tt.want {
			t.Errorf("got %s want %s", got, tt.want)
		}
	}
}
