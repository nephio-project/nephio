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

package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v66/github"
	"github.com/nephio-project/nephio/controllers/pkg/git"
	gittypes "github.com/nephio-project/nephio/controllers/pkg/git/types"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var lock = &sync.Mutex{}

const defaultGitHubAPI = "https://api.github.com"

type gc struct {
	client       resource.APIPatchingApplicator
	githubClient *github.Client
	// Where installation tokens are minted and what mints them, as fields so a
	// test can redirect them without touching process-wide http.DefaultClient.
	tokenEndpoint string
	tokenClient   *http.Client
	appID         string
	installID     string
	privateKey    *rsa.PrivateKey
	l             logr.Logger
}

var singleInstance *gc

func GetClient(ctx context.Context, client resource.APIPatchingApplicator) (git.Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("failed creating github client, value of ctx cannot be nil")
	}

	if client.Client == nil {
		return nil, fmt.Errorf("failed creating github client, value of client.Client cannot be nil")
	}
	// check if an instance is created using check-lock-check pattern implementation
	if singleInstance == nil {
		// Create a lock
		lock.Lock()
		defer lock.Unlock()
		// Check instance is still null as another thread of execution may have initialized it before the lock was acquired.
		if singleInstance == nil {
			singleInstance = &gc{client: client}
			log.FromContext(ctx).Info("GitHub Client Instance created now.")
			go singleInstance.Start(ctx)
		} else {
			log.FromContext(ctx).Info("GitHub Client Instance already created.")
		}
	} else {
		log.FromContext(ctx).Info("GitHub Client Instance already created.")
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
			time.Sleep(5 * time.Second)

			namespace := "default"
			secretName := "github-user-secret" // #nosec G101 -- Kubernetes Secret name, not a hardcoded credential
			if gitSecretName, ok := os.LookupEnv("GIT_SECRET_NAME"); ok {
				secretName = gitSecretName
			}

			// get secret that contains GitHub App credentials
			secret := &corev1.Secret{}
			if err := r.client.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      secretName,
			},
				secret); err != nil {
				r.l.Error(err, "Cannot get secret, please follow README and create the github secret")
				break
			}

			// Extract GitHub App credentials from secret
			personalAccessToken := string(secret.Data["personal_access_token"])
			appID := string(secret.Data["app_id"])
			installID := string(secret.Data["installation_id"])
			privateKeyPEM := string(secret.Data["private_key"])

			r.l.Info("extracted credentials from secret", "appID", appID, "installID", installID, "privateKeyLength", len(privateKeyPEM))

			if appID == "" || installID == "" || privateKeyPEM == "" || personalAccessToken == "" {
				r.l.Error(fmt.Errorf("missing credentials in secret"), "secret must contain app_id, installation_id, personal_access_token and private_key")
				break
			}

			// Parse the private key
			privateKey, err := parsePrivateKey(privateKeyPEM)
			if err != nil {
				r.l.Error(err, "Failed to parse private key")
				break
			}
			// Store credentials for installation token refresh
			r.appID = appID
			r.installID = installID
			r.privateKey = privateKey

			// Create GitHub client with personal access token
			githubClient := github.NewClient(nil).WithAuthToken(personalAccessToken)
			r.githubClient = githubClient
			r.l.Info("github init done")
			return
		}
	}
}

