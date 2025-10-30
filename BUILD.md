# Building EIS CLI with OAuth Credentials

This document explains how to build EIS CLI with OAuth credentials baked into the binary for distribution.

## Overview

The CLI supports two authentication methods:
1. **OAuth 2.0** (Recommended) - OAuth credentials can be injected at build time
2. **Basic Auth** (Legacy) - Username and app password loaded at runtime from config/env

## Build Methods

### 1. Development Build (No OAuth Injection)

For local development and testing:

```bash
make build
```

This builds the binary **without** OAuth credentials. Users will need to:
- Manually configure OAuth credentials in `~/.eiscli/config.yaml`, OR
- Set environment variables: `EISCLI_BITBUCKET_CLIENT_ID` and `EISCLI_BITBUCKET_CLIENT_SECRET`, OR
- Use legacy Basic Auth

### 2. Production Build (With OAuth Injection)

For distributing to end users:

```bash
# Set OAuth credentials
export BITBUCKET_OAUTH_CLIENT_ID="your-oauth-consumer-key"
export BITBUCKET_OAUTH_CLIENT_SECRET="your-oauth-consumer-secret"

# Build with credentials
make build-with-oauth
```

This injects OAuth credentials **into the binary at compile time**. Users can simply:
```bash
./eiscli auth login
# That's it! No config file needed.
```

### 3. Release Build (Optimized Binary)

For production releases (smaller binary size, optimized):

```bash
export BITBUCKET_OAUTH_CLIENT_ID="your-oauth-consumer-key"
export BITBUCKET_OAUTH_CLIENT_SECRET="your-oauth-consumer-secret"

make build-release
```

This creates an optimized binary with:
- OAuth credentials injected
- Debug symbols stripped (`-s -w`)
- Reproducible builds (`-trimpath`)
- Static binary (`CGO_ENABLED=0`)

## How It Works

### Build-Time Injection

OAuth credentials are injected using Go's `-ldflags -X` flag:

```bash
go build -ldflags "-X 'bitbucket.org/cover42/eiscli/internal/bitbucket.DefaultClientID=xxx' \
                    -X 'bitbucket.org/cover42/eiscli/internal/bitbucket.DefaultClientSecret=yyy'" \
         -o eiscli .
```

The values are set in these variables:
- `bitbucket.org/cover42/eiscli/internal/bitbucket.DefaultClientID`
- `bitbucket.org/cover42/eiscli/internal/bitbucket.DefaultClientSecret`

### Configuration Priority

The CLI loads configuration in this order (highest to lowest priority):

1. **Environment variables** (runtime) - `EISCLI_BITBUCKET_CLIENT_ID`, etc.
2. **Config file** (runtime) - `~/.eiscli/config.yaml`
3. **Build-time defaults** - Injected during compilation

This allows:
- End users to use the binary with no configuration
- Advanced users to override with env vars or config file
- Backward compatibility with legacy Basic Auth

### Legacy Basic Auth

Basic Auth credentials (`username` and `app_password`) are **always** loaded at runtime:
- From environment: `EISCLI_BITBUCKET_USERNAME`, `EISCLI_BITBUCKET_APP_PASSWORD`
- From config file: `~/.eiscli/config.yaml`

They are **never** injected at build time to maintain security.

## Bitbucket Pipelines

The CI/CD pipeline automatically builds release binaries with OAuth credentials.

### Setup Repository Variables

In Bitbucket: **Repository Settings → Pipelines → Repository variables**

Add these **secured** variables:
- `OAUTH_CLIENT_ID` = Your OAuth consumer key
- `OAUTH_CLIENT_SECRET` = Your OAuth consumer secret

### Pipeline Behavior

- **Pull Requests**: Builds without OAuth (for testing)
- **Main/Master Branch**: Builds with OAuth credentials
- **Tags (`v*`)**: Builds release binary with OAuth credentials

The pipeline exports these variables:
```bash
export BITBUCKET_OAUTH_CLIENT_ID=$OAUTH_CLIENT_ID
export BITBUCKET_OAUTH_CLIENT_SECRET=$OAUTH_CLIENT_SECRET
make build-release
```

