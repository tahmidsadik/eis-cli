# EIS CLI Quick Start Guide

This guide will help you get started with the EIS CLI quickly.

## Prerequisites

- Go 1.24.0 or later installed
- Bitbucket credentials (username and app password with Deployments: Write permission)
- Access to the cover42 workspace

## Installation

1. Clone or navigate to the repository:

```bash
cd /path/to/eis-cli
```

2. Build the CLI:

```bash
go build -o eiscli .
```

3. (Optional) Move to your PATH:

```bash
sudo mv eiscli /usr/local/bin/
```

## Configuration

### Quick Setup with Environment Variables

```bash
export EISCLI_BITBUCKET_USERNAME="your-username"
export EISCLI_BITBUCKET_APP_PASSWORD="your-app-password"
export EISCLI_BITBUCKET_WORKSPACE="cover42"
```

Add these to your `~/.bashrc`, `~/.zshrc`, or `~/.profile` for permanent setup.

### Alternative: Config File

Create `~/.eiscli/config.yaml`:

```yaml
bitbucket:
  username: "your-username"
  app_password: "your-app-password"
  workspace: "cover42"
```

## Basic Usage

### 1. List All Services

See all available services in the workspace:

```bash
./eiscli svc list
```

### 2. View Pipeline Builds

**Option A: Auto-detect from current git repository**

If you're inside a service's git repository, simply run:

```bash
cd /path/to/your/service/repo
./eiscli svc pipeline
```

The CLI will automatically detect the service name from the git remote URL.

**Option B: Specify service name explicitly**

Check the last 5 pipeline builds for a service:

```bash
./eiscli svc pipeline authservicev2
```

View more builds:

```bash
./eiscli svc pipeline tenantservice -l 10
```

### 3. View Variables

**Option A: View deployment variables for Test environment (default)**

```bash
./eiscli svc variables
```

**Option B: View deployment variables for a specific environment**

```bash
./eiscli svc variables myservice --env Staging
./eiscli svc variables myservice -e Production
```

**Option C: View all deployment variables across all environments**

```bash
./eiscli svc variables myservice --all
```

**Option D: View repository-level variables**

```bash
./eiscli svc variables myservice --type repository
```

### 4. Sync Variables from Kubernetes Templates

**Option A: Preview sync for testing environment (no changes made)**

```bash
./eiscli svc variables sync --env testing
```

**Option B: Actually sync variables for testing environment**

```bash
./eiscli svc variables sync --env testing --apply
```

**Option C: Sync for production environment**

```bash
./eiscli svc variables sync myservice --env prod --apply
```

The sync command:

- Reads variables from `kubernetes/overlays/{env}/.env.template`
- Shows a preview of ALL variables with color coding:
  - **Green**: New variables that will be created
  - **Cyan**: Variables that already exist in Bitbucket
- Only adds missing variables, never removes existing ones
- Auto-detects secured variables (PASSWORD, SECRET, KEY, TOKEN, etc.)
- Requires confirmation before creating variables (with `--apply`)

### 5. Common Services

Here are some commonly used EIS services:

- `authservicev2` - Authentication service
- `tenantservice` - Tenant management
- `accountservice` - Account management
- `notificationservicev2` - Notifications
- `billingservice` - Billing
- `documentservicev2` - Document management

## Example Workflow

```bash
# 1. Set up environment variables
export EISCLI_BITBUCKET_USERNAME="your-username"
export EISCLI_BITBUCKET_APP_PASSWORD="your-app-password"
export EISCLI_BITBUCKET_WORKSPACE="cover42"

# 2. List all services
./eiscli svc list

# 3. Check pipeline status - auto-detect from git repo
cd /path/to/your/service/repo
./eiscli svc pipeline -l 3

# Or specify service name explicitly
./eiscli svc pipeline authservicev2 -l 3

# 4. View variables for Test environment (default)
./eiscli svc variables

# View variables for Production environment
./eiscli svc variables --env Production

# View repository variables
./eiscli svc variables --type repository

# 5. Sync variables from Kubernetes templates (preview)
./eiscli svc variables sync --env testing

# Sync variables (actually create them)
./eiscli svc variables sync --env staging --apply

# 6. View help
./eiscli --help
./eiscli svc --help
./eiscli svc pipeline --help
```

## Troubleshooting

### "Configuration error" message

Make sure you've set all three environment variables:

- `EISCLI_BITBUCKET_USERNAME`
- `EISCLI_BITBUCKET_APP_PASSWORD`
- `EISCLI_BITBUCKET_WORKSPACE`

### "404 Not Found" error

The service name might be incorrect. Use `./eiscli svc list` to see all available services.

### Authentication errors

Verify your Bitbucket app password has the required permissions:

- Repositories: Read
- Pipelines: Read
- Deployments: Write (required for creating environments and variables)

### "Could not auto-detect from git repository" error

This happens when:

1. You're not inside a git repository, or
2. The git repository doesn't have an `origin` remote configured

**Solutions:**

- Make sure you're in the correct directory with `git remote -v`
- Or provide the service name explicitly: `./eiscli svc pipeline <service-name>`

### ".env.template file not found" error

This happens when syncing variables and the kubernetes folder structure doesn't match expectations:

1. The default path is `./kubernetes/overlays/{env}/.env.template`
2. Make sure you're running from the service root directory
3. Or specify a custom path: `--kubernetes-path ./k8s`

**Solutions:**

- Ensure the kubernetes folder exists with the correct structure
- Use `--kubernetes-path` flag to specify a different location
- Check that the overlay folder matches your environment name (testing, staging, prod, etc.)

### "Permission denied: You don't have permission to create deployment environments" error

This happens when your Bitbucket app password doesn't have the required permissions.

**Solutions:**

- Create a new Bitbucket app password with **Deployments: Write** permission
- Go to https://bitbucket.org/account/settings/app-passwords/
- Make sure to select: Repositories: Read, Pipelines: Read, **Deployments: Write**
- Update your config or environment variables with the new password

## Next Steps

- Explore other subcommands: `status`, `variables`, `new`
- Check the main README.md for detailed documentation
- Contribute new features!
