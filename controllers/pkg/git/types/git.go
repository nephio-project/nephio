package types

import (
	"net/http"
)

// Response represents the  response
type Response struct {
	*http.Response
}
type User struct {
	// the user's username
	UserName string `json:"login"`
}
type Repository struct {
	CloneURL string `json:"clone_url"`
}
type TrustModel string

// CreateRepoOption options when creating repository
type CreateRepoOption struct {
	// Name of the repository to create
	Name string `json:"name"`
	// Description of the repository to create
	Description string `json:"description"`
	// Whether the repository is private
	Private bool `json:"private"`
	// Issue Label set to use
	IssueLabels string `json:"issue_labels"`
	// Whether the repository should be auto-intialized?
	AutoInit bool `json:"auto_init"`
	// Whether the repository is template
	Template bool `json:"template"`
	// Gitignores to use
	Gitignores string `json:"gitignores"`
	// License to use
	License string `json:"license"`
	// Readme of the repository to create
	Readme string `json:"readme"`
	// DefaultBranch of the repository (used when initializes and in template)
	DefaultBranch string `json:"default_branch"`
	// TrustModel of the repository
	TrustModel TrustModel `json:"trust_model"`
}

// MergeStyle is used specify how a pull is merged
type MergeStyle string

// InternalTracker represents settings for internal tracker
type InternalTracker struct {
	// Enable time tracking (Built-in issue tracker)
	EnableTimeTracker bool `json:"enable_time_tracker"`
	// Let only contributors track time (Built-in issue tracker)
	AllowOnlyContributorsToTrackTime bool `json:"allow_only_contributors_to_track_time"`
	// Enable dependencies for issues and pull requests (Built-in issue tracker)
	EnableIssueDependencies bool `json:"enable_issue_dependencies"`
}

// ExternalTracker represents settings for external tracker
type ExternalTracker struct {
	// URL of external issue tracker.
	ExternalTrackerURL string `json:"external_tracker_url"`
	// External Issue Tracker URL Format. Use the placeholders {user}, {repo} and {index} for the username, repository name and issue index.
	ExternalTrackerFormat string `json:"external_tracker_format"`
	// External Issue Tracker Number Format, either `numeric` or `alphanumeric`
	ExternalTrackerStyle string `json:"external_tracker_style"`
}

// ExternalWiki represents setting for external wiki
type ExternalWiki struct {
	// URL of external wiki.
	ExternalWikiURL string `json:"external_wiki_url"`
}

// EditRepoOption options when editing a repository's properties
type EditRepoOption struct {
	// name of the repository
	Name *string `json:"name,omitempty"`
	// a short description of the repository.
	Description *string `json:"description,omitempty"`
	// a URL with more information about the repository.
	Website *string `json:"website,omitempty"`
	// either `true` to make the repository private or `false` to make it public.
	// Note: you will get a 422 error if the organization restricts changing repository visibility to organization
	// owners and a non-owner tries to change the value of private.
	Private *bool `json:"private,omitempty"`
	// either `true` to make this repository a template or `false` to make it a normal repository
	Template *bool `json:"template,omitempty"`
	// either `true` to enable issues for this repository or `false` to disable them.
	HasIssues *bool `json:"has_issues,omitempty"`
	// set this structure to configure internal issue tracker (requires has_issues)
	InternalTracker *InternalTracker `json:"internal_tracker,omitempty"`
	// set this structure to use external issue tracker (requires has_issues)
	ExternalTracker *ExternalTracker `json:"external_tracker,omitempty"`
	// either `true` to enable the wiki for this repository or `false` to disable it.
	HasWiki *bool `json:"has_wiki,omitempty"`
	// set this structure to use external wiki instead of internal (requires has_wiki)
	ExternalWiki *ExternalWiki `json:"external_wiki,omitempty"`
	// sets the default branch for this repository.
	DefaultBranch *string `json:"default_branch,omitempty"`
	// either `true` to allow pull requests, or `false` to prevent pull request.
	HasPullRequests *bool `json:"has_pull_requests,omitempty"`
	// either `true` to enable project unit, or `false` to disable them.
	HasProjects *bool `json:"has_projects,omitempty"`
	// either `true` to ignore whitespace for conflicts, or `false` to not ignore whitespace. `has_pull_requests` must be `true`.
	IgnoreWhitespaceConflicts *bool `json:"ignore_whitespace_conflicts,omitempty"`
	// either `true` to allow merging pull requests with a merge commit, or `false` to prevent merging pull requests with merge commits. `has_pull_requests` must be `true`.
	AllowMerge *bool `json:"allow_merge_commits,omitempty"`
	// either `true` to allow rebase-merging pull requests, or `false` to prevent rebase-merging. `has_pull_requests` must be `true`.
	AllowRebase *bool `json:"allow_rebase,omitempty"`
	// either `true` to allow rebase with explicit merge commits (--no-ff), or `false` to prevent rebase with explicit merge commits. `has_pull_requests` must be `true`.
	AllowRebaseMerge *bool `json:"allow_rebase_explicit,omitempty"`
	// either `true` to allow squash-merging pull requests, or `false` to prevent squash-merging. `has_pull_requests` must be `true`.
	AllowSquash *bool `json:"allow_squash_merge,omitempty"`
	// set to `true` to archive this repository.
	Archived *bool `json:"archived,omitempty"`
	// set to a string like `8h30m0s` to set the mirror interval time
	MirrorInterval *string `json:"mirror_interval,omitempty"`
	// either `true` to allow mark pr as merged manually, or `false` to prevent it. `has_pull_requests` must be `true`.
	AllowManualMerge *bool `json:"allow_manual_merge,omitempty"`
	// either `true` to enable AutodetectManualMerge, or `false` to prevent it. `has_pull_requests` must be `true`, Note: In some special cases, misjudgments can occur.
	AutodetectManualMerge *bool `json:"autodetect_manual_merge,omitempty"`
	// set to a merge style to be used by this repository: "merge", "rebase", "rebase-merge", or "squash". `has_pull_requests` must be `true`.
	DefaultMergeStyle *MergeStyle `json:"default_merge_style,omitempty"`
	// set to `true` to archive this repository.
}

