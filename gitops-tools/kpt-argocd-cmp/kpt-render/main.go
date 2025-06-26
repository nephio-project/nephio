package main

import (
	"fmt"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"log"
	"os"
	"path/filepath"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	err := filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				cleanedPath := filepath.Clean(path)
				bytes, err := os.ReadFile(cleanedPath)
				if err != nil {
					log.Printf("ERROR: Failed to read file %s: %v", path, err)
					return err
				}
				kos, err := fn.ParseKubeObjects(bytes)
				if err != nil {
					log.Printf("INFO: Ignoring non-KRM file (failed to parse KubeObjects): %s", path)
					return nil
				}
				for _, ko := range kos {
					kustomizationGK := schema.GroupKind{Group: "kustomize.config.k8s.io", Kind: "Kustomization"}
					isKustomization := ko.IsGroupKind(kustomizationGK)
					isLocalConfig := ko.GetAnnotation("config.kubernetes.io/local-config") == "true"

					if !isKustomization && !isLocalConfig {
						log.Printf("INFO: Resource included for deployment: %s/%s (kind: %s, path: %s)",
							ko.GetNamespace(), ko.GetName(), ko.GetKind(), path)
						fmt.Print(ko.String())
						fmt.Println("---")
					} else {
						log.Printf("INFO: Resource ignored (local config or Kustomization file): %s/%s (kind: %s, path: %s)",
							ko.GetNamespace(), ko.GetName(), ko.GetKind(), path)
					}
				}
			}

			return nil
		})
	if err != nil {
		log.Printf("ERROR: Error walking directory: %v", err)
	}
}
