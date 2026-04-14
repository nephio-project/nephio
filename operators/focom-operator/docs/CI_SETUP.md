# CI/CD Setup for FOCOM Operator

This document describes the Continuous Integration setup for the FOCOM Operator.

## GitHub Actions Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

**Jobs:**

#### Build and Test
- **Go Setup**: Uses Go 1.23.8 with module caching
- **Code Formatting**: Runs `gofmt` to ensure consistent formatting
- **Static Analysis**: Runs `go vet` for static code analysis
- **Build**: Compiles the operator using `make build`
- **Unit Tests**: Runs isolated unit tests that don't require external dependencies:
  - `./internal/nbi/models/` - Data model tests
  - `./internal/nbi/validation/` - Validation logic tests
  - `./internal/controller/` - Controller validation tests (specific test only)
- **Coverage**: Uploads test coverage to Codecov

#### Lint
- **golangci-lint**: Comprehensive linting using golangci-lint v1.59.1
- **Configuration**: Uses `.golangci.yml` for linter settings

#### Security
- **Gosec**: Security vulnerability scanning
- **SARIF Upload**: Results uploaded to GitHub Security tab

#### Docker Build Test
- **Docker Build**: Verifies the Dockerfile builds successfully

### 2. Integration Tests Workflow (`.github/workflows/integration-tests.yml`)

**Triggers:**
- Push to `main` branch
- Pull requests to `main` branch
- Manual workflow dispatch

**Purpose:**
- Sets up Kind (Kubernetes in Docker) cluster
- Prepares environment for integration tests
- Currently configured but not running tests due to external dependencies

## Test Strategy

### Unit Tests (Included in CI)
These tests run in isolation without external dependencies:

```bash
# Models tests - data structures and business logic
go test ./internal/nbi/models/

# Validation tests - input validation and business rules
go test ./internal/nbi/validation/

# Controller tests - specific validation logic only
go test ./internal/controller/ -run TestValidateTemplateAlignment
```

### Integration Tests (Not in CI yet)
These tests require external dependencies and are currently excluded:

```bash
# Full integration tests (require Porch, Kubernetes cluster)
make test

# E2E tests (require full environment)
make test-e2e
```

**Why excluded:**
- Require Porch (Nephio Package Orchestration) setup
- Need real Kubernetes cluster connectivity
- Have external service dependencies
- Some tests have mock/setup issues that need fixing

## Local Development

### Running Tests Locally

```bash
# Navigate to operator directory
cd focom-operator

# Run unit tests only (same as CI)
go test ./internal/nbi/models/ ./internal/nbi/validation/
go test ./internal/controller/ -run TestValidateTemplateAlignment

# Run all tests (including integration - may fail without proper setup)
make test

# Run linting
make lint-local

# Build the project
make build
```

### Pre-commit Checks

Before committing, ensure these pass locally:

```bash
# Format check
gofmt -s -l . | wc -l  # Should return 0

# Vet check
go vet ./...

# Build check
make build

# Unit tests
go test ./internal/nbi/models/ ./internal/nbi/validation/
```

## Future Improvements

### Integration Test Environment
To enable integration tests in CI, we need:

1. **Porch Setup**: Install and configure Nephio Porch in the Kind cluster
2. **Test Isolation**: Fix integration tests to use proper mocking/test doubles
3. **Environment Configuration**: Set up required secrets and configurations
4. **Test Stability**: Address flaky tests and timing issues

### Enhanced Coverage
- Add more unit tests for uncovered code paths
- Implement property-based testing for complex business logic
- Add performance benchmarks

### Security Enhancements
- Add dependency vulnerability scanning
- Implement container image scanning
- Add SAST (Static Application Security Testing) tools

## Troubleshooting

### Common CI Failures

1. **Formatting Issues**: Run `gofmt -w .` to fix formatting
2. **Linting Errors**: Check `.golangci.yml` configuration and fix reported issues
3. **Build Failures**: Ensure all dependencies are properly declared in `go.mod`
4. **Test Failures**: Run tests locally to debug issues

### Integration Test Issues

If you need to run integration tests locally:

1. Ensure you have a Kubernetes cluster available
2. Install Porch following the setup documentation
3. Configure proper authentication and access
4. Some tests may need environment-specific configuration

## Configuration Files

- `.github/workflows/ci.yml` - Main CI workflow
- `.github/workflows/integration-tests.yml` - Integration test workflow  
- `focom-operator/.golangci.yml` - Linter configuration
- `focom-operator/Makefile` - Build and test targets