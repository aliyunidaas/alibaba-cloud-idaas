# Agent Guide for Alibaba Cloud IDaaS CLI

This guide is designed for AI agents and developers working with the Alibaba Cloud IDaaS CLI codebase. It provides essential context, conventions, and patterns to ensure consistent and correct contributions.

## Table of Contents

- [Project Overview](#project-overview)
- [Codebase Structure](#codebase-structure)
- [Key Technologies](#key-technologies)
- [Development Workflow](#development-workflow)
- [Common Tasks](#common-tasks)
- [Testing](#testing)
- [Debugging](#debugging)
- [Security Considerations](#security-considerations)

---

## Project Overview

**Purpose**: CLI tool for Alibaba Cloud IDaaS (Identity as a Service) integration with cloud CLIs (Alibaba Cloud, AWS, Terraform, etc.)

**Key Features**:
- Fetch STS tokens via OIDC/OAuth2
- Support multiple authentication flows (Device Code, Client Credentials)
- External signer support (YubiKey, PKCS#11)
- AKless authentication
- OpenClaw secret provider integration

---

## Codebase Structure

```
alibaba-cloud-idaas/
├── main.go                      # Entry point
├── commands/                    # CLI command implementations
│   ├── agent/                   # Agent subcommands (for AI/automation use)
│   │   ├── command.go           # Agent command router
│   │   ├── access_token/        # Get agent access token
│   │   ├── get_secret/          # Fetch secrets from credential service
│   │   ├── put_secret/          # Store secrets (API keys) to credential service
│   │   ├── token_exchange/      # Token exchange (RFC 8693)
│   │   └── util/                # Shared utilities for agent commands
│   ├── clean_cache/             # Clean cached tokens
│   ├── common/                  # Shared utilities for commands
│   ├── execute/                 # Execute commands with credentials
│   ├── fetch_token/             # Fetch STS/OIDC tokens
│   ├── openclaw_secret/         # OpenClaw secret provider
│   ├── qr/                      # QR code generation
│   ├── serve/                   # Local HTTP server for token serving
│   ├── show_cache/              # Display cached tokens
│   ├── show_profile/            # List configured profiles
│   ├── show_signer_public_key/  # Display signer public key
│   ├── show_token/              # Display token info
│   ├── validate_jwt/            # JWT token validation (RS256)
│   └── version/                 # Version information
├── config/                      # Configuration parsing and validation
│   ├── config.go                # Config struct and loading
│   ├── digest.go                # Config digest utilities
│   ├── parse.go                 # Config parsing
│   └── signer.go                # Signer configuration
├── idaaslog/                    # Logging package (see idaaslog/README.md)
├── oidc/                        # OIDC/OAuth2 token providers
│   ├── oidc_common.go           # Common OIDC utilities
│   ├── oidc_device_code.go      # Device Code Flow (RFC 8628)
│   ├── oidc_idtoken_bearer.go   # ID Token bearer
│   ├── oidc_pkcs7_bearer.go     # PKCS#7 bearer token
│   ├── oidc_rfc7523.go          # RFC 7523 JWT assertion
│   └── oidc_x509_jwt_bearer.go  # X.509 JWT bearer
├── idp/                         # Identity provider clients
│   ├── client_credentials.go
│   ├── client_device_code.go
│   ├── client_oidc_common.go
│   └── client_credentials_*.go  # Various credential flow implementations
├── signer/                      # External signer implementations
│   ├── signer.go                # Signer interface
│   ├── external/                # External process signer
│   ├── key_file/                # Key file signer
│   ├── pkcs11/                  # PKCS#11 signer (conditional)
│   └── yubikey_piv/             # YubiKey PIV signer (conditional)
├── cloud/                       # Cloud provider integrations
│   ├── cloud_sts.go             # STS common interface
│   ├── alibaba_cloud/           # Alibaba Cloud STS
│   ├── aws/                     # AWS STS
│   ├── cloud_account/           # Cloud Account Token (AKless)
│   ├── cloud_common/            # Shared cloud utilities
│   ├── credential/              # Credential fetching
│   ├── oidc/                    # Cloud OIDC integration
│   └── openclaw/                # OpenClaw integration
├── privateca/                   # Private CA utilities
├── constants/                   # Constants and environment variables
├── utils/                       # Utility functions
│   ├── features/                # Feature flags (conditional compilation)
│   └── *.go                     # Various utilities
└── tools/                       # Development tools
```

---

## Key Technologies

### Language & Framework
- **Go 1.25+** (check `go.mod` for exact version)
- Standard library only (no web frameworks)

### Dependencies
Key third-party packages:
```go
github.com/pkg/errors            // Error handling with stack traces
github.com/urfave/cli/v2         // CLI argument parsing
github.com/skip2/go-qrcode       // QR code generation
golang.org/x/crypto/ssh/terminal // Terminal operations
github.com/itchyny/gojq          // JSON query
github.com/miekg/pkcs11          // PKCS#11 support (conditional)
github.com/go-piv/piv-go         // YubiKey PIV (conditional)
```

### Build Tags
Conditional compilation for optional features:
```bash
# Default build (includes all features)
go build

# Disable PKCS#11 and YubiKey support
go build -tags disable_pkcs11,disable_yubikey_piv
```

---

## Development Workflow

### Setup
```bash
# Clone and enter directory
cd /path/to/alibaba-cloud-idaas

# Verify dependencies
go mod download
go mod tidy

# Build
go build

# Run
./alibaba-cloud-idaas --help
```

### Common Build Commands
```bash
# Build with all features
go build

# Build without hardware signer support
go build -tags disable_pkcs11,disable_yubikey_piv

# Install to $GOPATH/bin
go install

# Cross-compile for other platforms
GOOS=linux GOARCH=amd64 go build
GOOS=darwin GOARCH=arm64 go build
GOOS=windows GOARCH=amd64 go build
```

### Code Style
- Follow Go conventions (`gofmt`, `golint`)
- Use `errors.Wrap()` from `github.com/pkg/errors` for stack traces
- Prefix error messages with context: `"Failed to fetch token: %v"`
- Use `idaaslog` package for logging (see `idaaslog/README.md`)

---

### Adding a New Config Profile Type

1. **Define struct** in `config/profile.go`:
```go
type MyProfile struct {
    Endpoint string `json:"endpoint"`
    // ... fields
}
```

2. **Add to `Profile` union type**:
```go
type Profile struct {
    // ... existing types
    MyProfile *MyProfile `json:"my_profile,omitempty"`
}
```

3. **Implement resolver** in appropriate package

### Adding Log Statements

Use appropriate log level:
```go
idaaslog.Debug.PrintfLn("Debug: %v", data)        // Development only
idaaslog.Info.PrintfLn("Info: operation started") // Important events
idaaslog.Warn.PrintfLn("Warning: deprecated")     // Recoverable issues
idaaslog.Error.PrintfLn("Error: %v", err)         // Failures
idaaslog.Unsafe.PrintfLn("Token: %s", token)      // Sensitive data (requires UNSAFE_DEBUG)
```

### Error Handling Pattern

```go
func doSomething() error {
    result, err := someOperation()
    if err != nil {
        return errors.Wrap(err, "failed to do something")
    }
    
    if !isValid(result) {
        return errors.New("invalid result from operation")
    }
    
    return nil
}

// In caller
if err != nil {
    idaaslog.Error.PrintfLn("%s", idaaslog.DumpError(err))
    return err
}
```

---

## Testing

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package
go test ./commands/

# Verbose output
go test -v ./...
```

### Writing Tests

Place tests in same package with `_test.go` suffix:
```go
// commands/fetch_token_test.go
package commands

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestFetchTokenValidProfile(t *testing.T) {
    // Setup
    config := loadTestConfig()
    
    // Execute
    result, err := fetchToken(config, "test-profile")
    
    // Verify
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotEmpty(t, result.AccessToken)
}
```

### Test Fixtures

Store test data in `testdata/` directory:
```
commands/
├── fetch_token.go
├── fetch_token_test.go
└── testdata/
    ├── valid_config.json
    └── invalid_config.json
```

---

## Debugging

### Enable Debug Logging

```bash
# Enable console output
export ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT=true

# Enable unsafe/sensitive data logging
export ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG=true

# Run command
./alibaba-cloud-idaas fetch-token --profile test
```

### Log File Location

```
$HOME/.cloud_idaas/cloud-cli/__log/
```

Find latest log:
```bash
ls -lt ~/.cloud_idaas/cloud-cli/__log/ | head -2
```

### Common Debug Patterns

```go
// Add temporary debug logging
idaaslog.Debug.PrintfLn("Variable value: %+v", variable)

// Dump full error with stack trace
idaaslog.Error.PrintfLn("Error occurred:\n%s", idaaslog.DumpError(err))

// Trace function entry/exit
func myFunction() error {
    idaaslog.Debug.PrintfLn("Entering myFunction")
    defer idaaslog.Debug.PrintfLn("Exiting myFunction")
    // ...
}
```

---

## Security Considerations

### ⚠️ Critical Rules

1. **Never log sensitive data without UNSAFE level**
   ```go
   // ❌ BAD - logs token at INFO level
   idaaslog.Info.PrintfLn("Token: %s", token)
   
   // ✅ GOOD - uses UNSAFE level (requires explicit enablement)
   idaaslog.Unsafe.PrintfLn("Token: %s", token)
   ```

2. **Always validate config before use**
   ```go
   if err := profile.Validate(); err != nil {
       return errors.Wrap(err, "invalid profile configuration")
   }
   ```

3. **Sanitize error messages**
   ```go
   // ❌ BAD - exposes internal details
   return errors.Wrap(err, fmt.Sprintf("Failed with token %s", token))
   
   // ✅ GOOD - generic message
   return errors.Wrap(err, "failed to fetch token")
   ```

4. **Use constant-time comparisons for secrets**
   ```go
   import "crypto/subtle"
   
   if subtle.ConstantTimeCompare([]byte(a), []byte(b)) != 1 {
       return errors.New("invalid credentials")
   }
   ```

### Environment Variables

| Variable                                   | Purpose                       | Security Impact                        |
|--------------------------------------------|-------------------------------|----------------------------------------|
| `ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG`         | Enable sensitive data logging | ⚠️ HIGH - Never enable in production   |
| `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT` | Copy logs to stderr           | ⚠️ MEDIUM - May expose data in console |
| `ALIBABA_CLOUD_IDAAS_PKSC11_PIN`           | PKCS#11 PIN                   | ⚠️ HIGH - Sensitive credential         |
| `ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN`          | YubiKey PIN                   | ⚠️ HIGH - Sensitive credential         |

### Code Review Checklist

- [ ] No hardcoded secrets/tokens
- [ ] Sensitive data uses `LogLevelUnsafe`
- [ ] Errors don't leak internal state
- [ ] Config validation is comprehensive
- [ ] External inputs are sanitized
- [ ] File permissions are secure (0600 for credentials)
- [ ] No unnecessary debug logging in production paths

---

## Agent-Specific Patterns

### When Modifying Existing Code

1. **Read surrounding code first** - match existing patterns
2. **Check for existing tests** - understand expected behavior
3. **Use same error wrapping style** - `errors.Wrap(err, "context")`
4. **Follow logging conventions** - check `idaaslog/README.md`

### When Adding Features

1. **Check if feature exists elsewhere** - search codebase first
2. **Match existing architecture** - don't introduce new patterns unnecessarily
3. **Add tests** - ensure new code is covered
4. **Update documentation** - `README.md`, config examples, etc.

### Common Search Patterns

```bash
# Find all usages of a function
grep -r "fetchToken" --include="*.go" .

# Find config struct definitions
grep -r "type.*Config struct" --include="*.go" .

# Find all profile types
grep -r "json:\"" config/profile.go

# Find error handling patterns
grep -r "errors.Wrap" --include="*.go" .
```

### Git Workflow

```bash
# Check status
git status

# Stage changes
git add commands/my_command.go

# Commit with descriptive message
git commit -m "feat: add my-command for XYZ functionality

- Implements new command in commands/my_command.go
- Adds validation for input parameters
- Includes unit tests with 90% coverage"

# Push
git push origin branch-name
```

---

## Troubleshooting

### Build Errors

**Disable PKCS#11**
```bash
# Build without PKCS#11 support
go build -tags disable_pkcs11
```

**Disable YubiKey PIV**
```bash
# Build without YubiKey support
go build -tags disable_yubikey_piv
```

**Token fetch fails silently**
```bash
# Enable debug logging
export ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT=true
export ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG=true
```

---

## Resources

- [Main README](README.md) - User documentation
- [idaaslog README](idaaslog/README.md) - Logging package guide
- [Go Documentation](https://pkg.go.dev/std) - Standard library
- [urfave/cli Documentation]([https://github.com/spf13/cobra](https://cli.urfave.org/v2/getting-started/)) - CLI framework

---

## Quick Reference

### Common Imports
```go
import (
    "github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
    "github.com/aliyunidaas/alibaba-cloud-idaas/config"
    "github.com/pkg/errors"
    "github.com/urfave/cli/v2"
)
```

### Error Handling Template
```go
if err != nil {
    idaaslog.Error.PrintfLn("%s", idaaslog.DumpError(err))
    return errors.Wrap(err, "context message")
}
```

### Logging Template
```go
idaaslog.Info.PrintfLn("Operation started: %s", name)
defer idaaslog.Info.PrintfLn("Operation completed: %s", name)
```

---

*Last updated: 2026-04-14*
