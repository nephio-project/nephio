package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	fnr "github.com/nephio-project/nephio/krm-functions/nfdeploy-fn/fn"
)

func main() {

	if err := fn.AsMain(fn.ResourceListProcessorFunc(fnr.Run)); err != nil {
		os.Exit(1)
	}
}