## Local Development with OAuth

### Option 1: Environment Variables

```bash
export BITBUCKET_OAUTH_CLIENT_ID="your-oauth-consumer-key"
export BITBUCKET_OAUTH_CLIENT_SECRET="your-oauth-consumer-secret"
make build-with-oauth
```

### Option 2: .env File (Recommended)

Create `.env` file (already in `.gitignore`):

```bash
# .env
BITBUCKET_OAUTH_CLIENT_ID=your-oauth-consumer-key
BITBUCKET_OAUTH_CLIENT_SECRET=your-oauth-consumer-secret
```

Load and build:
```bash
source .env
make build-with-oauth
```

Or:
```bash
export $(cat .env | xargs)
make build-with-oauth
```

## Verification

### Verify OAuth Injection

```bash
make verify-oauth-build
```

This checks if OAuth credentials were successfully injected into the binary.

### Test Without Config File

```bash
# Temporarily rename config
mv ~/.eiscli/config.yaml ~/.eiscli/config.yaml.backup

# Test auth status (should use build-time credentials)
./eiscli auth status

# Restore config
mv ~/.eiscli/config.yaml.backup ~/.eiscli/config.yaml
```

### Check Binary Size

```bash
ls -lh ./eiscli
file ./eiscli
```

Release builds are typically 10-15MB (depending on features).

## Distribution

### For End Users

Distribute the OAuth-injected binary. Users only need to:

1. **Download** the binary
2. **Run** `./eiscli auth login`
3. **Use** any command

No configuration required!

### For Developers (Team)

Two options:

**Option A: Use Distributed Binary** (Recommended)
- Download the OAuth-injected binary from releases
- Run `eiscli auth login`
- Each developer gets their own OAuth token

**Option B: Build from Source**
- Clone repository
- Run `make build` (no OAuth needed for development)
- Configure OAuth or use Basic Auth in `~/.eiscli/config.yaml`

## Security Considerations

### ✅ Safe to Include in Binary

OAuth **consumer credentials** (client_id, client_secret):
- Considered "public" for CLI applications
- Industry standard (GitHub CLI, Heroku CLI, etc.)
- Required to initiate OAuth flow
- Cannot be used without user authorization

### ⚠️ Never Include in Binary

User **access tokens**:
- Personal to each user
- Stored in `~/.eiscli/tokens.json` (0600 permissions)
- Generated per-user via OAuth flow
- Automatically refreshed every hour

Basic Auth **credentials** (username, password):
- Personal user credentials
- Loaded at runtime from config/env
- Never baked into binary

## Troubleshooting

### Build fails with "OAuth credentials not set"

```bash
# Set the environment variables
export BITBUCKET_OAUTH_CLIENT_ID="your-key"
export BITBUCKET_OAUTH_CLIENT_SECRET="your-secret"
make build-with-oauth
```

### CLI still asks for config file

The binary was built without OAuth injection. Rebuild with:
```bash
make build-with-oauth
```

### Want to override build-time credentials

Set environment variables at runtime:
```bash
export EISCLI_BITBUCKET_CLIENT_ID="different-key"
export EISCLI_BITBUCKET_CLIENT_SECRET="different-secret"
./eiscli auth login
```

Or use a config file:
```yaml
# ~/.eiscli/config.yaml
bitbucket:
  client_id: "different-key"
  client_secret: "different-secret"
  use_oauth: true
```

## Makefile Targets

```bash
make help                 # Show all available targets
make build                # Development build (no OAuth)
make build-with-oauth     # Build with OAuth credentials
make build-release        # Optimized release build
make verify-oauth-build   # Verify OAuth injection
make install              # Install to GOPATH/bin
make install-with-oauth   # Install with OAuth
make test                 # Run tests
make clean                # Clean build artifacts
make dist                 # Create distribution package
```

## Next Steps

1. Set up repository variables in Bitbucket Pipelines
2. Push to main branch to trigger OAuth-injected build
3. Download artifact from pipeline
4. Distribute to team

Happy building! 🚀

