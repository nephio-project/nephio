# Flux Configuration Files

This directory contains the Flux CD configuration for syncing FOCOM resources from Git to Kubernetes.

## Files

- `focom-resources-gitrepository.yaml` - Git source definition
- `focom-resources-kustomization.yaml` - What to apply from Git  
- `focom-resources-receiver.yaml` - Webhook receiver (optional)
- `kustomization.yaml` - Kustomize configuration

## Documentation

Complete documentation is available in `docs/flux/`:

- **[docs/flux/README.md](../../docs/flux/README.md)** - Documentation index
- **[docs/flux/quick-start.md](../../docs/flux/quick-start.md)** - Quick start guide
- **[docs/flux/deployment-guide.md](../../docs/flux/deployment-guide.md)** - Complete deployment guide
- **[docs/flux/makefile-usage.md](../../docs/flux/makefile-usage.md)** - Makefile targets
- **[docs/flux/webhook-setup.md](../../docs/flux/webhook-setup.md)** - Webhook configuration
- **[docs/flux/testing-guide.md](../../docs/flux/testing-guide.md)** - Testing procedures

## Quick Deployment

```bash
# Deploy with Flux
make deploy-with-flux IMG=<registry>/<image>:<tag>

# Deploy with Flux + Webhook
make deploy-with-flux-webhook IMG=<registry>/<image>:<tag>

# Undeploy
make undeploy-flux
```

See [docs/flux/](../../docs/flux/) for complete documentation.
