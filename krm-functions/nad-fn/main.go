package main

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/mutator"
	"os"
)

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(mutator.Run)); err != nil {
		os.Exit(1)
	}
}
