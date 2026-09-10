# GitHub Client Implementation

This package provides a GitHub client implementation for Nephio controllers using GitHub App authentication.

## Authentication Method

This client uses a **hybrid authentication approach**:

1. **Personal Access Token (PAT)**: Used for regular GitHub API operations (repository CRUD, user info, etc.)
2. **GitHub App Installation Tokens**: Generated on-demand via `CreateAccessToken()` for time-limited access (valid 1 hour)

The client requires GitHub App credentials (App ID, Installation ID, Private Key) to generate installation tokens, but uses a PAT for standard operations.

## Prerequisites

### 1. Create a GitHub App

1. Go to your GitHub account/organization Settings → Developer settings → GitHub Apps
2. Click "New GitHub App"
3. Configure the app:
   - **Name**: Choose a meaningful name (e.g., "Nephio Controller")
   - **Homepage URL**: Your organization's URL
   - **Webhook**: Can be disabled for basic operations
   - **Permissions**: Grant necessary repository permissions:
     - Contents: Read & Write
     - Administration: Read & Write (if you need to create/delete repos)
     - Metadata: Read (required)
4. Click "Create GitHub App"
5. Note the **App ID** (you'll need this)
6. Generate and download a **private key** (PEM format)

### 2. Install the GitHub App

1. Go to the app's settings
2. Click "Install App" in the left sidebar
3. Choose the account/organization where you want to install it
4. Select repositories (all or specific ones)
5. Note the **Installation ID** from the URL after installation (e.g., `https://github.com/settings/installations/12345678` - the number is your installation ID)

### 3. Create Kubernetes Secret

Create a secret containing your GitHub App credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-user-secret
  namespace: default
type: Opaque
stringData:
  personal_access_token: "ghp_your_personal_access_token_here"
  app_id: "123456"
  installation_id: "12345678"
  private_key: |
    -----BEGIN RSA PRIVATE KEY-----
    <your-private-key-content>
    -----END RSA PRIVATE KEY-----
```

**Important**: 
- Both `app_id` and `installation_id` must be quoted as strings in YAML
- `personal_access_token` is your GitHub Personal Access Token (PAT) for regular operations
- `private_key` is the GitHub App's private key for generating installation tokens

Or using kubectl:

```bash
kubectl create secret generic github-user-secret \
  --from-literal=personal_access_token="ghp_your_token_here" \
  --from-literal=app_id="123456" \
  --from-literal=installation_id="12345678" \
  --from-file=private_key=path/to/your-app.private-key.pem \
  -n default
```

### Generating a Personal Access Token (PAT)

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token" → "Generate new token (classic)"
3. Set a note (e.g., "Nephio Controller")
4. Select scopes:
   - `repo` (Full control of private repositories)
   - `admin:org` → `read:org` (if working with organization repos)
5. Click "Generate token" and copy the token (starts with `ghp_`)

## Environment Variables

The client uses the following environment variables:

- `GIT_SECRET_NAME`: Name of the secret containing GitHub App credentials (default: `github-user-secret`)

**Note**: The client looks for the secret in the `default` namespace (hardcoded).

## Usage

```go
import (
    githubclient "github.com/nephio-project/nephio/controllers/pkg/git/github"
    "github.com/nephio-project/nephio/controllers/pkg/resource"
)

// Initialize the client
client := resource.NewAPIPatchingApplicator(k8sClient)
githubClient, err := githubclient.GetClient(ctx, client)
if err != nil {
    log.Fatal(err)
}

// Wait for initialization
for !githubClient.IsInitialized() {
    time.Sleep(1 * time.Second)
}

// Create a repository
repo, resp, err := githubClient.CreateRepo(gittypes.CreateRepoOption{
    Name: "my-repo",
    Description: "My repository",
    Private: true,
    AutoInit: true,
})

// Generate an installation access token (like PAT)
token, resp, err := githubClient.CreateAccessToken(gittypes.CreateAccessTokenOption{
    Name: "my-token",
})
if err != nil {
    log.Fatal(err)
}
// token.Token contains the installation token (valid for 1 hour)
```

## How It Works

1. **Initialization**: The client retrieves credentials from the Kubernetes secret
   - Personal Access Token for GitHub API operations
   - GitHub App credentials (App ID, Installation ID, Private Key) for token generation
2. **Regular Operations**: Uses the Personal Access Token for all standard GitHub API calls (create repo, get user info, etc.)
3. **Installation Token Generation**: When `CreateAccessToken()` is called:
   - Generates a JWT signed with the GitHub App's private key (valid 10 minutes)
   - Exchanges the JWT for an installation access token via GitHub API
   - Returns the installation token (valid 1 hour)
4. **Token Lifecycle**: Installation tokens expire automatically after 1 hour and cannot be explicitly deleted

## Token Management

### Creating Tokens (Like PAT)
The `CreateAccessToken()` method generates GitHub installation tokens on-demand:
- Each call creates a new installation token valid for 1 hour
- Returns an `AccessToken` object containing the token string
- Token name is stored but not used by GitHub (for interface compatibility)
- No token ID is assigned (returns 0)

### Listing Tokens
`ListAccessTokens()` returns an empty list - installation tokens are ephemeral and cannot be listed.

### Deleting Tokens  
`DeleteAccessToken()` is a no-op - installation tokens expire automatically after 1 hour and cannot be explicitly deleted.

These methods satisfy the `git.Client` interface while working within GitHub App's token model.

## Security Considerations

**Regular Operations (PAT-based)**:
- Uses Personal Access Token tied to a user account
- Long-lived token (manual rotation recommended)
- Requires appropriate scopes (`repo`, `read:org`)

**Installation Tokens (GitHub App-based)**:
- **Scoped Permissions**: GitHub Apps can be granted specific, granular permissions
- **Installation-level**: Works at the organization/repository level
- **Short-lived**: Installation tokens expire after 1 hour
- **Auditable**: Actions are logged as performed by the GitHub App
- **On-demand Generation**: Tokens created only when needed

## Token Comparison

| Feature | Personal Access Token (Regular Ops) | Installation Token (CreateAccessToken) |
|---------|-------------------------------------|----------------------------------------|
| Use Case | Repository CRUD, User Info | Time-limited delegated access |
| Lifetime | No expiration (manual rotation) | 1 hour (auto-expires) |
| Scope | User account permissions | GitHub App installation permissions |
| Generation | Manual via GitHub UI | Programmatic on-demand |
| Audit Trail | User-attributed actions | App-attributed actions |
| Rate Limits | 5000 req/hr (per user) | 5000 req/hr (per installation) |

## Troubleshooting

### "Cannot get secret" error
- Ensure the secret exists in the `default` namespace
- Verify the secret name matches `GIT_SECRET_NAME` environment variable (default: `github-user-secret`)

### "401 Bad credentials" error
- Verify the `personal_access_token` is correct and not expired
- Ensure the PAT has the required scopes (`repo`, `read:org` if needed)
- Check that the token hasn't been revoked in GitHub settings

### "Failed to parse private key" error
- Ensure the private key is in PEM format
- Check that the entire key (including BEGIN/END markers) is in the secret

### "Failed to get installation token" error
- Verify the App ID and Installation ID are correct
- Ensure the GitHub App has necessary permissions
- Check that the GitHub App is installed on the target repositories

### Rate Limiting
- GitHub Apps have a rate limit of 5000 requests per hour per installation
- Monitor the `X-RateLimit-*` headers in API responses
