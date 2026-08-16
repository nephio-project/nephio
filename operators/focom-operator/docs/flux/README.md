# Flux Deployment Documentation

Documentation for deploying the FOCOM operator with Flux CD instead of ConfigSync.

## Quick Links

- **[quick-start.md](quick-start.md)** - TL;DR deployment guide
- **[deployment-guide.md](deployment-guide.md)** - Complete Flux configuration documentation
- **[makefile-usage.md](makefile-usage.md)** - Makefile targets for Flux deployment
- **[testing-guide.md](testing-guide.md)** - Step-by-step testing procedures
- **[webhook-setup.md](webhook-setup.md)** - Configure webhooks for instant sync
- **[what-gets-removed.md](what-gets-removed.md)** - What `make undeploy-flux` removes

## Quick Start

### Deploy with Pre-built Private Registry Image

```bash
# Set up Docker registry authentication
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token

# Deploy with Flux
make deploy-with-flux IMG=ghcr.io/your-org/focom-operator:latest IMAGE_PULL_SECRET=ghcr-secret
```

### Deploy with Custom Image

```bash
make deploy-with-flux IMG=<registry>/<image>:<tag>
```

### Deploy with Flux + Webhook (Instant Sync)

```bash
make deploy-with-flux-webhook IMG=ghcr.io/your-org/focom-operator:latest IMAGE_PULL_SECRET=ghcr-secret
```

### Undeploy

```bash
make undeploy-flux
```

## What is Flux?

Flux CD is a CNCF graduated GitOps tool that syncs Git repositories to Kubernetes clusters. It provides an alternative to ConfigSync with additional features:

- ✅ Webhook support for instant sync (<1s latency)
- ✅ Polling for reliable fallback (15s interval by default)
- ✅ Better observability and notifications
- ✅ CNCF graduated (vendor-neutral)
- ✅ Multi-repository support
- ✅ Rich CLI tooling

**Sync Mechanisms:**
- **Polling (default):** Checks Git every 15 seconds - reliable but has latency
- **Webhooks (optional):** Instant sync when Git changes - requires configuration
- **Both together (recommended):** Webhooks for speed + polling for reliability

## When to Use Flux

**Use Flux if:**
- You need instant sync via webhooks
- You want better observability
- You prefer CNCF graduated projects
- You need multi-repository support

**Use ConfigSync if:**
- You're following Nephio reference architecture
- You want the simplest configuration
- You don't need webhook support

## Configuration Files

The Flux configuration files are in `config/flux/`:

- `focom-resources-gitrepository.yaml` - Git source definition
- `focom-resources-kustomization.yaml` - What to apply from Git
- `focom-resources-receiver.yaml` - Webhook receiver (optional)
- `kustomization.yaml` - Kustomize config

## See Also

- [Main Deployment Guide](../DEPLOYMENT.md) - All deployment methods
- [kpt Deployment](../kpt/) - kpt-based deployment
- [Architecture](../ARCHITECTURE.md) - System architecture