type AccessTokenScope string

// AccessToken represents an API access token.
type AccessToken struct {
	ID             int64              `json:"id"`
	Name           string             `json:"name"`
	Token          string             `json:"sha1"`
	TokenLastEight string             `json:"token_last_eight"`
	Scopes         []AccessTokenScope `json:"scopes"`
}

// CreateAccessTokenOption options when create access token
type CreateAccessTokenOption struct {
	Name   string             `json:"name"`
	Scopes []AccessTokenScope `json:"scopes"`
}

// ListAccessTokensOptions options for listing a users's access tokens
type ListAccessTokensOptions struct {
	ListOptions
}

// ListOptions options for using  API pagination
type ListOptions struct {
	// Setting Page to -1 disables pagination on endpoints that support it.
	// Page numbering starts at 1.
	Page int
	// The default value depends on the server config DEFAULT_PAGING_NUM
	// The highest valid value depends on the server config MAX_RESPONSE_ITEMS
	PageSize int
}

const (
	AccessTokenScopeAll AccessTokenScope = "all"

	AccessTokenScopeRepo       AccessTokenScope = "repo"
	AccessTokenScopeRepoStatus AccessTokenScope = "repo:status"
	AccessTokenScopePublicRepo AccessTokenScope = "public_repo"

	AccessTokenScopeAdminOrg AccessTokenScope = "admin:org"
	AccessTokenScopeWriteOrg AccessTokenScope = "write:org"
	AccessTokenScopeReadOrg  AccessTokenScope = "read:org"

	AccessTokenScopeAdminPublicKey AccessTokenScope = "admin:public_key"
	AccessTokenScopeWritePublicKey AccessTokenScope = "write:public_key"
	AccessTokenScopeReadPublicKey  AccessTokenScope = "read:public_key"

	AccessTokenScopeAdminRepoHook AccessTokenScope = "admin:repo_hook" // #nosec G101 -- OAuth scope name, not a credential
	AccessTokenScopeWriteRepoHook AccessTokenScope = "write:repo_hook" // #nosec G101 -- OAuth scope name, not a credential
	AccessTokenScopeReadRepoHook  AccessTokenScope = "read:repo_hook"  // #nosec G101 -- OAuth scope name, not a credential

	AccessTokenScopeAdminOrgHook AccessTokenScope = "admin:org_hook" // #nosec G101 -- OAuth scope name, not a credential

	AccessTokenScopeAdminUserHook AccessTokenScope = "admin:user_hook"

	AccessTokenScopeNotification AccessTokenScope = "notification"

	AccessTokenScopeUser       AccessTokenScope = "user"
	AccessTokenScopeReadUser   AccessTokenScope = "read:user"
	AccessTokenScopeUserEmail  AccessTokenScope = "user:email"
	AccessTokenScopeUserFollow AccessTokenScope = "user:follow"

	AccessTokenScopeDeleteRepo AccessTokenScope = "delete_repo"

	AccessTokenScopePackage       AccessTokenScope = "package"
	AccessTokenScopeWritePackage  AccessTokenScope = "write:package"
	AccessTokenScopeReadPackage   AccessTokenScope = "read:package"
	AccessTokenScopeDeletePackage AccessTokenScope = "delete:package"

	AccessTokenScopeAdminGPGKey AccessTokenScope = "admin:gpg_key" // #nosec G101 -- OAuth scope name, not a credential
	AccessTokenScopeWriteGPGKey AccessTokenScope = "write:gpg_key" // #nosec G101 -- OAuth scope name, not a credential
	AccessTokenScopeReadGPGKey  AccessTokenScope = "read:gpg_key"  // #nosec G101 -- OAuth scope name, not a credential

	AccessTokenScopeAdminApplication AccessTokenScope = "admin:application" // #nosec G101 -- OAuth scope name, not a credential
	AccessTokenScopeWriteApplication AccessTokenScope = "write:application" // #nosec G101 -- OAuth scope name, not a credential
	AccessTokenScopeReadApplication  AccessTokenScope = "read:application"  // #nosec G101 -- OAuth scope name, not a credential

	AccessTokenScopeSudo AccessTokenScope = "sudo"
)
