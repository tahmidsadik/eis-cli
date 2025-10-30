# EIS CLI Examples

This document shows examples of using the EIS CLI commands.

## Pipeline Command Examples

### Basic Usage - Show Pipeline List with Links

```bash
./eiscli svc pipeline authservicev2 -l 3
```

**Output:**

```
Fetching pipeline builds for service: authservicev2 (last 3 builds)

====================================================================================================
#1  Build #884  ✓ COMPLETED
    Result:     SUCCESSFUL
    Target:     branch: staging
    Commit:     70e14a5 - Update authentication flow
    Trigger:    pipeline_trigger_push
    Creator:    Hussein Farhat
    Created:    2025-10-24 08:35:15
    Duration:   4m 30s
    Build Time: 238 seconds
    Link:       https://bitbucket.org/cover42/authservicev2/pipelines/results/884
----------------------------------------------------------------------------------------------------
#2  Build #883  ✓ COMPLETED
    Result:     SUCCESSFUL
    Target:     branch: develop
    Commit:     be2e8d8 - Fix token expiration
    Trigger:    pipeline_trigger_push
    Creator:    Hussein Farhat
    Created:    2025-10-23 21:49:48
    Duration:   3m 39s
    Build Time: 186 seconds
    Link:       https://bitbucket.org/cover42/authservicev2/pipelines/results/883
----------------------------------------------------------------------------------------------------
#3  Build #882  ✓ COMPLETED
    Result:     SUCCESSFUL
    Target:     branch: master
    Commit:     be2e8d8 - Merge develop to master
    Trigger:    pipeline_trigger_push
    Creator:    Hussein Farhat
    Created:    2025-10-23 21:40:28
    Duration:   8m 5s
    Build Time: 1144 seconds
    Link:       https://bitbucket.org/cover42/authservicev2/pipelines/results/882
====================================================================================================
```

### With Steps - Show Pipeline Steps

```bash
./eiscli svc pipeline tenantservice -l 2 --logs
```

**Output:**

```
Fetching pipeline builds for service: tenantservice (last 2 builds)
Fetching pipeline steps and logs...

====================================================================================================
#1  Build #3356  ✓ COMPLETED
    Result:     SUCCESSFUL
    Target:     branch: staging
    Commit:     0a84a16 - Add tenant validation
    Trigger:    pipeline_trigger_push
    Creator:    Ricard Comas Gomez
    Created:    2025-10-28 08:59:24
    Duration:   4m 33s
    Build Time: 452 seconds
    Link:       https://bitbucket.org/cover42/tenantservice/pipelines/results/3356

    Steps:
      1. ✓ Build and push to test account - SUCCESSFUL
      2. ✓ Deploy to staging environment - SUCCESSFUL
----------------------------------------------------------------------------------------------------
#2  Build #3355  ✓ COMPLETED
    Result:     SUCCESSFUL
    Target:     branch: develop
    Commit:     28a2695 - Update tenant schema
    Trigger:    pipeline_trigger_manual
    Creator:    Ricard Comas Gomez
    Created:    2025-10-28 08:58:43
    Duration:   5m 47s
    Build Time: 666 seconds
    Link:       https://bitbucket.org/cover42/tenantservice/pipelines/results/3355

    Steps:
      1. ✓ Lint and test - SUCCESSFUL
====================================================================================================
```

### Failed Pipeline with Steps

When a pipeline fails, you can see which step failed:

```bash
./eiscli svc pipeline tenantservice -l 5 --logs
```

