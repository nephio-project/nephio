# GitHub Client Implementation

This package provides a GitHub client implementation for Nephio controllers using GitHub App authentication.

## Authentication Method

This client uses **GitHub App authentication** instead of personal access tokens for improved security and flexibility:

1. **JWT (JSON Web Token)**: Generated using the GitHub App's private key and App ID
2. **Installation Token**: Short-lived access token obtained by exchanging the JWT

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
  app_id: "123456"
  installation_id: "12345678"
  private_key: |
    -----BEGIN RSA PRIVATE KEY-----
    <your-private-key-content>
    -----END RSA PRIVATE KEY-----
```

**Important**: Both `app_id` and `installation_id` must be quoted as strings in YAML.

Or using kubectl:

```bash
kubectl create secret generic github-user-secret \
  --from-literal=app_id="123456" \
  --from-literal=installation_id="12345678" \
  --from-file=private_key=path/to/your-app.private-key.pem \
  -n default
```

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

1. **Initialization**: The client retrieves GitHub App credentials from the Kubernetes secret
2. **JWT Generation**: Creates a JWT signed with the private key, valid for 10 minutes
3. **Installation Token**: Exchanges the JWT for an installation access token via GitHub API
4. **API Operations**: Uses the installation token for all GitHub API operations
5. **Token Generation**: `CreateAccessToken()` generates new installation tokens on-demand (valid 1 hour)
6. **Token Lifecycle**: Installation tokens expire automatically after 1 hour and cannot be explicitly deleted

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

## Security Benefits

- **Scoped Permissions**: GitHub Apps can be granted specific, granular permissions
- **Installation-level**: Works at the organization/repository level, not tied to a user
- **Short-lived Tokens**: Installation tokens expire after 1 hour (vs. PATs which are long-lived)
- **Auditable**: All actions are logged as performed by the GitHub App
- **No User Context**: Doesn't require a specific user's PAT, making it more maintainable

## Differences from Personal Access Tokens (PAT)

| Feature | GitHub App | Personal Access Token |
|---------|-----------|----------------------|
| Scope | Organization/Installation | User account |
| Lifetime | 1 hour (on-demand generation) | No expiration (manual rotation) |
| Permissions | Fine-grained | Broad account access |
| Audit Trail | App-attributed actions | User-attributed actions |
| Rate Limits | Higher (5000 req/hr) | Lower (5000 req/hr shared) |

## Troubleshooting

### "Cannot get secret" error
- Ensure the secret exists in the `default` namespace
- Verify the secret name matches `GIT_SECRET_NAME` environment variable (default: `github-user-secret`)

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
