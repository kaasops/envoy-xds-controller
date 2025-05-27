# Testing Guide: Envoy XDS Controller

This document provides guidelines and instructions for testing the Envoy XDS Controller.

## Table of Contents

1. [Testing Overview](#testing-overview)
2. [Test Types](#test-types)
3. [Running Tests](#running-tests)
4. [Writing Tests](#writing-tests)
5. [Continuous Integration](#continuous-integration)
6. [Test Coverage](#test-coverage)

## Testing Overview

The Envoy XDS Controller uses a comprehensive testing strategy to ensure code quality and functionality. The testing approach includes:

- Unit tests for individual components
- Integration tests for component interactions
- End-to-end tests for full system validation
- Linting for code quality

## Test Types

### Unit Tests

Unit tests focus on testing individual functions and methods in isolation. They are located alongside the code they test and follow the Go convention of `*_test.go` files.

Key packages with unit tests:
- `internal/xds/`: Tests for xDS server implementation
- `internal/cache/`: Tests for cache implementation
- `internal/updater/`: Tests for configuration updaters
- `api/`: Tests for API types and methods

### Integration Tests

Integration tests verify that different components work together correctly. These tests typically involve multiple packages and may use mocks for external dependencies.

### End-to-End Tests

End-to-end (e2e) tests validate the entire system in a real or simulated environment. The e2e tests for Envoy XDS Controller:

- Use a Kind Kubernetes cluster
- Deploy the controller and required components
- Verify functionality through API calls and state checks
- Clean up resources after testing

E2e tests are located in the `test/e2e/` directory.

### Linting

Linting ensures code quality and consistency. The project uses `golangci-lint` with a configuration defined in `.golangci.yml`.

## Running Tests

### Prerequisites

- Go v1.22.0+
- Docker (for e2e tests)
- Kind (for e2e tests)
- Access to a Kubernetes cluster (for e2e tests)

### Running Unit Tests

To run all unit tests:

```bash
make test
```

To run tests for a specific package:

```bash
go test ./path/to/package
```

To run tests with verbose output:

```bash
go test -v ./path/to/package
```

### Running End-to-End Tests

Before running e2e tests, ensure you have a Kind cluster running:

```bash
kind create cluster
```

Then run the e2e tests:

```bash
make test-e2e
```

### Running Linters

To run linters:

```bash
make lint
```

To automatically fix linting issues where possible:

```bash
make lint-fix
```

## Writing Tests

### Unit Test Guidelines

1. **Test File Location**: Place test files in the same package as the code being tested.
2. **Naming Convention**: Use `*_test.go` suffix for test files.
3. **Test Function Naming**: Name test functions as `Test<FunctionName>` or `Test<Behavior>`.
4. **Table-Driven Tests**: Use table-driven tests for testing multiple scenarios.
5. **Mocking**: Use interfaces and mock implementations for external dependencies.

Example unit test:

```
// TestResourceBuilder_BuildListener tests the BuildListener function
func TestResourceBuilder_BuildListener(t *testing.T) {
    // Setup test cases
    testCases := []struct {
        name     string
        input    *v1alpha1.Listener
        expected *envoy_config_listener_v3.Listener
        wantErr  bool
    }{
        // Test cases...
    }

    // Run test cases
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            builder := NewResourceBuilder()
            result, err := builder.BuildListener(tc.input)

            if tc.wantErr {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### End-to-End Test Guidelines

1. **Test Setup**: Create necessary resources for the test.
2. **Test Execution**: Perform actions to test functionality.
3. **Verification**: Verify the expected outcomes.
4. **Cleanup**: Clean up resources after the test.

Example e2e test structure:

```
// Ginkgo test for Envoy XDS Controller
var _ = Describe("Envoy XDS Controller", func() {
    Context("When creating a VirtualService", func() {
        It("Should create corresponding Envoy configuration", func() {
            // Setup
            vs := createVirtualService()

            // Execution
            err := k8sClient.Create(context.Background(), vs)
            Expect(err).NotTo(HaveOccurred())

            // Verification
            Eventually(func() bool {
                // Check if configuration was created correctly
                return checkEnvoyConfiguration(vs)
            }, timeout, interval).Should(BeTrue())

            // Cleanup
            err = k8sClient.Delete(context.Background(), vs)
            Expect(err).NotTo(HaveOccurred())
        })
    })
})
```

## Continuous Integration

The project uses GitHub Actions for continuous integration. The CI pipeline:

1. Runs unit tests
2. Runs linters
3. Builds the controller
4. Runs e2e tests
5. Builds and pushes Docker images (on release)

## Test Coverage

To generate a test coverage report:

```bash
make test
go tool cover -html=cover.out
```

The project aims to maintain high test coverage, especially for critical components like the xDS server, cache, and API handlers.
