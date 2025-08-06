# E2E Test Data Directory Structure

This directory contains test data used by the end-to-end (e2e) tests. The directories are organized by test scenario to make it easier to understand how they correspond to the tests.

## Directory Structure

| Directory | Purpose | Test File | Description |
|-----------|---------|-----------|-------------|
| basic_https_service | Basic HTTPS virtual service | envoy_basic_test.go | Contains configuration for basic HTTPS virtual service tests with TLS |
| virtual_service_templates | Virtual service templates | envoy_templates_test.go | Contains configuration for testing virtual service templates |
| template_extra_fields | Template extra fields | envoy_templates_test.go | Contains configuration for testing template extra fields functionality |
| file_access_logging | File access logging | envoy_basic_test.go | Contains configuration for testing file-based access logging |
| tcp_proxy | TCP proxy | envoy_tcp_proxy_test.go | Contains configuration for testing TCP proxy functionality |
| http_service | HTTP service | envoy_basic_test.go | Contains configuration for testing HTTP services without TLS |
| unused_virtual_service | Unused virtual service | - | Contains a single virtual service file, not used in automated tests |
| template_validation | Template validation | envoy_validation_test.go | Contains configuration for testing template validation with multiple root routes |
| grpc | gRPC API | grpc_api_test.go | Contains configuration for testing gRPC API functionality |
