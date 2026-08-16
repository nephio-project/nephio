# kpt Deployment Documentation

This directory contains documentation specific to deploying the FOCOM operator using kpt.

## Documents

### [Quick Reference](quick-reference.md)
Quick commands and common questions for kpt deployment.

**Use this when:**
- You need a quick command reference
- You have a specific question about kpt deployment
- You want to see workflow comparisons

**Contents:**
- Quick commands (generate, init, deploy, remove)
- Common questions and answers
- Workflow comparisons (make deploy vs kpt)
- Integration with ConfigSync
- Troubleshooting

### [Deployment Checklist](checklist.md)
Step-by-step checklist for deploying with kpt.

**Use this when:**
- You're deploying for the first time
- You want to ensure you don't miss any steps
- You need verification commands at each stage

**Contents:**
- Pre-deployment checklist
- Deployment steps with verification
- Post-deployment checklist
- Uninstall steps
- Troubleshooting

### [Undeployment Guide](undeployment.md)
Comprehensive guide for removing the operator deployed with kpt.

**Use this when:**
- You need to remove the operator
- You're troubleshooting a failed undeployment
- You need to understand why order matters

**Contents:**
- Why removal order matters
- Complete undeployment procedure
- Automated undeployment script
- Verification checklist
- Troubleshooting stuck resources
- Partial removal scenarios
- Recovery procedures

## Quick Start

### Deploy
```bash
# Generate package
make kpt-package IMG=<registry>/<image>:<tag>

# Initialize (one-time)
cd kpt-package/
kpt live init --namespace focom-operator-system --inventory-id focom-operator

# Deploy operator
kpt live apply

# Deploy ConfigSync
cd ..
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -
kubectl apply -k config/configsync
```

### Undeploy
```bash
# Remove ConfigSync first
kubectl delete -k config/configsync
kubectl delete secret gitea-secret -n config-management-system

# Wait for cleanup
sleep 10

# Remove operator
cd kpt-package/
kpt live destroy
```

## When to Use kpt Deployment

**Use kpt deployment when:**
- You want state tracking of deployed resources
- You need automatic pruning of deleted resources
- You want to preview changes before applying
- You're implementing GitOps workflows
- You need better control over resource lifecycle

**Use `make deploy` when:**
- You're doing rapid development iterations
- You want the simplest deployment method
- You want ConfigSync deployed automatically
- You don't need state tracking

## See Also

- [Main Deployment Guide](../DEPLOYMENT.md) - All deployment methods
- [Architecture](../ARCHITECTURE.md) - System architecture
- [Troubleshooting](../TROUBLESHOOTING.md) - General troubleshooting
- [Porch Setup](../PORCH_SETUP.md) - Porch configuration

## Document Structure

```
docs/kpt/
├── README.md              # This file - overview and navigation
├── quick-reference.md     # Quick commands and FAQ
├── checklist.md          # Step-by-step deployment checklist
└── undeployment.md       # Comprehensive undeployment guide
```
