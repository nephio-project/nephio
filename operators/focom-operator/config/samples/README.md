# Configuration Samples

This directory contains sample configuration files and helper scripts for setting up the FOCOM operator.

## Files

### focom-porch-repository.yaml

**⚠️ CRITICAL PREREQUISITE** - This must be created BEFORE deploying the FOCOM operator!

Porch Repository CR that configures Porch to use the focom-resources Git repository. This is the main Git repository where all FOCOM resources (OClouds, TemplateInfos, FocomProvisioningRequests) will be stored.

**Usage:**
```bash
# 1. First, create the Git repository in your Git server (Gitea, GitHub, GitLab)
#    Example: http://172.18.0.200:3000/nephio/focom-resources.git

# 2. Edit the file to match your Git repository URL
vim config/samples/focom-porch-repository.yaml

# 3. Apply to cluster BEFORE deploying FOCOM operator
kubectl apply -f config/samples/focom-porch-repository.yaml

# 4. Verify repository is ready
kubectl get repository focom-resources -n default
# Should show READY=True
```

**Required Configuration:**
- `spec.git.repo`: Your Git repository URL
- `spec.git.branch`: Branch to use (typically `main`)
- `spec.git.secretRef.name`: Reference to gitea-secret (if authentication required)

**Important Notes:**
- ⚠️ The Git repository must exist before applying this CR
- ⚠️ This must be applied before deploying the FOCOM operator
- ⚠️ The operator will fail to start if this repository is not ready
- The repository will store all approved FOCOM resources as Kpt packages
- ConfigSync will sync resources from this repository to Kubernetes CRs

### gitea-secret.yaml

Template for creating a Kubernetes secret with Git credentials for Porch and ConfigSync.

**Usage:**
```bash
# Copy and edit the template
cp gitea-secret.yaml /tmp/gitea-secret.yaml
# Edit with your base64-encoded credentials
vim /tmp/gitea-secret.yaml
# Apply to cluster
kubectl apply -f /tmp/gitea-secret.yaml
```

**Required Fields:**
- `username`: Base64-encoded Git username
- `password`: Base64-encoded Git password
- `token`: Base64-encoded Git access token

**Secret Type:** `kubernetes.io/basic-auth`

### create-gitea-secret.sh

Helper script to create the gitea-secret interactively.

**Usage:**
```bash
./create-gitea-secret.sh
```

The script will prompt for:
- Username (default: nephio)
- Password
- Access token

The script automatically:
- Base64-encodes the credentials
- Creates the secret in the `default` namespace
- Validates the secret was created successfully

**Prerequisites:**
- `kubectl` configured and connected to your cluster
- Git access token from your Git server (Gitea, GitHub, GitLab, etc.)

## Creating Git Access Token

### Gitea

1. Log in to Gitea
2. Go to Settings → Applications → Generate New Token
3. Give it a name (e.g., "porch-access")
4. Select scopes: `repo` (full repository access)
5. Generate token and copy it

### GitHub

1. Go to Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token
3. Select scopes: `repo` (full repository access)
4. Generate token and copy it

### GitLab

1. Go to Preferences → Access Tokens
2. Create personal access token
3. Select scopes: `read_repository`, `write_repository`
4. Create token and copy it

## Verification

After creating the secret, verify it exists:

```bash
# Check secret exists
kubectl get secret gitea-secret -n default

# Verify secret type and fields
kubectl get secret gitea-secret -n default -o yaml

# Check that all required fields are present
kubectl get secret gitea-secret -n default -o jsonpath='{.data}' | jq 'keys'
# Should show: ["password", "token", "username"]
```

## Troubleshooting

### Secret not found

If the secret doesn't exist:
```bash
# Create using the helper script
./create-gitea-secret.sh

# Or create manually
kubectl create secret generic gitea-secret \
  -n default \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=<username> \
  --from-literal=password=<password> \
  --from-literal=bearerToken=<gitea-access-token>
```

### Authentication failures

If Porch or ConfigSync cannot authenticate:
1. Verify the token is valid and not expired
2. Verify the token has correct permissions (repo access)
3. Verify the secret has all three fields (`username`, `password`, `bearerToken`)
4. Verify the secret type is `kubernetes.io/basic-auth`
5. Restart Porch pods after updating the secret

### Wrong namespace

The secret must be in the `default` namespace for Porch, and will be copied to `config-management-system` for ConfigSync automatically during deployment.

## Additional Resources

- [Porch Setup Guide](../../docs/porch-setup.md)
- [Deployment Guide](../../docs/DEPLOYMENT.md)
- [Troubleshooting Guide](../../docs/TROUBLESHOOTING.md)
