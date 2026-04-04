package main

import (
	"fmt"
	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/lib/cel"
)

func main() {
	rl := &fn.ResourceList{
		Items: fn.KubeObjects{},
	}
	ko, _ := fn.ParseKubeObject([]byte(`
apiVersion: v1
kind: Service
metadata:
  name: foo
`))
	rl.Items = append(rl.Items, ko)

	res, err := cel.EvaluateCondition("items[0].metadata.name == 'foo'", rl)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", res)
	}

	res, err = cel.EvaluateCondition("items[0].metadata.name == 'bar'", rl)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", res)
	}
}
