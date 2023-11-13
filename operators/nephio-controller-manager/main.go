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

	porchclient "github.com/nephio-project/nephio/controllers/pkg/porch/client"
	ctrlrconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconciler "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/ipam"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/vlan"
	"go.uber.org/zap/zapcore"

	//"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

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
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/generic-specializer"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/network"

	//_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/ipam-specializer"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/repository"
	_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/token"
	//_ "github.com/nephio-project/nephio/controllers/pkg/reconcilers/vlan-specializer"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enabledReconcilersString string

	//klog.InitFlags(nil)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&enabledReconcilersString, "reconcilers", "", "reconcilers that should be enabled; use * to mean 'enable all'")

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	/*
		if len(flag.Args()) != 0 {
			setupLog.Errorf("unexpected additional (non-flag) arguments: %v", flag.Args())
			os.Exit(1)
		}
	*/

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "cannot initializer schema")
		//klog.Errorf("error initializing scheme: %s", err.Error())
		os.Exit(1)
	}
	err := porchclient.AddToScheme(scheme)
	if err != nil {
		setupLog.Error(err, "cannot initializer schema with porch API(s)")
		//	klog.Errorf("error initializing scheme with Porch APIs: %s", err.Error())
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
		setupLog.Error(err, "cannot create porch client")
		//klog.Errorf("unable to create porch client: #{err}")
		os.Exit(1)
	}

	porchRESTClient, err := porchclient.CreateRESTClient(ctrl.GetConfigOrDie())
	if err != nil {
		setupLog.Error(err, "cannot create porch REST client")
		//klog.Errorf("error creating porch REST client: %s", err.Error())
		os.Exit(1)
	}
	ctx := ctrl.SetupSignalHandler()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), managerOptions)
	if err != nil {
		setupLog.Error(err, "cannot create manager")
		//klog.Errorf("error creating manager: #{err}")
		os.Exit(1)
	}

	// Prepare configuration for reconcilers
	backendAddress := "127.0.0.1:9999"
	if address, ok := os.LookupEnv("CLIENT_PROXY_ADDRESS"); ok {
		backendAddress = address
	}

	ctrlCfg := &ctrlrconfig.ControllerConfig{
		Address:         backendAddress,
		PorchClient:     porchClient,
		PorchRESTClient: porchRESTClient,
		IpamClientProxy: ipam.New(ctx, clientproxy.Config{
			Address: backendAddress,
		}),
		VlanClientProxy: vlan.New(ctx, clientproxy.Config{
			Address: backendAddress,
		}),
	}

	enabledReconcilers := parseReconcilers(enabledReconcilersString)
	var enabled []string
	for name, r := range reconciler.Reconcilers {
		if !reconcilerIsEnabled(enabledReconcilers, name) {
			continue
		}
		if _, err = r.SetupWithManager(ctx, mgr, ctrlCfg); err != nil {
			setupLog.Error(err, "cannot setup with manager", "reconciler", name)
			//klog.Errorf("error creating %q reconciler: %s", name, err.Error())
			os.Exit(1)
		}
		enabled = append(enabled, name)
	}

	if len(enabled) == 0 {
		setupLog.Info("no reconcilers are enabled; did you forget to pass the --reconcilers flag?")
		//klog.Warningf("no reconcilers are enabled; did you forget to pass the --reconcilers flag?")
	} else {
		setupLog.Info("enabled reconcilers", "reconcilers", strings.Join(enabled, ","))
		//klog.Infof("enabled reconcilers: %v", strings.Join(enabled, ","))
	}

	//+kubebuilder:scaffold:builder
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
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
	if val, found := os.LookupEnv(fmt.Sprintf("ENABLE_%s", strings.ToUpper(reconciler))); found {
		if val == "true" {
			return true
		}
	}
	return false
}
