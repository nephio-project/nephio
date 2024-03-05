/*
Copyright 2023 The Nephio Authors.

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

package giteaclient

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GiteaClient interface {
	Start(ctx context.Context)
	IsInitialized() bool
	Get() *gitea.Client
	GetMyUserInfo() (*gitea.User, *gitea.Response, error)
	DeleteRepo(owner string, repo string) (*gitea.Response, error)
	GetRepo(userName string, repoCRName string) (*gitea.Repository, *gitea.Response, error)
	CreateRepo(createRepoOption gitea.CreateRepoOption) (*gitea.Repository, *gitea.Response, error)
	EditRepo(userName string, repoCRName string, editRepoOption gitea.EditRepoOption) (*gitea.Repository, *gitea.Response, error)
	DeleteAccessToken(value interface{}) (*gitea.Response, error)
	ListAccessTokens(opts gitea.ListAccessTokensOptions) ([]*gitea.AccessToken, *gitea.Response, error)
	CreateAccessToken(opt gitea.CreateAccessTokenOption) (*gitea.AccessToken, *gitea.Response, error)
}

var lock = &sync.Mutex{}

var singleInstance *gc

func GetClient(ctx context.Context, client resource.APIPatchingApplicator) (GiteaClient, error) {
	if ctx == nil {
		return nil, fmt.Errorf("failed creating gitea client, value of ctx cannot be nil")
	}

	if client.Client == nil {
		return nil, fmt.Errorf("failed creating gitea client, value of client.Client cannot be nil")
	}
	// check if an instance is created using check-lock-check pattern implementation
	if singleInstance == nil {
		// Create a lock
		lock.Lock()
		defer lock.Unlock()
		// Check instance is still null as another thread of execution may have initialized it before the lock was acquired.
		if singleInstance == nil {
			singleInstance = &gc{client: client}
			log.FromContext(ctx).Info("Gitea Client Instance created now.")
			go singleInstance.Start(ctx)
		} else {
			log.FromContext(ctx).Info("Gitea Client Instance already created.")
		}
	} else {
		log.FromContext(ctx).Info("Gitea Client Instance already created.")
	}
	return singleInstance, nil
}

type gc struct {
	client resource.APIPatchingApplicator

	giteaClient *gitea.Client
	l           logr.Logger
}

func (r *gc) Start(ctx context.Context) {
	for {
		select {
		// The context is the one returned by ctrl.SetupSignalHandler().
		// cancel() of this context will trigger <- ctx.Done().
		// The Idea for continuously retrying is for enabling the user to
		// create a secret eventually even after the controllers are started.
		case <-ctx.Done():
			fmt.Printf("controller manager context cancelled: Exit\n")
			return
		default:
			r.l = log.FromContext(ctx)
			//var err error
			time.Sleep(5 * time.Second)

			gitURL, ok := os.LookupEnv("GIT_URL")
			if !ok {
				r.l.Error(fmt.Errorf("git url not defined"), "cannot connect to git server")
				break
			}

			namespace := os.Getenv("POD_NAMESPACE")
			if gitNamespace, ok := os.LookupEnv("GIT_NAMESPACE"); ok {
				namespace = gitNamespace
			}
			secretName := "git-user-secret"
			if gitSecretName, ok := os.LookupEnv("GIT_SECRET_NAME"); ok {
				secretName = gitSecretName
			}

			// get secret that was created when installing gitea
			secret := &corev1.Secret{}
			if err := r.client.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      secretName,
			},
				secret); err != nil {
				r.l.Error(err, "Cannot get secret, please follow README and create the gitea secret")
				break
			}

			// To create/list tokens we can only use basic authentication using username and password
			giteaClient, err := gitea.NewClient(
				gitURL,
				getClientAuth(secret))
			if err != nil {
				r.l.Error(err, "cannot authenticate to gitea")
				break
			}

			r.giteaClient = giteaClient
			r.l.Info("gitea init done")
			return
		}
	}
}

func getClientAuth(secret *corev1.Secret) gitea.ClientOption {
	return gitea.SetBasicAuth(string(secret.Data["username"]), string(secret.Data["password"]))
}

func (r *gc) IsInitialized() bool {
	return r.giteaClient != nil
}

func (r *gc) Get() *gitea.Client {
	return r.giteaClient
}

func (r *gc) GetMyUserInfo() (*gitea.User, *gitea.Response, error) {
	return r.giteaClient.GetMyUserInfo()
}

func (r *gc) DeleteRepo(owner string, repo string) (*gitea.Response, error) {
	return r.giteaClient.DeleteRepo(owner, repo)
}

func (r *gc) GetRepo(userName string, repoCRName string) (*gitea.Repository, *gitea.Response, error) {
	return r.giteaClient.GetRepo(userName, repoCRName)
}

func (r *gc) CreateRepo(createRepoOption gitea.CreateRepoOption) (*gitea.Repository, *gitea.Response, error) {
	return r.giteaClient.CreateRepo(createRepoOption)
}

func (r *gc) EditRepo(userName string, repoCRName string, editRepoOption gitea.EditRepoOption) (*gitea.Repository, *gitea.Response, error) {
	return r.giteaClient.EditRepo(userName, repoCRName, editRepoOption)
}

func (r *gc) DeleteAccessToken(value interface{}) (*gitea.Response, error) {
	return r.giteaClient.DeleteAccessToken(value)
}

func (r *gc) ListAccessTokens(opts gitea.ListAccessTokensOptions) ([]*gitea.AccessToken, *gitea.Response, error) {
	return r.giteaClient.ListAccessTokens(opts)
}

func (r *gc) CreateAccessToken(opt gitea.CreateAccessTokenOption) (*gitea.AccessToken, *gitea.Response, error) {
	return r.giteaClient.CreateAccessToken(opt)
}
