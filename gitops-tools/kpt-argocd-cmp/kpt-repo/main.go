package main

import (
	"fmt"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func outputArgoApplication(kptFile *fn.KubeObject, path string) error {
	destinationName := os.Getenv("ARGOCD_ENV_DESTINATION_NAME")
	projectName := os.Getenv("ARGOCD_APP_PROJECT_NAME")
	if destinationName == "" || projectName == "" {
		return nil
	}
	dirPath := os.Getenv("ARGOCD_APP_SOURCE_PATH") + "/" + filepath.Dir(path)

	log.Printf("INFO: CMP Inputs - destinationName='%s', projectName='%s', dirPath='%s'", destinationName, projectName, dirPath)

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
		err := ko.SetNestedField(os.Getenv("ARGOCD_APP_SOURCE_REPO_URL"), "spec", "source", "repoURL")
		if err != nil {
			return err
		}
	} else {
		err := ko.SetNestedField(repo_url, "spec", "source", "repoURL")
		if err != nil {
			return err
		}
	}
	err = ko.SetNestedField(dirPath, "spec", "source", "path")
	if err != nil {
		return err
	}
	err = ko.SetNestedField(destinationName, "spec", "destination", "name")
	if err != nil {
		return err
	}
	err = ko.SetNestedField(projectName, "spec", "project")
	if err != nil {
		return err
	}
	err = ko.SetName(name)
	if err != nil {
		return err
	}

	log.Printf("INFO: Successfully generated Argo CD Application '%s'.", name)

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
				cleanedPath := filepath.Clean(path)
				if !strings.HasPrefix(cleanedPath, ".") {
					log.Println("ERROR: Invalid path")
					return nil
				}
				bytes, err := os.ReadFile(cleanedPath)
				if err != nil {
					return err
				}
				ko, err := fn.ParseKubeObject(bytes)
				if err != nil {
					return err
				}
				err = outputArgoApplication(ko, path)
				if err != nil {
					log.Printf("ERROR: Failed to generate Argo Application from Kptfile: %v", err)
					return err
				}
			}

			return nil
		})
	if err != nil {
		log.Printf("ERROR: Failed to walk directory: %v", err)
		os.Exit(1)
	}
}
