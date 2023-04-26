package main

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/nfdeploy-fn/mutator"
	"os"
)

func main() {
	runner := fn.ResourceListProcessorFunc(mutator.Run)

	if err := fn.AsMain(runner); err != nil {
		os.Exit(1)
	}
}
