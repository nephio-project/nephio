package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/mutator"
)

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(mutator.Run)); err != nil {
		os.Exit(1)
	}
}
