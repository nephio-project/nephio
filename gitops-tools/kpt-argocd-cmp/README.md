# kpt-argocd-cmp
Nephio heavily relies on kpt to package, render, mutate, validate and generate Kubernetes objects. ArgoCD doesn't currently have a built-in plugin to handle installation of manifests, and, as such, this repo introduces a plugin specifically built to render the kpt package pipeline properly. It consists of two [Conifg Management Plugins](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/) (CMPs) for ArgoCD that handle the creation of package applications and local configs.

This work is adapted from the [treactor-krm-functions/argo](https://github.com/treactor/treactor-krm-functions/tree/main/argo) PoC.

## kpt-repo
This plugin creates an "app-of-apps" style ArgoCD Application that takes a source repository and looks for Kptfiles to create ArgoCD applications for the cooresponding packages.

## kpt-render
The applications created by `kpt-repo` will target a second plugin, `kpt-render`, that filters out KRM files with the `config.kubernetes.io/local-config: "true"` annotation, or,  that are Kustomize files. This deals with a primary limitation of ArgoCD, where for plain yaml packages, it will attempt to install the `local-config` manifests into the destination cluster.  

## patch.sh
This file applies the plugins to the `argocd-repo-server` pod, using the images created via the corresponding Dockerfiles and pushed to a registery. Once patched, `argo-repo-server` will create containers for each plugin based on the images provided, and start the plugins. 

## Usage
In order for one to use the plugin, our prefered method is to target `kpt-repo` as the plugin for an ArgoCD Application. This mapping works in our use case as this "app-of-apps" Application represents a repository source and cluster destination. There are other methods to target CMPs (such as discovery rules) that are outside of the scope of this work.
