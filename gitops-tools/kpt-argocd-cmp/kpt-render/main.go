package main

import (
	"fmt"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"log"
	"os"
	"path/filepath"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	err := filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				bytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				kos, err := fn.ParseKubeObjects(bytes)
				if err != nil {
					// ignore non KRM files
					fmt.Fprintf(os.Stderr, "Ignored:\n")
					fmt.Fprintf(os.Stderr, path)
					return nil
				}
				for _, ko := range kos{
					kustomizationGK := schema.GroupKind{Group: "kustomize.config.k8s.io", Kind: "Kustomization"}
					isKustomization := ko.IsGroupKind(kustomizationGK)
					isLocalConfig := ko.GetAnnotation("config.kubernetes.io/local-config") == "true"

					if !isKustomization && !isLocalConfig {
						fmt.Fprintf(os.Stderr, "Hit:\n")
                                        	fmt.Fprintf(os.Stderr, path)					
						fmt.Print(ko.String())
						fmt.Println("---")
					} else {
						fmt.Fprintf(os.Stderr, "Miss:\n")
                                        	fmt.Fprintf(os.Stderr, path)					
					}
				}
			}

			return nil
		})
	if err != nil {
		log.Println(err)
	}
}
