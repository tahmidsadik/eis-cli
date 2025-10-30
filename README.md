# EIS CLI

A command-line tool for developers working with the EIS platform. It helps manage services, repositories, pipelines, and deployment configurations.

## Installation

### Build from source

```bash
go build -o eiscli .
```

### Install globally

```bash
go install
```

## Configuration

Before using the CLI, you need to configure your Bitbucket authentication. The CLI supports two authentication methods:

### Option 1: OAuth 2.0 (Recommended)

OAuth provides better security and easier setup - no app passwords needed!

#### Step 1: Create OAuth Consumer in Bitbucket

1. Go to your Bitbucket workspace settings: `https://bitbucket.org/[workspace]/workspace/settings/oauth-consumers/new`
2. Click "Add consumer"
3. Fill in the details:
   - **Name**: "EIS CLI" (or your preference)
   - **Callback URL**: `http://localhost:PORT/callback` (use this exact format - PORT is dynamic)
   - **Permissions**: Select these scopes:
     - **Repositories**: Read
     - **Pipelines**: Read and Write
     - **Pipeline Variables**: Write
     - **Webhooks**: Read and Write
4. Click "Save"
5. Copy the **Key** (client_id) and **Secret** (client_secret)

#### Step 2: Configure EIS CLI

Create a configuration file at `~/.eiscli/config.yaml` or `./config.yaml`:

```yaml
bitbucket:
  client_id: "your-oauth-consumer-key"
  client_secret: "your-oauth-consumer-secret"
  use_oauth: true
  workspace: "your-workspace"
```

Or use environment variables:

```bash
export EISCLI_BITBUCKET_CLIENT_ID="your-oauth-consumer-key"
export EISCLI_BITBUCKET_CLIENT_SECRET="your-oauth-consumer-secret"
export EISCLI_BITBUCKET_USE_OAUTH="true"
export EISCLI_BITBUCKET_WORKSPACE="your-workspace"
```

#### Step 3: Login

```bash
eiscli auth login
```

This will:
1. Open your browser for authorization
2. Exchange the authorization code for access token
3. Save tokens securely to `~/.eiscli/tokens.json`

**Token Management:**
- Access tokens expire after 1 hour but are automatically refreshed
- Run `eiscli auth status` to check token validity
- Run `eiscli auth refresh` to manually refresh token
- Run `eiscli auth logout` to clear stored tokens

### Option 2: Basic Auth with App Password (Legacy)

If you prefer to use app passwords (not recommended for new setups):

#### Environment Variables

```bash
export EISCLI_BITBUCKET_USERNAME="your-username"
export EISCLI_BITBUCKET_APP_PASSWORD="your-app-password"
export EISCLI_BITBUCKET_WORKSPACE="your-workspace"
```

#### Configuration File

```yaml
bitbucket:
  username: "your-username"
  app_password: "your-app-password"
  workspace: "your-workspace"
  use_oauth: false  # or omit this line
```

See `config.yaml.example` for a complete template.

### Creating a Bitbucket App Password (for Basic Auth)

**Note:** This method is being phased out in favor of OAuth. Use OAuth for new installations.

1. Go to https://bitbucket.org/account/settings/app-passwords/
2. Click "Create app password"
3. Give it a label (e.g., "EIS CLI")
4. Select the following permissions:
   - **Repositories**: Read
   - **Pipelines**: Read and Write
   - **Deployments**: Write (required for creating deployment environments and variables)
   - **Webhooks**: Read and Write
