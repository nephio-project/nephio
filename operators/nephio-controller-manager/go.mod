module github.com/nephio-project/nephio/operators/nephio-controller-manager

go 1.22

replace (
	github.com/nephio-project/nephio/controllers/pkg => ../../controllers/pkg
	github.com/nephio-project/nephio/krm-functions/configinject-fn => ../../krm-functions/configinject-fn
	github.com/nephio-project/nephio/krm-functions/ipam-fn => ../../krm-functions/ipam-fn
	github.com/nephio-project/nephio/krm-functions/lib => ../../krm-functions/lib
	github.com/nephio-project/nephio/krm-functions/vlan-fn => ../../krm-functions/vlan-fn
)

require (
	github.com/nephio-project/nephio/controllers/pkg v0.0.0-20230531154408-a4237c40cb76
	github.com/nokia/k8s-ipam v0.0.4-0.20230628092530-8a292aec80a4
	go.uber.org/zap v1.26.0
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc
	k8s.io/apimachinery v0.29.2
	k8s.io/client-go v0.29.2
	sigs.k8s.io/controller-runtime v0.17.2

)

require (
	code.gitea.io/sdk/gitea v0.15.1-0.20230509035020-970776d1c1e9 // indirect
	github.com/GoogleContainerTools/kpt-functions-sdk/go/api v0.0.0-20230427202446-3255accc518d // indirect
	github.com/GoogleContainerTools/kpt-functions-sdk/go/fn v0.0.0-20230427202446-3255accc518d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/davidmz/go-pageant v1.0.2 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch/v5 v5.8.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-fed/httpsig v1.1.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hansthienpondt/nipam v0.0.5 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/henderiw-nephio/network v0.0.0-20230626193806-04743403261e // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kentik/patricia v1.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nephio-project/api v1.0.1-0.20231127124455-cf14bd57b08d // indirect
	github.com/nephio-project/nephio/krm-functions/configinject-fn v0.0.0-00010101000000-000000000000 // indirect
	github.com/nephio-project/nephio/krm-functions/ipam-fn v0.0.0-00010101000000-000000000000 // indirect
	github.com/nephio-project/nephio/krm-functions/lib v0.0.0-20230609191131-85aa39064ef8 // indirect
	github.com/nephio-project/nephio/krm-functions/vlan-fn v0.0.0-00010101000000-000000000000 // indirect
	github.com/nephio-project/porch v1.3.0 // indirect
	github.com/nephio-project/porch/api v1.3.0 // indirect
	github.com/nephio-project/porch/controllers v0.0.0-20240724125300-23d62c235f95 // indirect
	github.com/openconfig/gnmi v0.9.1 // indirect
	github.com/openconfig/goyang v1.4.0 // indirect
	github.com/openconfig/ygot v0.28.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.18.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/srl-labs/ygotsrl/v22 v22.11.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go4.org/netipx v0.0.0-20230303233057-f1b76eb4bb35 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/oauth2 v0.18.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230913181813-007df8e322eb // indirect
	google.golang.org/grpc v1.59.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.29.2 // indirect
	k8s.io/apiextensions-apiserver v0.29.0 // indirect
	k8s.io/component-base v0.29.2 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00 // indirect
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57 // indirect
	sigs.k8s.io/cluster-api v1.4.0-beta.2.0.20230527123250-e111168cdff3 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.5-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.3-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
