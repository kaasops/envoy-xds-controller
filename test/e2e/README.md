# E2E Tests

This directory contains end-to-end tests for the Envoy XDS Controller. The tests are organized into several contexts:

- **Basic Functionality**: Tests basic functionality of the controller, including applying virtual service manifests and access log configurations.
- **Validation**: Tests validation of various manifests.
- **TCP Proxy**: Tests TCP proxy functionality.
- **Templates**: Tests virtual service template functionality.
- **GRPC API**: Tests the gRPC API functionality.

## Test Dependencies

There is a dependency between the Basic Functionality test and the Templates test. The Basic Functionality test applies the templates configuration first, and then applies the file access logging configuration. This is necessary because the file access logging configuration depends on the templates configuration being applied first.

The `EnvoyFixture.ApplyManifests` method will skip manifests that have already been applied, so the Templates test will not duplicate the application of the templates configuration if it has already been applied by the Basic Functionality test.

## Test Data

The test data is organized into directories that correspond to the test contexts:

- **basic_https_service**: Contains basic HTTPS virtual service configuration.
- **file_access_logging**: Contains file-based access log configuration.
- **virtual_service_templates**: Contains virtual service template configuration.
- **tcp_proxy**: Contains TCP proxy configuration.
- **http_service**: Contains HTTP listener and virtual service configuration.
- **template_validation**: Contains template validation configuration.
- **unused_virtual_service**: Contains a single virtual service file, not used in automated tests.
- **grpc**: Contains configuration for testing gRPC API functionality.

## Running the Tests

To run the tests, use the following command:

```bash
make test-e2e
```

## Troubleshooting

If you encounter issues with the tests, check the following:

1. Make sure the test environment is set up correctly (Kind cluster, Envoy proxy, etc.).
2. Check the logs of the controller and Envoy proxy for errors.
3. Verify that the test data is correct and up-to-date.
4. If you modify the test sequence, be aware of the dependencies between tests.