package main

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/nad-fn/mutator"
	"os"
)

func main() {
	//	dat, _ := os.Open("C:\\Users\\ganesh.c\\Documents\\GitHub\\pkg-examples\\nadfn\\created.yaml")
	//	os.Stdin = dat
	if err := fn.AsMain(fn.ResourceListProcessorFunc(mutator.Run)); err != nil {
		os.Exit(1)
	}
}
