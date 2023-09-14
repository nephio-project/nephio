# gen-configmap-fn

## Overview

<!--mdtogo:Short-->

This function is a simple ConfigMap generator.

<!--mdtogo-->

Application config is often captured in ConfigMaps, but it is difficult to edit
the contents of the ConfigMap with KRM functions. This allows function inputs to
be used to generate ConfigMaps, and those inputs can be more easily edited than
the raw ConfigMap data values.

<!--mdtogo:Long-->

## Usage

To use this function, define a GenConfigMap resource for each ConfigMap you want
to generate. It can be used declaratively or imperatively, but it does require
the GenConfigMap Kind; you cannot run it using a simple ConfigMap for inputs.

When run, it will create or overwrite a ConfigMap, and generate `data` field
entries for each of the listed values in the function config.

### FunctionConfig

```yaml
apiVersion: fn.kpt.dev/v1alpha1
kind: GenConfigMap
metadata:
  name: my-generator
configMapMetadata:
  name: my-configmap
  labels:
    foo: bar
params:
  hostname: foo.example.com
  port: 8992
  scheme: https
  region: us-east1
data:
- type: literal
  key: hello
  value: there
- type: gotmpl
  key: dburl
  value: "{{.scheme}}://{{.hostname}}:{{.port}}/{{.region}}/db"
```

The function config above will generate the following ConfigMap:

```yaml
kind: ConfigMap
metadata:
  name: my-configmap
   labels:
    foo: bar
data:
  dburl: https://foo.example.com:8992/us-east1/db
  hello: there
```

If `configMapMetadata.name` is not defined, the generated ConfigMap will use the
name of the GenConfigMap resource.

<!--mdtogo-->

## Future Work

This function is very basic right now. Integration with CEL and implementation
of features found in the Kustomize ConfigMapGenerator would be valuable.
