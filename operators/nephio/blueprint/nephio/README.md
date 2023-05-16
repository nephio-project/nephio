# nephio

## Description
nephio controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] nephio`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree nephio`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init nephio
kpt live apply nephio --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
