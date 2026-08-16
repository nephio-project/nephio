# FOCOM Operator Scripts

This directory contains utility scripts for testing and managing the FOCOM Operator.

## test-api.sh

A bash script that demonstrates the complete FOCOM NBI API workflow using curl commands.

### Prerequisites

- `curl` installed
- `jq` installed (for JSON parsing)
- FOCOM Operator deployed and accessible

### Usage

```bash
# Default (assumes API at http://localhost:8080)
./scripts/test-api.sh

# Custom base URL
BASE_URL=http://your-api-endpoint:8080 ./scripts/test-api.sh
```

### What It Does

The script performs a complete workflow demonstration:

1. **Health Check** - Verifies API is accessible
2. **OCloud Workflow**
   - Creates OCloud draft
   - Validates draft
   - Approves draft
3. **TemplateInfo Workflow**
   - Creates TemplateInfo draft
   - Validates draft
   - Approves draft
4. **FocomProvisioningRequest Workflow**
   - Creates FPR draft (references OCloud and TemplateInfo)
   - Validates draft
   - Approves draft
5. **List Resources** - Shows all created resources

### Example Output

```
========================================
FOCOM NBI API Test Script
========================================

ℹ Base URL: http://localhost:8080

========================================
Testing Health Endpoints
========================================

Testing /health/live...
✓ Health check passed
{
  "service": "focom-nbi",
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}

========================================
Creating OCloud Draft
========================================

✓ OCloud draft created
ℹ OCloud ID: 550e8400-e29b-41d4-a716-446655440000
...

========================================
Test Complete!
========================================

✓ All tests passed successfully
ℹ Created resources:
  - OCloud ID: 550e8400-e29b-41d4-a716-446655440000
  - TemplateInfo ID: 550e8400-e29b-41d4-a716-446655440001
  - FPR ID: 550e8400-e29b-41d4-a716-446655440002
```

### Troubleshooting

**jq not found**
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# RHEL/CentOS
sudo yum install jq
```

**Connection refused**
```bash
# Verify API is accessible
curl http://localhost:8080/health/live

# Check port-forward is running
kubectl port-forward -n focom-operator-system \
  svc/focom-operator-controller-manager-nbi-service 8080:8080
```

**Script fails at specific step**
- Check the error message and HTTP status code
- Review operator logs: `kubectl logs -n focom-operator-system -l control-plane=controller-manager`
- Verify previous steps completed successfully

### Customization

You can modify the script to test different scenarios:

```bash
# Edit the script
vim scripts/test-api.sh

# Modify resource names, namespaces, or parameters
# Add additional test cases
# Change the workflow order
```

### Integration with CI/CD

The script can be used in CI/CD pipelines:

```yaml
# Example GitLab CI
test-api:
  stage: test
  script:
    - kubectl port-forward -n focom-operator-system svc/focom-operator-controller-manager-nbi-service 8080:8080 &
    - sleep 5
    - ./scripts/test-api.sh
  after_script:
    - pkill -f "port-forward"
```

## Future Scripts

Additional scripts that could be added:

- `cleanup.sh` - Delete all test resources
- `load-test.sh` - Performance testing
- `backup.sh` - Backup resources from Porch
- `restore.sh` - Restore resources to Porch
- `validate-deployment.sh` - Verify deployment is healthy

## Contributing

When adding new scripts:

1. Make them executable: `chmod +x scripts/your-script.sh`
2. Add usage documentation in this README
3. Include error handling and helpful output
4. Test on multiple environments
