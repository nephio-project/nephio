# specialzers

## Description
specialzers controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] specialzers`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree specialzers`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init specialzers
kpt live apply specialzers --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
