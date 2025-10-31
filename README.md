# EIS CLI

A command-line tool for managing services, repositories, pipelines, and deployments on the EIS platform.

## Installation

### Download Binary

Download the latest release for your platform:

- **macOS (Intel)**: [eiscli-darwin-amd64](https://bitbucket.org/cover42/eis-cli/downloads/)
- **macOS (Apple Silicon)**: [eiscli-darwin-arm64](https://bitbucket.org/cover42/eis-cli/downloads/)
- **Linux (x86_64)**: [eiscli-linux-amd64](https://bitbucket.org/cover42/eis-cli/downloads/)

#### macOS: Remove Quarantine Flag

After downloading on macOS, remove the quarantine attribute to prevent security warnings:

```bash
xattr -d com.apple.quarantine ./eiscli*
```

Without this step, macOS may flag the binary as untrusted and refuse to run it.

#### Make Executable and Install

```bash
# Make executable
chmod +x eiscli-*

# Rename and move to PATH (choose a short name for convenience)
sudo mv eiscli-* /usr/local/bin/eiscli
# or for even shorter commands:
sudo mv eiscli-* /usr/local/bin/eis
```

You can now use the CLI from anywhere:

```bash
eiscli auth login
# or if you renamed it to 'eis':
eis auth login
```

### Build from Source

```bash
go build -o eiscli .
sudo mv eiscli /usr/local/bin/
```

## Getting Started

### 1. Authenticate

```bash
eiscli auth login
```

A browser window will open for authorization. The CLI will handle all necessary configuration automatically.

### 2. Try It Out

```bash
# List all services
eiscli svc list

# View recent pipeline builds (auto-detects from git repo)
cd /path/to/your/service
eiscli pipelines

# List deployment variables
eiscli vars --env Test
```

## Commands

### Authentication

```bash
# Login via OAuth
eiscli auth login

# Check authentication status
eiscli auth status

# Logout
eiscli auth logout

# Manually refresh token (normally automatic)
eiscli auth refresh
```

### Service Management

Most commands auto-detect the service name from your current git repository.

```bash
# List all services
eiscli svc list
```

### Pipelines

```bash
# View recent pipeline builds (default: last 5)
eiscli pipelines [service-name]

# View more builds
eiscli pipelines -l 10

# View with detailed steps and logs
eiscli pipelines --logs --log-lines 20
```

**Options:**

- `-l, --limit`: Number of builds to display (default: 5)
- `--logs`: Show pipeline steps and log snippets
- `--log-lines`: Log lines per step (default: 25)

### Variables

```bash
# List deployment variables (default: Test environment)
eiscli vars [service-name]

# List for specific environment
eiscli vars --env Staging

# List all environments
eiscli vars --all

# List repository variables
eiscli vars --type repository
```

**Options:**

- `-e, --env`: Environment name (Test, Staging, Production, Production-Zurich)
- `-t, --type`: Variable type (deployment, repository)
- `-a, --all`: Show all environments
- `--auto-create-env`: Auto-create missing environments

### Sync Variables

Syncs variables from Kubernetes `.env.template` files to Bitbucket. Shows preview by default (like `terraform plan`).

```bash
# Preview changes (no modifications)
eiscli vars sync --env testing

# Apply changes (prompts for confirmation)
eiscli vars sync --env prod --apply

# Custom kubernetes path
eiscli vars sync --env staging --kubernetes-path ./k8s --apply
```

**Options:**

- `-e, --env`: Environment (testing, staging, prod, prod-zurich, dev) **[required]**
- `-k, --kubernetes-path`: Path to kubernetes folder (default: ./kubernetes)
- `-a, --apply`: Apply changes (default: preview only)
- `--auto-create-env`: Auto-create missing environments

**Template Format** (`kubernetes/overlays/{env}/.env.template`):

```bash
KEY_NAME=${PLACEHOLDER_VALUE}
DATABASE_PASSWORD=${DB_PASS}  # Auto-detected as secured
```

Variables containing PASSWORD, SECRET, TOKEN, KEY, etc. are automatically marked as secured.

### Ingress Registration

Register a microservice in the API Gateway ingress controller. Must be run from the `dist-orchestration` repository.

```bash
# Add service to test environment
eiscli ingress add --service myservice --env test

# Add to production in Zurich
eiscli ingress add --service myservice --env prod --region zurich
```

**Options:**

- `-s, --service`: Service name **[required]**
- `-e, --env`: Environment (test, dev, staging, prod, perf) **[required]**
- `-r, --region`: Region (frankfurt, zurich) (default: frankfurt)

### ECR Registry

Check and manage AWS ECR registries. Auto-selects AWS profile based on environment.

```bash
# Check ECR registry (default: testing)
eiscli ecr [service-name]

# Check specific environment
eiscli ecr --env staging

# Check all environments
eiscli ecr --all

# Create if doesn't exist
eiscli ecr --create
```

**Options:**

- `-e, --env`: Environment (default: testing)
- `-c, --create`: Create repository if missing
- `-a, --all`: Check all environments

**AWS Configuration** (`~/.eiscli/config.yaml`):

```yaml
aws:
  default_profile: "default" # For prod/testing
  nonprod_profile: "staging" # For staging/dev
  region: "eu-central-1"
```

## Configuration

The CLI automatically creates `~/.eiscli/config.yaml` with sensible defaults on first run. You can customize these settings as needed.

### Configuration File

The auto-generated `~/.eiscli/config.yaml` contains:

```yaml
bitbucket:
  workspace: "cover42" # Change if using a different workspace

# AWS and deployment settings are commented out by default
# Uncomment and modify as needed
```

You can edit this file to customize settings. Most developers won't need to change anything.

### Environment Variables

You can override any config setting with environment variables:

```bash
# Bitbucket workspace (if using a different one)
export EISCLI_BITBUCKET_WORKSPACE="your-workspace"

# AWS profile overrides
export AWS_PROFILE="custom-profile"
export AWS_REGION="eu-central-1"
```

## License

Proprietary - Cover42/EIS Platform
