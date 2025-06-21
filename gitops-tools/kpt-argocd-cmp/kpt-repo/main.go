package main

import (
	"fmt"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func outputArgoApplication(kptFile *fn.KubeObject, path string) error {
	destinationName := os.Getenv("ARGOCD_ENV_DESTINATION_NAME") 
	fmt.Fprintf(os.Stderr, "destination:\n")
       	fmt.Fprintf(os.Stderr, destinationName)
	projectName := os.Getenv("ARGOCD_APP_PROJECT_NAME") 
	fmt.Fprintf(os.Stderr, "projectName:\n")
        fmt.Fprintf(os.Stderr, projectName)
	if destinationName == "" || projectName == "" {
		return nil
	}
	dirPath := os.Getenv("ARGOCD_APP_SOURCE_PATH") + "/" + filepath.Dir(path)
	fmt.Fprintf(os.Stderr, "dirPath:\n")
        fmt.Fprintf(os.Stderr, dirPath)
	name := os.Getenv("ARGOCD_APP_NAME") + "-" + kptFile.GetName()
	
	repo_url := kptFile.GetString("repo")

	ko, err := fn.ParseKubeObject([]byte(`
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: app
  namespace: argocd
spec:
  destination:
    namespace: default
    name: __NAME__
  project: __PROJECT__
  source:
    path: __PATH__
    plugin:
      name: kpt-render
      env:
    repoURL: __REPO_URL__
    targetRevision: main
  syncPolicy:
    automated:
      prune: true
      selfHeal: true`))
	if err != nil {
		return err
	}
	if repo_url == "" {
		ko.SetNestedField(os.Getenv("ARGOCD_APP_SOURCE_REPO_URL"), "spec", "source", "repoURL")
	} else {
		ko.SetNestedField(repo_url, "spec", "source", "repoURL")
	}
	ko.SetNestedField(dirPath, "spec", "source", "path")
	ko.SetNestedField(destinationName, "spec", "destination", "name")
	ko.SetNestedField(projectName, "spec", "project")
	ko.SetName(name)
	fmt.Print(ko.String())
	fmt.Println("---")
	return nil
}

func main() {
	err := filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(path, "Kptfile") {
				bytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				ko, err := fn.ParseKubeObject(bytes)
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "path:\n")
        			fmt.Fprintf(os.Stderr, path)
				err = outputArgoApplication(ko, path)
				if err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
