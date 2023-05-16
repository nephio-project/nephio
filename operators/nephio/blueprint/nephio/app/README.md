# app

## Description
nephio app

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] app`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree app`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init app
kpt live apply app --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
