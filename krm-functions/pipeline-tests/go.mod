module github.com/nephio-project/nephio/krm-functions/pipeline-tests

go 1.20

// always test the latest versions of all functions/libs
replace (
	github.com/nephio-project/nephio/krm-functions/dnn-fn => ../dnn-fn
	github.com/nephio-project/nephio/krm-functions/interface-fn => ../interface-fn
	github.com/nephio-project/nephio/krm-functions/ipam-fn => ../ipam-fn
	github.com/nephio-project/nephio/krm-functions/lib => ../lib
	github.com/nephio-project/nephio/krm-functions/nad-fn => ../nad-fn
	github.com/nephio-project/nephio/krm-functions/nfdeploy-fn => ../nfdeploy-fn
	github.com/nephio-project/nephio/krm-functions/vlan-fn => ../vlan-fn
)

require (
	github.com/GoogleContainerTools/kpt-functions-sdk/go/fn v0.0.0-20230427202446-3255accc518d
	github.com/nephio-project/api v0.0.0-20230605215709-2e793ab3e7f2
	github.com/nephio-project/nephio/krm-functions/dnn-fn v0.0.0-20230516034137-53f1f1859c10
	github.com/nephio-project/nephio/krm-functions/interface-fn v0.0.0-20230516034137-53f1f1859c10
	github.com/nephio-project/nephio/krm-functions/ipam-fn v0.0.0-20230516034137-53f1f1859c10
	github.com/nephio-project/nephio/krm-functions/lib v0.0.0-20230605213956-a1e470f419a4
	github.com/nephio-project/nephio/krm-functions/nad-fn v0.0.0-20230516034137-53f1f1859c10
	github.com/nephio-project/nephio/krm-functions/nfdeploy-fn v0.0.0-20230516034137-53f1f1859c10
	github.com/nephio-project/nephio/krm-functions/vlan-fn v0.0.0-00010101000000-000000000000
	github.com/nokia/k8s-ipam v0.0.4-0.20230604044109-366c07a61afe
)

require (
	cloud.google.com/go/compute v1.19.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/GoogleContainerTools/kpt v1.0.0-beta.29.0.20230327202912-01513604feaa // indirect
	github.com/GoogleContainerTools/kpt-functions-sdk/go/api v0.0.0-20230427202446-3255accc518d // indirect
	github.com/GoogleContainerTools/kpt/porch/api v0.0.0-20230504200302-14c7b353e6b6 // indirect
	github.com/GoogleContainerTools/kpt/rollouts v0.0.0-20230209223911-c6c49d0a0636 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bytecodealliance/wasmtime-go v0.39.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v20.10.17+incompatible // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v20.10.17+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/emicklei/go-restful/v3 v3.10.2 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/fvbommel/sortorder v1.0.1 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-containerregistry v0.11.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hansthienpondt/nipam v0.0.5 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jedib0t/go-pretty/v6 v6.4.4 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.4.0 // indirect
	github.com/kentik/patricia v1.2.0 // indirect
	github.com/klauspost/compress v1.15.8 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198 // indirect
	github.com/otiai10/copy v1.7.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prep/wasmexec v0.0.0-20220807105708-6554945c1dec // indirect
	github.com/prometheus/client_golang v1.15.1 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/cobra v1.6.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spyzhov/ajson v0.7.1 // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.starlark.net v0.0.0-20210901212718-87f333178d59 // indirect
	go4.org/netipx v0.0.0-20230303233057-f1b76eb4bb35 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/oauth2 v0.8.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.27.2 // indirect
	k8s.io/apiextensions-apiserver v0.27.2 // indirect
	k8s.io/apimachinery v0.27.2 // indirect
	k8s.io/cli-runtime v0.25.3 // indirect
	k8s.io/client-go v0.27.2 // indirect
	k8s.io/component-base v0.27.2 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230525220651-2546d827e515 // indirect
	k8s.io/kubectl v0.25.3 // indirect
	k8s.io/utils v0.0.0-20230505201702-9f6742963106 // indirect
	sigs.k8s.io/cli-utils v0.34.0 // indirect
	sigs.k8s.io/controller-runtime v0.15.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
