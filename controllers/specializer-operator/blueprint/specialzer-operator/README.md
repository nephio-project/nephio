# specializer-operator

## Description
specializer-operator controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] specializer-operator`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree specializer-operator`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init specializer-operator
kpt live apply specializer-operator --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
