# crd

## Description
nephio crd

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] crd`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree crd`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init crd
kpt live apply crd --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
