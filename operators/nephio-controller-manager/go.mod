module github.com/nephio-project/nephio/operators/nephio-controller-manager

go 1.20

replace k8s.io/api => k8s.io/api v0.26.1

replace k8s.io/apimachinery => k8s.io/apimachinery v0.26.1

replace k8s.io/client-go => k8s.io/client-go v0.26.1

require (
	github.com/nephio-project/nephio-controller-poc v0.0.2
	github.com/nephio-project/nephio/controllers/pkg v0.0.0-20230527153803-31b37fcf142b
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc
	k8s.io/apimachinery v0.27.2
	k8s.io/client-go v0.27.2
	k8s.io/klog/v2 v2.100.1
	sigs.k8s.io/controller-runtime v0.14.6
)

require (
	github.com/davidmz/go-pageant v1.0.2 // indirect
	github.com/go-fed/httpsig v1.1.0 // indirect
	golang.org/x/crypto v0.3.0 // indirect
)

require (
	code.gitea.io/sdk/gitea v0.15.1-0.20230509035020-970776d1c1e9 // indirect
	github.com/GoogleContainerTools/kpt v1.0.0-beta.29.0.20230327202912-01513604feaa // indirect
	github.com/GoogleContainerTools/kpt-functions-sdk/go/api v0.0.0-20220720212527-133180134b93 // indirect
	github.com/GoogleContainerTools/kpt-functions-sdk/go/fn v0.0.0-20230302070146-e8e9cb3c3ae2 // indirect
	github.com/GoogleContainerTools/kpt/porch/api v0.0.0-20230504200302-14c7b353e6b6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.10.2 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/hansthienpondt/nipam v0.0.5 // indirect
	github.com/hashicorp/go-version v1.5.0 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kentik/patricia v1.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nephio-project/api v0.0.0-20230522173958-63a41669b495 // indirect
	github.com/nephio-project/nephio/krm-functions/ipam-fn v0.0.0-20230519080401-f95bbb7f58a6 // indirect
	github.com/nephio-project/nephio/krm-functions/lib v0.0.0-20230508215739-b13457eda5c9 // indirect
	github.com/nephio-project/nephio/krm-functions/vlan-fn v0.0.0-20230519080401-f95bbb7f58a6 // indirect
	github.com/nokia/k8s-ipam v0.0.4-0.20230508220232-534a4724d032 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.15.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go4.org/netipx v0.0.0-20230303233057-f1b76eb4bb35 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/term v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.27.2 // indirect
	k8s.io/apiextensions-apiserver v0.27.1 // indirect
	k8s.io/component-base v0.27.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.12.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
