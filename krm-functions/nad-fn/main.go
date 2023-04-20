package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	//"github.com/nephio-project/nephio/krm-functions/nad-fn/mutator"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/mutatordownstream"
)

func main() {
	dat, _ := os.Open("C:\\Users\\ganesh.c\\Documents\\GitHub\\ganchandrasekaran\\nephio\\krm-functions\\nad-fn\\kptdata.yaml")
	os.Stdin = dat
	if err := fn.AsMain(fn.ResourceListProcessorFunc(mutatordownstream.Run)); err != nil {
		os.Exit(1)
	}
}
