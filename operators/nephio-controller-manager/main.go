/*
Copyright 2022-2023 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nephio-project/nephio/controllers/pkg/giteaclient"
	porchclient "github.com/nephio-project/nephio/controllers/pkg/porch/client"
	ctrlrconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconciler "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"golang.org/x/exp/slices"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2/klogr"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	//+kubebuilder:scaffold:imports

	// Import our reconcilers
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/approval"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/bootstrap-packages"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/bootstrap-secret"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/ipam-specializer"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/repository"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/token"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/vlan-specializer"
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enabledReconcilersString string

	klog.InitFlags(nil)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&enabledReconcilersString, "reconcilers", "", "reconcilers that should be enabled; use * to mean 'enable all'")

	flag.Parse()

	if len(flag.Args()) != 0 {
		klog.Errorf("unexpected additional (non-flag) arguments: %v", flag.Args())
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		klog.Errorf("error initializing scheme: %s", err.Error())
		os.Exit(1)
	}
	err := porchclient.AddToScheme(scheme)
	if err != nil {
		klog.Errorf("error initializing scheme with Porch APIs: %s", err.Error())
		os.Exit(1)
	}

	managerOptions := ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         metricsAddr,
		Port:                       9443,
		HealthProbeBindAddress:     probeAddr,
		LeaderElection:             false,
		LeaderElectionID:           "nephio-operators.nephio.org",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
	}

	ctrl.SetLogger(klogr.New())
	porchClient, err := porchclient.CreateClient(ctrl.GetConfigOrDie())
	if err != nil {
		klog.Errorf("unable to create porch client: #{err}")
		os.Exit(1)
	}

	porchRESTClient, err := porchclient.CreateRESTClient(ctrl.GetConfigOrDie())
	if err != nil {
		klog.Errorf("error creating porch REST client: %s", err.Error())
		os.Exit(1)
	}
	ctx := ctrl.SetupSignalHandler()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), managerOptions)
	if err != nil {
		klog.Errorf("error creating manager: #{err}")
		os.Exit(1)
	}

	// Start a Gitea Client
	// Prepare configuration for reconcilers
	clientProxy := "127.0.0.1:9999"
	if address, ok := os.LookupEnv("CLIENT_PROXY_ADDRESS"); ok {
		clientProxy = address
	}
	// Sending the porchclient to getgitea, this will be used to get
	// the secret objects for gitea client authentication. The client
	// of the manager of this controller cannot be used at this point.
	// Should this be conditional ? Only if we have repo/token reconciler
	g := giteaclient.New(resource.NewAPIPatchingApplicator(porchClient))
	go g.Start(ctx)

	enabledReconcilers := parseReconcilers(enabledReconcilersString)
	var enabled []string
	for name, r := range reconciler.Reconcilers {
		if !reconcilerIsEnabled(enabledReconcilers, name) {
			continue
		}
		if _, err = r.SetupWithManager(ctx, mgr, &ctrlrconfig.ControllerConfig{
			Address:         clientProxy,
			PorchClient:     porchClient,
			PorchRESTClient: porchRESTClient,
			GiteaClient:     g,
		}); err != nil {
			klog.Errorf("error creating %q reconciler: %s", name, err.Error())
			os.Exit(1)
		}
		enabled = append(enabled, name)
	}

	if len(enabled) == 0 {
		klog.Warningf("no reconcilers are enabled; did you forget to pass the --reconcilers flag?")
	} else {
		klog.Infof("enabled reconcilers: %v", strings.Join(enabled, ","))
	}

	//+kubebuilder:scaffold:builder
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Errorf("error adding health check: #{err}")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Errorf("error adding ready check: #{err}")
		os.Exit(1)
	}

	klog.Infof("starting manager")
	if err := mgr.Start(ctx); err != nil {
		klog.Errorf("error running manager: #{err}")
		os.Exit(1)
	}
}

func parseReconcilers(reconcilers string) []string {
	return strings.Split(reconcilers, ",")
}

func reconcilerIsEnabled(reconcilers []string, reconciler string) bool {
	if slices.Contains(reconcilers, "*") {
		return true
	}
	if slices.Contains(reconcilers, reconciler) {
		return true
	}
	if _, found := os.LookupEnv(fmt.Sprintf("ENABLE_%s", strings.ToUpper(reconciler))); found {
		return true
	}
	return false
}
