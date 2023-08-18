package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"

	fnr "github.com/nephio-project/nephio/krm-functions/gen-configmap-fn/fn"
)

var _ fn.ResourceListProcessor = &fnr.GenConfigMap{}

func main() {
	if err := fn.AsMain(&fnr.GenConfigMap{}); err != nil {
		os.Exit(1)
	}
}