5. Click "Create"
6. Copy the generated password (you won't be able to see it again!)

### Migrating from App Password to OAuth

If you're currently using an app password and want to migrate to OAuth:

1. Follow the OAuth setup steps above to create an OAuth consumer
2. Add OAuth credentials to your config file:
   ```yaml
   bitbucket:
     # Add these OAuth settings
     client_id: "your-oauth-consumer-key"
     client_secret: "your-oauth-consumer-secret"
     use_oauth: true
     
     # Keep your existing settings (will be ignored when use_oauth is true)
     username: "your-username"
     app_password: "your-app-password"
     workspace: "your-workspace"
   ```
3. Run `eiscli auth login` to authenticate via OAuth
4. Test with any command (e.g., `eiscli svc list`)
5. Once confirmed working, you can remove the `username` and `app_password` lines from your config

## Usage

```bash
eiscli [command]
```

## Available Commands

### Root Command

```bash
eiscli --help
```

### Authentication (`auth`)

The `auth` command manages OAuth authentication with Bitbucket.

#### Login

```bash
eiscli auth login
```

Authenticate via OAuth by opening your browser and granting access. Tokens are saved securely to `~/.eiscli/tokens.json`.

#### Check Status

```bash
eiscli auth status
```

Shows your current authentication method (OAuth or Basic Auth) and token status.

**Example output (OAuth):**
```
Authentication Status
====================

Authentication Method: OAuth 2.0

✓ Access token is valid

Token Type: Bearer
Expires At: Mon, 31 Oct 2025 14:30:00 UTC
Time Until Expiry: 45m
Scopes: repository pipeline pipeline:write pipeline:variable webhook

Token File: /Users/username/.eiscli/tokens.json
```

#### Logout

```bash
eiscli auth logout
```

Clears stored OAuth tokens. You'll need to run `eiscli auth login` again to use OAuth.

#### Refresh Token

```bash
eiscli auth refresh
```

Manually refreshes the OAuth access token (normally done automatically).

### Service Management (`svc`)

The `svc` command provides subcommands to manage services in the EIS platform.

#### Create a New Service

```bash
eiscli svc new <service-name>
```

Creates a new service with all necessary scaffolding including repository setup, ECR registry, and pipeline configuration.

#### Check Service Status

```bash
eiscli svc status [service-name]
```

Displays the status of a service including:

- Repository information from Bitbucket
- ECR registry details from AWS
- Pipeline variables and configuration
- Current deployment status

**Auto-Detection:** If you run this command from within a git repository, the service name will be automatically detected from the git remote URL.

#### List All Services

```bash
eiscli svc list
```

Lists all repositories/services in the configured workspace. Useful for discovering available service names.

#### View Pipeline Builds

```bash
eiscli svc pipeline [service-name]
```

Lists the last 5 pipeline builds from Bitbucket (default).

**Auto-Detection:** If you run this command from within a git repository, the service name will be automatically detected from the git remote URL. You can still override it by providing a service name explicitly.

Displays comprehensive information including:

- Build number and status (✓ COMPLETED, ✗ FAILED, ● IN_PROGRESS, etc.)
- Result (SUCCESSFUL, FAILED, etc.)
- Target branch/tag
- Commit hash and message
- Trigger type (push, manual, schedule)
- Creator
- Created timestamp
- Duration
- Build time in seconds

**Options:**

- `-l, --limit <number>`: Number of pipeline builds to display (default: 5)
- `-s, --logs`: Show pipeline steps and log snippets (slower, fetches additional data)
- `--log-lines <number>`: Number of log lines to display per step (default: 10, only with --logs)

**Examples:**

```bash
# Auto-detect service from current git repository
cd /path/to/your/service/repo
eiscli svc pipeline

# View last 5 pipeline builds with explicit service name
eiscli svc pipeline authservicev2

# View last 10 builds
eiscli svc pipeline tenantservice -l 10

# View pipelines with detailed steps
eiscli svc pipeline authservicev2 -l 3 --logs

# View with more log lines per step
eiscli svc pipeline tenantservice -l 2 --logs --log-lines 20

# First, list all services to find the correct name
eiscli svc list
```

**Features:**

- Displays clickable Bitbucket URLs for each pipeline
- With `--logs` flag: Shows all pipeline steps with their status
- Identifies failed steps with visual indicators
- Shows duration and build time for each pipeline

#### List Variables

```bash
eiscli svc variables [service-name]
```

Lists Bitbucket deployment variables and repository variables in a formatted table view.

**Auto-Detection:** If you run this command from within a git repository, the service name will be automatically detected from the git remote URL.

**Default Behavior:** Shows deployment variables for the "Test" environment.

**Options:**

- `-t, --type <type>`: Type of variables to display (deployment, repository) (default: deployment)
- `-e, --env <name>`: Environment name to filter (deployment variables only) (default: Test)
- `-a, --all`: Show variables for all environments (deployment variables only)
- `--auto-create-env`: Automatically create missing environments without prompting
- `--env-type <type>`: Override environment type (Test, Staging, Production)

**Features:**

- Displays variables in a clean table format with Name, Value, and Secured columns
- Masks secured variable values with asterisks (**\*\*\*\***)
- Shows total count of variables
- Automatically prompts to create missing deployment environments
- Supports environment-specific filtering for deployment variables
- Repository variables are environment-agnostic

**Examples:**

```bash
# Auto-detect service and show Test environment deployment variables (default)
eiscli svc variables

# Show deployment variables for a specific environment
eiscli svc variables myservice --env Staging
eiscli svc variables myservice -e Production

# Show deployment variables for ALL environments
eiscli svc variables myservice --all

# Show repository-level variables
eiscli svc variables myservice --type repository
eiscli svc variables myservice -t repository

# Auto-create missing environment without prompting
eiscli svc variables myservice --env NewEnv --auto-create-env

# Override inferred environment type
eiscli svc variables myservice --env CustomEnv --env-type Production

# Explicit service name with auto-detected from git
cd /path/to/service/repo
eiscli svc variables --env Production
```

**Common Environments:**

- Test
- Development
- Staging
- Production
- Production-Zurich

**Note:** Repository variables and deployment variables are kept separate and cannot be viewed together.

#### Sync Variables

```bash
eiscli svc variables sync [service-name] --env <environment>
```

Syncs deployment variables from Kubernetes `.env.template` files to Bitbucket deployment environments.

**Auto-Detection:** If you run this command from within a git repository, the service name will be automatically detected from the git remote URL.

**Default Behavior:** Shows a preview of changes (like `terraform plan`). Use `--apply` to actually create the variables.

**Important:** This command only ADDS missing variables and never removes existing ones from Bitbucket.

**Options:**

- `-e, --env <environment>`: Environment to sync (required: testing, staging, prod, prod-zurich, dev)
- `-k, --kubernetes-path <path>`: Path to kubernetes folder (default: ./kubernetes)
- `-a, --apply`: Actually apply the changes (without this, just shows preview)
- `--auto-create-env`: Automatically create missing environments without prompting
- `--env-type <type>`: Override environment type (Test, Staging, Production)

**Environment Mapping:**

- testing → Test
- staging → Staging
- prod → Production
- prod-zurich → Production-Zurich
- dev → Development

**Features:**

- Reads variables from `kubernetes/overlays/{env}/.env.template` files
- Parses template format: `KEY=${VALUE_HERE}`
- Automatically detects secured variables based on naming patterns (PASSWORD, SECRET, KEY, TOKEN, API_KEY, etc.)
- Automatically prompts to create missing deployment environments with smart type inference
- Shows preview table with ALL variables from template:
  - **Green text**: New variables that will be created
  - **Cyan text**: Variables that already exist in Bitbucket
- Variables are sorted with NEW ones first, then EXISTING ones
- Requires confirmation before creating variables (when using `--apply`)
- Only adds missing variables, never removes existing ones
- Displays summary with color-coded results

**Examples:**

```bash
# Preview what would be synced for testing environment (no changes made)
eiscli svc variables sync myservice --env testing

# Preview for staging environment
eiscli svc variables sync myservice --env staging

# Actually sync variables for production (with confirmation prompt)
eiscli svc variables sync myservice --env prod --apply

# Sync from a custom kubernetes folder location
eiscli svc variables sync myservice --env testing --kubernetes-path ./k8s --apply

# Auto-create missing environment without prompting
eiscli svc variables sync myservice --env testing --apply --auto-create-env

# Override inferred environment type when creating
eiscli svc variables sync myservice --env custom --env-type Staging --apply

# Auto-detect service from git repo
cd /path/to/service/repo
eiscli svc variables sync --env staging
```

**Template File Format:**

The `.env.template` files should contain variables in the format:

```bash
# Comments are ignored
KEY_NAME=${PLACEHOLDER_VALUE}
ANOTHER_KEY=${VALUE}
DATABASE_PASSWORD=${DB_PASS}  # Will be marked as secured
API_KEY=${API_KEY}            # Will be marked as secured
```

**Secured Variable Detection:**

Variables are automatically marked as secured if their name contains any of these keywords (case-insensitive):

- PASSWORD
- SECRET
- TOKEN
- API_KEY / APIKEY
- PRIVATE
- CREDENTIAL / CREDENTIALS
- AUTH
- Variables ending with `_KEY`

#### Ingress Controller Service Registration

```bash
eiscli svc ingress add --service <service-name> --env <environment> [--region <region>]
```

Automatically register a microservice in the API Gateway ingress controller configurations.

**Required Flags:**

- `-s, --service <name>`: Service name to register (e.g., `dunningservice`)
- `-e, --env <environment>`: Target environment (`test`, `dev`, `staging`, `prod`, `perf`)

**Optional Flags:**

- `-r, --region <region>`: Target region (default: `frankfurt`, supports: `frankfurt`, `zurich`)

**Features:**

- Adds three path mappings for your service (`/{service}`, `/{service}/api`, `/{service}/api-json`)
- Automatically excludes API documentation paths from Lambda authorization
- Updates all API ingress files in the target environment (e.g., `api_ingress.yaml`, `funk_api_ingress.yaml`)
- Detects already-configured services and skips them
- Identifies misconfigured services and prompts for manual intervention
- Provides comprehensive summary of changes

**Examples:**

```bash
# Add service to test environment in Frankfurt (default region)
eiscli svc ingress add --service myservice --env test

# Add service to production in Frankfurt
eiscli svc ingress add --service myservice --env prod --region frankfurt

# Add service to production in Zurich
eiscli svc ingress add --service myservice --env prod --region zurich

# Add service to staging
eiscli svc ingress add -s myservice -e staging
```

**Requirements:**

- Must be run from the **dist-orchestration repository root directory**
- Service uses port 80 (standard for all EIS services)
- For Zurich region, only `prod` environment is supported

See `INGRESS_COMMAND_USAGE.md` for detailed documentation and troubleshooting.

#### ECR Registry Management

```bash
eiscli svc ecr [service-name]
```

Check if ECR registry exists for a service and optionally create it in the correct AWS account.

**Auto-Detection:** If you run this command from within a git repository, the service name will be automatically detected from the git remote URL.

**AWS Profile Selection:** The command automatically selects the correct AWS profile based on the environment:

- **Production environments** (testing, test, production, production-zurich) → uses `default_profile` (default: "default")
- **Non-production environments** (staging, development) → uses `nonprod_profile` (default: "staging")

**Options:**

- `-e, --env <environment>`: Environment to check (default: testing)
- `-c, --create`: Create the repository if it doesn't exist (prompts for name)
- `-a, --all`: Check all environments (both production and non-production profiles)

**Features:**

- Displays repository details including URI, ARN, and creation date
- Shows clickable AWS Console links
- Interactive repository name prompt when creating (with service name as default)
- Supports AWS_PROFILE environment variable override
- Auto-detects service name from git repository

**Examples:**

```bash
# Auto-detect service and check testing environment (default)
cd /path/to/service/repo
eiscli svc ecr

# Check if ECR exists for a specific service in testing
eiscli svc ecr myservice --env testing

# Check staging environment (uses non-prod profile)
eiscli svc ecr myservice --env staging

# Check production environment
eiscli svc ecr myservice --env production

# Check all environments at once
eiscli svc ecr myservice --all

# Create repository if it doesn't exist (prompts for name)
eiscli svc ecr myservice --env testing --create

# Use custom AWS profile via environment variable
AWS_PROFILE=my-custom-profile eiscli svc ecr myservice --env testing
```

**AWS Configuration:**

Configure AWS profiles in `~/.aws/config`:

```ini
[default]
region = eu-central-1
output = json

[profile staging]
region = eu-central-1
output = json
```

Configure EIS CLI in `~/.eiscli/config.yaml` or `./config.yaml`:

```yaml
aws:
  default_profile: "default" # For production, testing, test, production-zurich
  nonprod_profile: "staging" # For staging, development
  region: "eu-central-1"
```

**Environment Variable Overrides:**

- `AWS_PROFILE`: Override the AWS profile for all environments
- `AWS_REGION`: Override the AWS region

**Requirements:**

- Valid AWS credentials configured for the specified profiles
- IAM permissions for ECR operations:
  - `ecr:DescribeRepositories` (to check if repository exists)
  - `ecr:CreateRepository` (to create new repositories)
  - `sts:GetCallerIdentity` (to get AWS account ID)

### Auto-Creating Deployment Environments

When listing or syncing variables, if the target deployment environment doesn't exist in Bitbucket, the CLI will prompt you to create it automatically.

**Interactive Mode (default):**

```bash
eiscli svc variables myservice --env Staging

# Output:
# Deployment environment 'Staging' not found in Bitbucket.
#
# Available environments:
#   - Test (Test)
#   - Production (Production)
#
# Would you like to create the 'Staging' environment? [y/N]: y
#
# Inferred environment type: Staging
# Is this correct? [Y/n]: y
#
# Creating deployment environment 'Staging' (type: Staging)...
# ✓ Environment created successfully! (UUID: {...})
```

**Auto-Create Mode:**

Use `--auto-create-env` flag to skip prompts (useful for CI/CD):

```bash
eiscli svc variables myservice --env Staging --auto-create-env
```

**Override Environment Type:**

Use `--env-type` to override the inferred environment type:

```bash
eiscli svc variables myservice --env CustomEnv --env-type Production
```

**Configuration File:**

Set auto-create behavior in `config.yaml`:

```yaml
deployment:
  auto_create_environments: true # Never prompt, always create
  default_environment_type: "Test" # Fallback type when can't infer
```

**Environment Type Inference:**

The CLI automatically infers environment types from names:

| Environment Name Pattern  | Inferred Type |
| ------------------------- | ------------- |
| Test, Testing, test       | Test          |
| Dev, Development          | Test          |
| Staging, Stage            | Staging       |
| Production, Prod, _-prod_ | Production    |

**Requirements:**

- Your Bitbucket app password must have **Deployments: Write** permission
- Without this permission, you'll see an error message with instructions

### AWS Configuration

The CLI uses AWS profiles to manage ECR registries in different accounts. Configure your AWS profiles in `~/.aws/config`:

**Profile Mapping:**

| Environment Pattern                          | AWS Profile                            | Use Case                 |
| -------------------------------------------- | -------------------------------------- | ------------------------ |
| testing, test, production, production-zurich | `default_profile` (default: "default") | Production workloads     |
| staging, development, dev                    | `nonprod_profile` (default: "staging") | Non-production workloads |

**Configuration Example:**

```yaml
aws:
  default_profile: "default"
  nonprod_profile: "staging"
  region: "eu-central-1"
```

**Environment Variables:**

- `AWS_PROFILE`: Override AWS profile for all operations
- `AWS_REGION`: Override AWS region (default: eu-central-1)

## Development Status

This CLI is currently under active development. The following features are planned:

- [x] Basic CLI structure with Cobra
- [x] `svc` command with subcommands
- [x] Bitbucket API integration
- [x] Pipeline status retrieval
- [x] Variable management (list deployment and repository variables)
- [x] Variable management (sync variables from Kubernetes templates to Bitbucket)
- [x] Variable management (auto-create missing deployment environments)
- [ ] Variable management (update, delete variables)
- [x] AWS ECR integration
- [x] Ingress controller service registration
- [ ] Service creation scaffolding

## Project Structure

```
.
├── main.go                    # Entry point
├── cmd/
│   ├── root.go               # Root command
│   ├── svc.go                # Service command
│   ├── svc_new.go            # New service subcommand
│   ├── svc_status.go         # Status subcommand
│   ├── svc_pipeline.go       # Pipeline subcommand (✓ implemented)
│   ├── svc_variables.go      # Variables subcommand (✓ implemented)
│   ├── svc_ingress.go        # Ingress registration subcommand (✓ implemented)
│   └── svc_ecr.go            # ECR registry subcommand (✓ implemented)
├── internal/
│   ├── bitbucket/
│   │   └── client.go         # Bitbucket API client wrapper
│   ├── config/
│   │   └── config.go         # Configuration management
│   └── aws/
│       └── ecr_client.go     # AWS ECR client
└── config.yaml.example        # Example configuration file
```

## Contributing

This tool is for internal EIS platform development.

## License

Proprietary - Cover42/EIS Platform
