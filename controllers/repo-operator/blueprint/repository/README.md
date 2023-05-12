# repository

## Description
repository controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] repository`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree repository`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init repository
kpt live apply repository --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