// parsePrivateKey parses a PEM-encoded RSA private key
func parsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// generateJWT creates a JWT token for GitHub App authentication
func generateJWT(appID string, privateKey *rsa.PrivateKey) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// getInstallationToken exchanges a JWT for an installation access token
func (r *gc) getInstallationToken(jwtToken, installID string) (string, error) {
	endpoint := r.tokenEndpoint
	if endpoint == "" {
		endpoint = defaultGitHubAPI
	}
	httpClient := r.tokenClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	url := fmt.Sprintf("%s/app/installations/%s/access_tokens", strings.TrimRight(endpoint, "/"), installID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Token, nil
}

// createInstallationToken mints a new installation access token and returns it.
// It leaves r.githubClient alone: that client carries the personal access token
// the user and repository calls need, and the authenticated-user endpoint does
// not accept an installation token.
func (r *gc) createInstallationToken() (string, error) {
	jwtToken, err := generateJWT(r.appID, r.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	installToken, err := r.getInstallationToken(jwtToken, r.installID)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}

	return installToken, nil
}

func (r *gc) IsInitialized() bool {
	return r.githubClient != nil
}

func (r *gc) Get() any {
	return r.githubClient
}

func (r *gc) GetMyUserInfo() (*gittypes.User, *gittypes.Response, error) {
	userInfo, resp, err := r.githubClient.Users.Get(context.Background(), "")
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.User{
		UserName: *userInfo.Login,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) DeleteRepo(owner string, repo string) (*gittypes.Response, error) {
	resp, err := r.githubClient.Repositories.Delete(context.Background(), owner, repo)
	if err != nil {
		return &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) GetRepo(userName string, repoCRName string) (*gittypes.Repository, *gittypes.Response, error) {
	repo, resp, err := r.githubClient.Repositories.Get(context.Background(), userName, repoCRName)
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Repository{
		CloneURL: *repo.CloneURL,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) CreateRepo(createRepoOption gittypes.CreateRepoOption) (*gittypes.Repository, *gittypes.Response, error) {
	repo := &github.Repository{
		Name:              github.String(createRepoOption.Name),
		Description:       github.String(createRepoOption.Description),
		Private:           github.Bool(createRepoOption.Private),
		AutoInit:          github.Bool(createRepoOption.AutoInit),
		GitignoreTemplate: github.String(createRepoOption.Gitignores),
		LicenseTemplate:   github.String(createRepoOption.License),
	}

	// Set default branch if provided
	if createRepoOption.DefaultBranch != "" {
		repo.DefaultBranch = github.String(createRepoOption.DefaultBranch)
	}

	createdRepo, resp, err := r.githubClient.Repositories.Create(context.Background(), "", repo)
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Repository{
		CloneURL: *createdRepo.CloneURL,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) EditRepo(userName string, repoCRName string, editRepoOption gittypes.EditRepoOption) (*gittypes.Repository, *gittypes.Response, error) {
	repo := &github.Repository{}

	if editRepoOption.Name != nil {
		repo.Name = editRepoOption.Name
	}
	if editRepoOption.Description != nil {
		repo.Description = editRepoOption.Description
	}
	if editRepoOption.Private != nil {
		repo.Private = editRepoOption.Private
	}

	editedRepo, resp, err := r.githubClient.Repositories.Edit(context.Background(), userName, repoCRName, repo)
	if err != nil {
		return nil, &gittypes.Response{Response: resp.Response}, err
	}
	return &gittypes.Repository{
		CloneURL: *editedRepo.CloneURL,
	}, &gittypes.Response{Response: resp.Response}, nil
}

func (r *gc) DeleteAccessToken(value interface{}) (*gittypes.Response, error) {
	// GitHub installation tokens expire automatically after 1 hour
	// They cannot be explicitly deleted via API
	// This is a dummy implementation to satisfy the interface
	return &gittypes.Response{}, nil
}

func (r *gc) ListAccessTokens(opts gittypes.ListAccessTokensOptions) ([]*gittypes.AccessToken, *gittypes.Response, error) {
	// GitHub installation tokens are ephemeral and expire after 1 hour
	// There is no API to list them, as they are generated on-demand
	// This is a dummy implementation returning an empty list
	return []*gittypes.AccessToken{}, &gittypes.Response{}, nil
}

func (r *gc) CreateAccessToken(opt gittypes.CreateAccessTokenOption) (*gittypes.AccessToken, *gittypes.Response, error) {
	installToken, err := r.createInstallationToken()
	if err != nil {
		return nil, nil, err
	}

	// Return the installation token in AccessToken format.
	// GitHub installation tokens don't have IDs, so we use 0.
	// The token name is provided by the caller.
	return &gittypes.AccessToken{
		ID:    0, // Installation tokens don't have IDs
		Name:  opt.Name,
		Token: installToken,
	}, &gittypes.Response{}, nil
}
