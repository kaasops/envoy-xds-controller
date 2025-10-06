# Status Propagation E2E Test

This directory contains test manifests for the VirtualService status propagation e2e test.

## Test Scenario

The test verifies that VirtualService status is correctly propagated through the following workflow:

1. **Valid VirtualService** → Status: Valid, Snapshot: Created
2. **Invalid VirtualService** (missing domains) → Status: Invalid, Snapshot: Unchanged
3. **Valid VirtualService** (updated) → Status: Valid, Snapshot: Updated

## Test Objective

This test validates the following bugfixes and improvements:

- **Status Application Fix**: Statuses are applied even when build fails (updater.go:245-278)
- **Root Cause Extraction**: Error messages show only the innermost error via `getRootCause()`
- **Snapshot Isolation**: Invalid VirtualServices don't update Envoy snapshots
- **StatusStorage Integration**: Status is stored separately from VS objects

## Test Files

- `listener.yaml` - HTTP listener configuration (port 10080)
- `vs-valid.yaml` - Valid VirtualService with domains
- `vs-invalid.yaml` - Invalid VirtualService (domains commented out)
- `vs-valid-updated.yaml` - Updated valid VirtualService with different domains

## Key Validation Points

### Status Propagation
- Invalid VS status shows concise error message (root cause extraction)
- Error message should NOT contain full error chain
- Example: `"invalid VirtualHost.Domains: value must contain at least 1 item(s)"`
- NOT: `"MainBuilder.BuildResources failed: failed to build resources from virtual service: ..."`

### Snapshot Behavior
- Valid VS creates/updates Envoy snapshot
- Invalid VS does NOT modify existing snapshot
- Switching back to valid VS updates the snapshot

## Test Flow

### Step 1: Apply Valid VirtualService
```yaml
virtualHost:
  domains: [ "example.local" ]
```
- VirtualService status: `invalid: false, message: ""`
- Envoy snapshot: Created with route config for `example.local`

### Step 2: Apply Invalid VirtualService
```yaml
virtualHost:
  # domains missing - invalid!
```
- VirtualService status: `invalid: true, message: "invalid VirtualHost.Domains: ..."`
- Envoy snapshot: **Unchanged** (still contains previous valid config)

### Step 3: Apply Valid VirtualService (Updated)
```yaml
virtualHost:
  domains: [ "updated.local" ]
```
- VirtualService status: `invalid: false, message: ""`
- Envoy snapshot: Updated with new route config for `updated.local`

## Running the Test

```bash
# Run all e2e tests
make test-e2e

# Run only status propagation tests
go test ./test/e2e -ginkgo.focus="Status Propagation"

# Run with verbose output
go test ./test/e2e -ginkgo.focus="Status Propagation" -v
```

## Note on Webhooks

This test automatically detects webhook status:

- **Webhook enabled**: Invalid VS will be rejected by webhook, test skips step 2
- **Webhook disabled**: Invalid VS is accepted, status is updated, full test executes

The test uses `ApplyManifests()` and checks the error to determine webhook status.

## Expected Output

```
STEP: Step 1: Apply valid VirtualService and verify snapshot is created
STEP: Verifying VirtualService status is valid
STEP: Step 2: Apply invalid VirtualService (missing domains) and verify status changes
STEP: Verifying VirtualService status is now invalid
STEP: Status message: invalid VirtualHost.Domains: value must contain at least 1 item(s)
STEP: Verifying snapshot has NOT changed (invalid VS should not update Envoy config)
STEP: Step 3: Apply valid VirtualService again and verify status and snapshot update
STEP: Verifying VirtualService status is valid again
STEP: Test completed successfully: valid -> invalid (status only) -> valid (status + snapshot)
```

## Integration with Core Fixes

This test validates all changes made in `/internal/xds/updater/updater.go`:

1. **getRootCause() function (lines 281-295)**: Extracts innermost error for user-friendly messages
2. **Status application (lines 351, 370)**: Uses `getRootCause()` for concise error messages
3. **Status propagation (lines 258-261)**: Statuses applied even when build fails
