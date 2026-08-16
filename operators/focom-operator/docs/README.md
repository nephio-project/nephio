# FOCOM Operator Documentation

Complete documentation for the FOCOM (Fabric O-Cloud Manager) Operator.

## Getting Started

### Quick Start
- **[QUICK_START.md](QUICK_START.md)** - 10-minute quick start guide

### Deployment
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Complete deployment guide with multiple methods
  - Method 1: Standard deployment with `make deploy` (ConfigSync)
  - Method 2: kpt-based deployment
  - Method 3: Direct kubectl deployment
- **[flux/](flux/)** - Flux CD deployment documentation
  - [Quick Start](flux/quick-start.md)
  - [Deployment Guide](flux/deployment-guide.md)
  - [Makefile Usage](flux/makefile-usage.md)
  - [Webhook Setup](flux/webhook-setup.md)
  - [Testing Guide](flux/testing-guide.md)
  - [What Gets Removed](flux/what-gets-removed.md)
- **[kpt/](kpt/)** - kpt-specific deployment documentation
  - [Quick Reference](kpt/quick-reference.md)
  - [Deployment Checklist](kpt/checklist.md)
  - [Undeployment Guide](kpt/undeployment.md)

### Configuration
- **[PORCH_SETUP.md](PORCH_SETUP.md)** - Porch setup and configuration

## Architecture & Design

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture with diagrams
  - Component overview
  - Resource model and relationships
  - Data flow diagrams
  - Integration points
  - Technology stack
  - User workflows

- **[USER_GUIDE.md](USER_GUIDE.md)** - User guide for FOCOM NBI API
  - Understanding FOCOM resources
  - Administrator setup (OClouds, TemplateInfos)
  - User operations (cluster deployment)
  - Common workflows
  - Troubleshooting
  - Best practices

## Testing

- **[TESTING.md](TESTING.md)** - Testing guide
- **[TEST_SUMMARY.md](TEST_SUMMARY.md)** - Test status summary
- **[UNIT_TEST_FINAL_STATUS.md](UNIT_TEST_FINAL_STATUS.md)** - Detailed test analysis

## Operations

- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Troubleshooting guide
  - Common issues and solutions
  - Diagnostic commands
  - Recovery procedures

## Demos & Presentations

- **[DEMO_GUIDE.md](DEMO_GUIDE.md)** - Demo presentation guide

## Documentation Structure

```
docs/
├── README.md                      # This file - documentation index
├── QUICK_START.md                 # 10-minute quick start
├── DEPLOYMENT.md                  # Complete deployment guide
├── ARCHITECTURE.md                # System architecture
├── PORCH_SETUP.md                 # Porch configuration
├── TROUBLESHOOTING.md             # Troubleshooting guide
├── TESTING.md                     # Testing guide
├── TEST_SUMMARY.md                # Test status
├── UNIT_TEST_FINAL_STATUS.md      # Detailed test analysis
├── DEMO_GUIDE.md                  # Demo guide
├── flux/                          # Flux CD deployment docs
│   ├── README.md                  # Flux docs overview
│   ├── quick-start.md            # Quick start guide
│   ├── deployment-guide.md       # Complete Flux configuration
│   ├── makefile-usage.md         # Makefile targets
│   ├── webhook-setup.md          # Webhook configuration
│   ├── testing-guide.md          # Testing procedures
│   ├── what-gets-removed.md      # Undeployment clarification
│   ├── deploy-flux.sh            # Deployment script
│   └── setup-webhook.sh          # Webhook setup script
└── kpt/                           # kpt deployment docs
    ├── README.md                  # kpt docs overview
    ├── quick-reference.md         # Quick commands
    ├── checklist.md              # Deployment checklist
    └── undeployment.md           # Undeployment guide
```

## Common Tasks

### Deploy the Operator

**Quick deployment with private registry:**
```bash
# Set up Docker registry authentication
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token

# Deploy with pre-built image
make deploy IMG=ghcr.io/your-org/focom-operator:latest IMAGE_PULL_SECRET=ghcr-secret
```

**Build and deploy your own image:**
```bash
make docker-build docker-push deploy IMG=<registry>/<image>:<tag>
```

**Flux deployment:**
```bash
make deploy-with-flux IMG=<registry>/<image>:<tag>

# Or with webhook for instant sync
make deploy-with-flux-webhook IMG=<registry>/<image>:<tag>
```

**kpt deployment:**
```bash
make kpt-package IMG=<registry>/<image>:<tag>
cd kpt-package/
kpt live init --namespace focom-operator-system --inventory-id focom-operator
kpt live apply
```

See [DEPLOYMENT.md](DEPLOYMENT.md) for complete instructions.

### Test the Operator

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Test API
curl http://localhost:8080/health/live
```

See [TESTING.md](TESTING.md) for complete testing guide.

### Troubleshoot Issues

```bash
# Check operator logs
kubectl logs -n focom-operator-system -l control-plane=controller-manager

# Check ConfigSync status (if using ConfigSync)
kubectl get rootsync focom-resources -n config-management-system

# Check Flux status (if using Flux)
kubectl get gitrepository,kustomization -n flux-system

# Check Porch
kubectl get packagerevisions -n default
```

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed troubleshooting.

## Additional Resources

### In This Repository
- [Main README](../README.md) - Project overview
- [API Documentation](../api/openapi/focom-nbi-api.yaml) - OpenAPI specification
- [Postman Collection](../api/postman/) - API testing collection
- [Configuration Samples](../config/samples/) - Sample configurations

### External Resources
- [Nephio Documentation](https://docs.nephio.org/)
- [Porch Documentation](https://github.com/nephio-project/porch)
- [ConfigSync Documentation](https://cloud.google.com/anthos-config-management/docs/config-sync-overview)
- [Flux CD Documentation](https://fluxcd.io/flux/)
- [kpt Documentation](https://kpt.dev/)

## Contributing

When adding new documentation:
1. Follow the existing structure and naming conventions
2. Update this README with links to new documents
3. Add cross-references in related documents
4. Use clear, concise language
5. Include code examples where appropriate
6. Add troubleshooting sections for complex procedures

## Document Conventions

- **Filenames:** Use UPPERCASE for main docs, lowercase for subdirectories
- **Headings:** Use sentence case
- **Code blocks:** Always specify language for syntax highlighting
- **Commands:** Show expected output where helpful
- **Warnings:** Use ⚠️ emoji for critical information
- **Checkboxes:** Use `- [ ]` for checklists
