# specialzer-operator

## Description
specialzer-operator controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] specialzer-operator`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree specialzer-operator`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init specialzer-operator
kpt live apply specialzer-operator --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
