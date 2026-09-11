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

package gitea

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/git"
	gittypes "github.com/nephio-project/nephio/controllers/pkg/git/types"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var lock = &sync.Mutex{}

type gc struct {
	client      resource.APIPatchingApplicator
	giteaClient *gitea.Client
	l           logr.Logger
}

var singleInstance *gc

func GetClient(ctx context.Context, client resource.APIPatchingApplicator) (git.Client, error) {
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

func (r *gc) Get() any {
	return r.giteaClient
}

func (r *gc) GetMyUserInfo() (*gittypes.User, *gittypes.Response, error) {
	userInfo, resp, err := r.giteaClient.GetMyUserInfo()
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.User{
		UserName: userInfo.UserName,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) DeleteRepo(owner string, repo string) (*gittypes.Response, error) {
	resp, err := r.giteaClient.DeleteRepo(owner, repo)
	if err != nil {
		return &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) GetRepo(userName string, repoCRName string) (*gittypes.Repository, *gittypes.Response, error) {
	repo, resp, err := r.giteaClient.GetRepo(userName, repoCRName)
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Repository{
		CloneURL: repo.CloneURL,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) CreateRepo(createRepoOption gittypes.CreateRepoOption) (*gittypes.Repository, *gittypes.Response, error) {
	repo, resp, err := r.giteaClient.CreateRepo(gitea.CreateRepoOption{
		Name:          createRepoOption.Name,
		Description:   createRepoOption.Description,
		Private:       createRepoOption.Private,
		IssueLabels:   createRepoOption.IssueLabels,
		AutoInit:      createRepoOption.AutoInit,
		Template:      createRepoOption.Template,
		Gitignores:    createRepoOption.Gitignores,
		License:       createRepoOption.License,
		Readme:        createRepoOption.Readme,
		DefaultBranch: createRepoOption.DefaultBranch,
		TrustModel:    gitea.TrustModel(createRepoOption.TrustModel),
	})
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Repository{
		CloneURL: repo.CloneURL,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) EditRepo(userName string, repoCRName string, editRepoOption gittypes.EditRepoOption) (*gittypes.Repository, *gittypes.Response, error) {
	repo, resp, err := r.giteaClient.EditRepo(userName, repoCRName, gitea.EditRepoOption{
		Name:        editRepoOption.Name,
		Description: editRepoOption.Description,
		Private:     editRepoOption.Private,
	})
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Repository{
		CloneURL: repo.CloneURL,
	}, &gittypes.Response{Response: resp.Response}, nil
}

// wrapResponse survives a nil response, which the SDK returns when it rejects
// the argument before making a request, and on a transport error.
func wrapResponse(resp *gitea.Response) *gittypes.Response {
	if resp == nil {
		return nil
	}
	return &gittypes.Response{Response: resp.Response}
}

// giteaReadsNameAsID reports whether Gitea's token endpoints resolve name as a
// token ID rather than as a name. They parse the path segment with
// ParseInt(s, 0, 64) and look it up as a name only when that yields zero, so
// base 0 pulls in "007" and "0x22" alongside the decimal names.
func giteaReadsNameAsID(name string) bool {
	id, err := strconv.ParseInt(name, 0, 64)
	return err == nil && id != 0
}

// DeleteAccessToken deletes a token by name or by ID. The SDK has accepted
// either since Gitea 1.13, so narrowing this to an ID left the reconciler,
// which holds the name, with nothing it could pass.
func (r *gc) DeleteAccessToken(value interface{}) (*gittypes.Response, error) {
	switch v := value.(type) {
	case int64:
	case string:
		// Sending a name Gitea reads as an ID deletes whichever token holds
		// that ID and answers 204, so resolve it to the right one first.
		if giteaReadsNameAsID(v) {
			return r.deleteAccessTokenNamed(v)
		}
	default:
		return nil, fmt.Errorf("DeleteAccessToken: value must be a token name or an int64 id, got %T", value)
	}
	resp, err := r.giteaClient.DeleteAccessToken(value)
	return wrapResponse(resp), err
}

// deleteAccessTokenNamed deletes the token carrying name, for the names a path
// segment cannot express.
func (r *gc) deleteAccessTokenNamed(name string) (*gittypes.Response, error) {
	// Page -1 disables pagination. A page holds 30 tokens by default and the
	// one being deleted may be past it.
	tokens, resp, err := r.giteaClient.ListAccessTokens(gitea.ListAccessTokensOptions{
		ListOptions: gitea.ListOptions{Page: -1},
	})
	if err != nil {
		return wrapResponse(resp), fmt.Errorf("finding the id of token %q: %w", name, err)
	}

	for _, token := range tokens {
		if token.Name == name {
			resp, err := r.giteaClient.DeleteAccessToken(token.ID)
			return wrapResponse(resp), err
		}
	}

	// Nothing carries the name, which is the state deletion asks for. Gitea
	// refuses a second token of an existing name, so one pass settles it.
	return nil, nil
}

func (r *gc) ListAccessTokens(opts gittypes.ListAccessTokensOptions) ([]*gittypes.AccessToken, *gittypes.Response, error) {
	giteaOpts := gitea.ListAccessTokensOptions{
		ListOptions: gitea.ListOptions{
			Page:     opts.Page,
			PageSize: opts.PageSize,
		},
	}
	tokens, resp, err := r.giteaClient.ListAccessTokens(giteaOpts)
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	var result []*gittypes.AccessToken
	for _, t := range tokens {
		result = append(result, &gittypes.AccessToken{
			ID:    t.ID,
			Name:  t.Name,
			Token: t.Token,
		})
	}
	return result, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) CreateAccessToken(opt gittypes.CreateAccessTokenOption) (*gittypes.AccessToken, *gittypes.Response, error) {
	giteaOpt := gitea.CreateAccessTokenOption{
		Name: opt.Name,
	}
	token, resp, err := r.giteaClient.CreateAccessToken(giteaOpt)
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.AccessToken{
		ID:    token.ID,
		Name:  token.Name,
		Token: token.Token,
	}, &gittypes.Response{Response: resp.Response}, nil
}
