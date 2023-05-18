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
package reconcilerinterface

import (
	"github.com/henderiw-nephio/nephio-controllers/pkg/giteaclient"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// ControllerConfig: this is going to be a common struct required by all
// reconcilers for accepting config env parameters
// move this to a config dir .. TBD
type ControllerConfig struct {
	PorchClient client.Client
	GiteaClient giteaclient.GiteaClient
	Poll        time.Duration
	Copts       controller.Options
	Address     string // backend server address
}

type Reconciler interface {
	reconcile.Reconciler

	// Setup registers the reconciler to run under the specified manager
	SetupWithManager(ctrl.Manager, interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error)
}

var Reconcilers = map[string]Reconciler{}

func Register(name string, r Reconciler) {
	Reconcilers[name] = r
}
