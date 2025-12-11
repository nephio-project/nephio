package git

import (
	"context"

	gittypes "github.com/nephio-project/nephio/controllers/pkg/git/types"
)

// ProviderType represents the type of git provider
type ProviderType string

const (
	// ProviderGitea represents Gitea git provider
	ProviderGitea ProviderType = "gitea"
	// ProviderGitHub represents GitHub git provider
	ProviderGitHub ProviderType = "github"
	// ProviderGitLab represents GitLab git provider
	ProviderGitLab ProviderType = "gitlab"
)

//go:generate mockery --name=Client --output=. --outpkg=git
type Client interface {
	Start(ctx context.Context)
	IsInitialized() bool
	Get() any
	GetMyUserInfo() (*gittypes.User, *gittypes.Response, error)
	DeleteRepo(owner string, repo string) (*gittypes.Response, error)
	GetRepo(userName string, repoCRName string) (*gittypes.Repository, *gittypes.Response, error)
	CreateRepo(createRepoOption gittypes.CreateRepoOption) (*gittypes.Repository, *gittypes.Response, error)
	EditRepo(userName string, repoCRName string, editRepoOption gittypes.EditRepoOption) (*gittypes.Repository, *gittypes.Response, error)
	DeleteAccessToken(value interface{}) (*gittypes.Response, error)
	ListAccessTokens(opts gittypes.ListAccessTokensOptions) ([]*gittypes.AccessToken, *gittypes.Response, error)
	CreateAccessToken(opt gittypes.CreateAccessTokenOption) (*gittypes.AccessToken, *gittypes.Response, error)
}
