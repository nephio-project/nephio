# pkg-upf

## Description
sample description

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] pkg-upf`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree pkg-upf`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init pkg-upf
kpt live apply pkg-upf --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