**Output (showing failed build #3354):**

```
====================================================================================================
#3  Build #3354  ✗ COMPLETED
    Result:     FAILED
    Target:     branch: develop
    Commit:     28a2695 - Add new feature
    Trigger:    pipeline_trigger_manual
    Creator:    Ricard Comas Gomez
    Created:    2025-10-27 22:05:03
    Duration:   7m 9s
    Build Time: 832 seconds
    Link:       https://bitbucket.org/cover42/tenantservice/pipelines/results/3354

    Steps:
      1. ✗ Lint and test - FAILED
====================================================================================================
```

### Example with Log Snippets (When Available)

When logs are available from Bitbucket API, they would display like this:

```
====================================================================================================
#1  Build #3354  ✗ COMPLETED
    Result:     FAILED
    Target:     branch: develop
    Commit:     28a2695 - Add new feature
    Trigger:    pipeline_trigger_manual
    Creator:    Ricard Comas Gomez
    Created:    2025-10-27 22:05:03
    Duration:   7m 9s
    Build Time: 832 seconds
    Link:       https://bitbucket.org/cover42/tenantservice/pipelines/results/3354

    Steps:
      1. ✗ Lint and test - FAILED
         Last log lines:
         │ Running test suite...
         │ ✓ Tenant creation tests passed
         │ ✓ Tenant update tests passed
         │ ✗ Tenant deletion tests failed
         │ Error: Database connection timeout
         │ Expected 200, got 500
         │ FAIL: 1 test failed
         │ npm ERR! Test failed. See above for more details.
====================================================================================================
```

## Ingress Command Examples

### Add a Service to API Gateway Ingress Controller

Register a new service in the test environment:

```bash
./eiscli svc ingress add --service myservice --env test
```

**Output:**

```
═══════════════════════════════════════════════════════
                    SUMMARY
═══════════════════════════════════════════════════════

Environment: test (frankfurt)
Service: myservice

✓ Updated files:
  • dist-orchestration/ingress/frankfurt/test/api_ingress.yaml
  • dist-orchestration/ingress/frankfurt/test/funk_api_ingress.yaml

✓ All files updated successfully!
═══════════════════════════════════════════════════════
```

### Add a Service to Production (Frankfurt)

```bash
./eiscli svc ingress add --service myservice --env prod --region frankfurt
```

**Output:**

```
═══════════════════════════════════════════════════════
                    SUMMARY
═══════════════════════════════════════════════════════

Environment: prod (frankfurt)
Service: myservice

✓ Updated files:
  • dist-orchestration/ingress/frankfurt/prod/api_ingress.yaml
  • dist-orchestration/ingress/frankfurt/prod/campingfreunde_api_ingress.yaml
  • dist-orchestration/ingress/frankfurt/prod/funk_api_ingress.yaml

✓ All files updated successfully!
═══════════════════════════════════════════════════════
```

### Add a Service to Zurich Region

```bash
./eiscli svc ingress add --service myservice --env prod --region zurich
```

**Output:**

```
═══════════════════════════════════════════════════════
                    SUMMARY
═══════════════════════════════════════════════════════

Environment: prod (zurich)
Service: myservice

✓ Updated files:
  • dist-orchestration/ingress/zurich/api_ingress.yaml

✓ All files updated successfully!
═══════════════════════════════════════════════════════
```

### Service Already Configured

When a service is already configured, it will be skipped:

```bash
./eiscli svc ingress add --service dunningservice --env test
```

**Output:**

```
═══════════════════════════════════════════════════════
                    SUMMARY
═══════════════════════════════════════════════════════

Environment: test (frankfurt)
Service: dunningservice

⊘ Skipped files (already configured):
  • dist-orchestration/ingress/frankfurt/test/api_ingress.yaml
  • dist-orchestration/ingress/frankfurt/test/funk_api_ingress.yaml

═══════════════════════════════════════════════════════
```

### Partial Configuration Detected

When a service has partial configuration, manual intervention is required:

```bash
./eiscli svc ingress add --service misconfigured --env test
```

**Output:**

```
═══════════════════════════════════════════════════════
                    SUMMARY
═══════════════════════════════════════════════════════

Environment: test (frankfurt)
Service: misconfigured

✗ Files with issues (manual intervention required):
  • dist-orchestration/ingress/frankfurt/test/api_ingress.yaml - Found paths: [/misconfigured], ExcludedAPI: false, ExcludedJSON: false

Please manually remove partial configurations and run the command again.

═══════════════════════════════════════════════════════
```

## List Command Examples

### List All Services in Workspace

```bash
./eiscli svc list
```

**Output:**

```
Fetching repositories from workspace: cover42

Found 284 repositories:

1. emil_api_server
2. emil_be_core
3. authservicev2
4. tenantservice
5. accountservice
...
```

## Configuration Examples

### Using Environment Variables

```bash
export EISCLI_BITBUCKET_USERNAME="bitbucket-app-username"
export EISCLI_BITBUCKET_APP_PASSWORD="your-app-password"
export EISCLI_BITBUCKET_WORKSPACE="bitbucket-app-workspace"

./eiscli svc pipeline authservicev2
```

### Using Config File

Create `~/.eiscli/config.yaml`:

```yaml
bitbucket:
  username: "bitbucket-app-username"
  app_password: "your-app-password"
  workspace: "bitbucket-app-workspace"
```

Then simply run:

```bash
./eiscli svc pipeline authservicev2
```

## Advanced Usage

### Check Multiple Services

```bash
# Create a simple script to check multiple services
for service in authservicev2 tenantservice accountservice; do
  echo "=== $service ==="
  ./eiscli svc pipeline $service -l 1
  echo ""
done
```

### Monitor Failed Builds

```bash
# Show last 10 builds and filter for failures
./eiscli svc pipeline tenantservice -l 10 | grep -B 10 "FAILED"
```

### Get Detailed Information for Failed Builds

```bash
# When you see a failed build, use --logs to get more details
./eiscli svc pipeline tenantservice -l 5 --logs | grep -A 20 "FAILED"
```
