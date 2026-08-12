# alibaba-cloud-idaas Command Reference

**English** | [中文](command-reference_zh.md)

> Version: v0.2.0-beta

## Command Overview

| Command | Purpose | Status |
|------|------|------|
| `onboard` | Discover instance + login + list roles + generate CLI tool config | ✅ |
| `login` | Device-code login to IDaaS and cache the access token | ✅ |
| `fetch-token` | Fetch credentials and output JSON (credential_process contract) | ✅ |
| `serve` | Start a local HTTP credential service (for SDK credentials_uri) | ✅ |
| `show` | Query subcommand family (profiles/roles/cache/token/status/instance/signer-key) | ✅ |
| `show-token` | Display current credentials in human-readable form | ✅ |
| `show-profiles` | List configured profiles | ✅ |
| `show-cache` | Display cache entries | ✅ |
| `clean-cache` | Remove all caches | ✅ |
| `logout` | Remove cached tokens (profile config is preserved) | ✅ |
| `status` | Display current profile / login state / serve daemon state | ✅ |
| `execute` | Inject environment variables and run a command | ✅ |
| `show-signer-public-key` | Display the signer public key | ✅ |
| `qr` | Generate a QR code | ✅ |
| `validate-jwt` | Validate a JWT (RS256 only) | ✅ |
| `openclaw-secret` | Get an OpenClaw secret | ✅ |
| `agent` | Agent subcommand family (see below) | ✅ |

---

## Top-level Commands

### `onboard`

Zero-config onboarding: discover instance → login (triggers `login` automatically) → list assumable roles → generate config for CLI tools.

```shell
# First time (instance + client-id required)
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx --target aliyun-cli,aws-cli

# Subsequent runs (with an existing profile, --instance / --client-id are inferred automatically)
alibaba-cloud-idaas onboard
alibaba-cloud-idaas onboard --target aws-cli

# Do not write CLI config (generate the broker profile only)
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx --target none
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-i` | IDaaS instance domain (inferred from an existing profile when omitted) | — | First run |
| `--target` | — | Target CLI tools (comma-separated: `aliyun-cli`/`aws-cli`/`tencentcloud-cli`/`mcp`/`none`) | all applicable | No |
| `--prefix` | — | Prefix for generated profile names | `aliyun` | No |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` | No |
| `--vpc` | — | Prefer the VPC endpoint | `false` | No |
| `--client-id` | — | Broker client application ID (passed through to `login`) | explicit > existing profile > discovery `cli_client_id` > `iap_developer` | First run |
| `--force-new` | `-N` | Force device-code login (passed through to `login`) | `false` | No |

### `login`

Device-code login to IDaaS: obtain an access token and cache it. Writes neither profiles nor CLI config.

```shell
# First login (instance + client-id required)
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx

# Refresh login (issuer + scope + client-id are read from the profile, so --instance / --client-id are not needed)
alibaba-cloud-idaas login --profile aliyun-readonly

# Refresh token has also expired (instance + client-id must be supplied again)
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx --force-new

# Custom scope
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx --scope "urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential urn:cloud:idaas:pam|credential:obtain"
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-i` | Instance domain (first-login mode; inferred from an existing profile when omitted) | — | One of `--instance` / `--profile` |
| `--profile` | `-p` | Existing profile name (refresh mode; issuer + scope + client-id are read from the profile) | — | One of `--instance` / `--profile` |
| `--scope` | `-s` | Space-separated `audience\|scope` combinations | `urn:cloud:idaas:pam\|.all` | No |
| `--client-id` | — | Broker client application ID (read from the profile in `--profile` mode) | must be explicit in `--instance` mode | In `--instance` mode |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` | No |
| `--force-new` | `-N` | Ignore the cache and log in again | `false` | No |

### `fetch-token`

Fetch credentials and output JSON (aliyun-cli External / aws-cli credential_process contract).

```shell
alibaba-cloud-idaas fetch-token --profile aliyun-readonly
alibaba-cloud-idaas fetch-token --profile aws-role --format raw
alibaba-cloud-idaas fetch-token --profile oidc1 --oidc-field access_token
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS profile | `IDAAS_PROFILE` or `current_profile` |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--format` | `-f` | STS output format: `aliyuncli`/`ossutilv2`/`raw` | `aliyuncli` |
| `--oidc-field` | — | OIDC token field: `id_token`/`access_token` | — |
| `--oidc-format` | — | OIDC format: `type1`/`type2` | `type1` |
| `--output` | `-o` | Write to a file | stdout |
| `--force-new` | `-N` | Force refresh, ignoring all caches | `false` |
| `--force-new-cloud-token` | — | Force refresh of the cloud credential (lower-level cache) | `false` |

### `serve`

Start a local HTTP credential service for the Alibaba Cloud SDK `credentials_uri`.

```shell
alibaba-cloud-idaas serve --ssrf-token my-secret-token
alibaba-cloud-idaas serve --port 8080 --ssrf-token my-secret-token
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--port` | `-p` | Listening port | `1127` |
| `--ssrf-token` | — | SSRF token (header `X-Aliyun-Parameters-Secrets-Token` or query `__ssrf_token`) | — |
| `--unsafe-listen-host` | — | Listening address (defaults to `127.0.0.1`, can be set to `0.0.0.0`) | `127.0.0.1` |
| `--unsafe-disable-ssrf` | — | Disable SSRF validation | `false` |

**Endpoints**:
- `GET /cloud_token?profile=X&__ssrf_token=T` → STS JSON (`credentials_uri` contract)
- `GET /version` → version information

### `show`

Query subcommand family. Never modifies configuration.

#### `show profiles`

List configured profiles.

```shell
alibaba-cloud-idaas show profiles
alibaba-cloud-idaas show profiles -f aliyun
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile-filter` | `-f` | Filter by profile name | — |
| `--no-color` | — | Colorless output | `false` |

#### `show roles`

List the cloud roles the current user can assume (does not generate profiles).

```shell
# First time (instance + client-id required)
alibaba-cloud-idaas show roles --instance acme.aliyunidaas.com --client-id app_xxx

# Subsequent runs (inferred automatically from an existing profile)
alibaba-cloud-idaas show roles

# JSON output
alibaba-cloud-idaas show roles --json
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-i` | IDaaS instance domain (inferred from an existing profile when omitted) | — | First run |
| `--scope` | `-s` | Scope | `urn:cloud:idaas:pam\|.all` | No |
| `--client-id` | — | Broker client application ID (inferred from an existing profile when omitted) | — | First run |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` | No |
| `--vpc` | — | Prefer the VPC endpoint | `false` | No |
| `--json` | — | Machine-readable JSON output | `false` | No |

#### `show status`

Display current profile / provider / instance / serve daemon state.

```shell
alibaba-cloud-idaas show status
alibaba-cloud-idaas show status --profile aliyun-readonly
alibaba-cloud-idaas show status --json
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | Target profile | `IDAAS_PROFILE` or `current_profile` |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--json` | — | Machine-readable JSON output | `false` |

#### `show instance`

Display instance discovery information.

```shell
alibaba-cloud-idaas show instance --instance acme.aliyunidaas.com
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-d` | IDaaS instance domain | — | ✅ |
| `--vpc` | — | Prefer the VPC endpoint | `false` | No |

#### `show cache` / `show token` / `show signer-key`

Placeholder subcommands (currently delegate to the legacy top-level commands `show-cache` / `show-token` / `show-signer-public-key`).

### `show-token`

Display current credentials in human-readable form (colored output).

```shell
alibaba-cloud-idaas show-token --profile aliyun-readonly
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS profile | `IDAAS_PROFILE` or `current_profile` |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--oidc-field` | — | OIDC token field | — |
| `--no-color` | — | Colorless output | `false` |
| `--force-new` | `-N` | Force refresh | `false` |
| `--force-new-cloud-token` | — | Force refresh of the cloud credential | `false` |

### `show-profiles`

List configured profiles.

```shell
alibaba-cloud-idaas show-profiles
alibaba-cloud-idaas show-profiles --profile-filter aliyun
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile-filter` | `-p` | Filter by profile name | — |
| `--no-color` | — | Colorless output | `false` |

### `show-cache`

Display cache entries.

```shell
alibaba-cloud-idaas show-cache
alibaba-cloud-idaas show-cache --category oidc_token --name app_xxx
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--category` | `-c` | Cache category (`oidc_token`/`cloud_token`/`token_response`) | all |
| `--name` | `-n` | Filter by cache name | — |

> Note: here `-c` is the short form of `--category`, **not** `--config` as in other commands. `--name` only takes effect together with `--category`.

### `clean-cache`

Remove all caches.

```shell
alibaba-cloud-idaas clean-cache
```

No flags.

### `logout`

Remove cached tokens while preserving profile configuration.

```shell
alibaba-cloud-idaas logout                          # remove all caches
alibaba-cloud-idaas logout --profile aliyun-readonly # remove one profile
alibaba-cloud-idaas logout --profile aliyun-readonly --dry-run  # preview
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | Profile to clear (clears everything when omitted) | — |
| `--dry-run` | — | Show what would be removed without deleting | `false` |

### `status`

Display current profile, login state and serve daemon state.

```shell
alibaba-cloud-idaas status
alibaba-cloud-idaas status --profile aliyun-readonly
alibaba-cloud-idaas status --json
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | Target profile | `IDAAS_PROFILE` or `current_profile` |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--json` | — | Machine-readable JSON output | `false` |

### `execute`

Inject environment variables and run a command.

```shell
alibaba-cloud-idaas execute --profile aliyun-readonly --env-region cn-hangzhou aliyun oss ls
alibaba-cloud-idaas execute --profile aliyun-readonly bash
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS profile | `IDAAS_PROFILE` or `current_profile` |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--env-region` | `-R` | Set the region environment variables | — |
| `--force-new` | `-N` | Force refresh | `false` |
| `--force-new-cloud-token` | — | Force refresh of the cloud credential | `false` |
| `--show-token` | — | Display credentials before running | `false` |

### `show-signer-public-key`

Display the signer public key.

```shell
alibaba-cloud-idaas show-signer-public-key --profile aliyun3
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS profile | — |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |

### `qr`

Generate a QR code.

```shell
alibaba-cloud-idaas qr --content "https://example.com"
alibaba-cloud-idaas qr --content "https://example.com" --small
```

| Flag | Description | Default |
|------|------|--------|
| `--content` | QR code content | — |
| `--small` | Small-size QR code | `false` |

### `validate-jwt`

Validate a JWT (RS256 only).

```shell
alibaba-cloud-idaas validate-jwt --token "eyJhbGciOi..."
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--token` | `-t` | JWT token | — | ✅ |

### `openclaw-secret`

Get an OpenClaw secret.

```shell
alibaba-cloud-idaas openclaw-secret --profile agent1
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS profile | — |
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--force-new` | `-N` | Force refresh | `false` |

---

## agent Subcommands

### `agent access-token`

Get an agent access token.

```shell
alibaba-cloud-idaas agent access-token --profile agent1
alibaba-cloud-idaas agent access-token --profile agent1 --scope "urn:cloud:idaas:pam|.all"
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile` | `-p` | IDaaS profile | — |
| `--scope` | `-s` | Scope, in the form `audience\|scope-value` | read from config |
| `--force-new` | `-N` | Force refresh | `false` |

### `agent get-secret`

Get a secret.

```shell
alibaba-cloud-idaas agent get-secret --profile agent1 --name default_model
alibaba-cloud-idaas agent get-secret --profile agent1 --name default_model --json-query .default_model.value.apiKeyContent.apiKey
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile` | `-p` | IDaaS profile | — |
| `--scope` | `-s` | Scope | `urn:cloud:idaas:pam\|credential:obtain` |
| `--json-query` | `-q` | JSON query expression | — |
| `--name` | `-n` | Secret name (may be repeated) | — |
| `--raw` | — | Output the raw response | `false` |
| `--string-raw` | — | Output the raw JSON string | `false` |
| `--force-new` | `-N` | Force refresh | `false` |

### `agent put-secret`

Store a secret.

```shell
alibaba-cloud-idaas agent put-secret --profile agent1 --name my-key --value "sk-xxx"
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` | No |
| `--profile` | `-p` | IDaaS profile | — | No |
| `--scope` | `-s` | Scope | `urn:cloud:idaas:pam\|credential:manage` | No |
| `--name` | `-n` | Secret name | — | ✅ |
| `--display-name` | — | Display name | same as `--name` | No |
| `--value` | — | Secret value | — | ✅ |

### `agent decrypt-secret`

Decrypt a secret.

```shell
alibaba-cloud-idaas agent decrypt-secret --profile agent1 --name default_model --ciphertext "encrypted..."
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` | No |
| `--profile` | `-p` | IDaaS profile | — | No |
| `--scope` | `-s` | Scope | `urn:cloud:idaas:pam\|credential:decrypt` | No |
| `--name` | `-n` | Credential identifier | — | ✅ |
| `--ciphertext` | — | Ciphertext | — | ✅ |

### `agent token-exchange`

Token Exchange (RFC 8693).

```shell
alibaba-cloud-idaas agent token-exchange --profile agent1 --subject-token "eyJ..." 
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--config` | `-c` | Config file path | `~/.aliyun/alibaba-cloud-idaas.json` | No |
| `--profile` | `-p` | IDaaS profile | — | No |
| `--scope` | `-s` | Scope | — | No |
| `--subject-token-type` | `-T` | Subject token type | `urn:ietf:params:oauth:token-type:access_token` | No |
| `--subject-token` | `-S` | Subject token | — | ✅ |

---

## Environment Variables

| Variable | Description |
|------|------|
| `IDAAS_PROFILE` | Default profile (precedence: `--profile` > `IDAAS_PROFILE` > `current_profile`) |
| `ALIBABA_CLOUD_IDAAS_USER_AGENT` | User-Agent for OIDC requests |
| `ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG` | Write sensitive data to the log |
| `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT` | Copy the log to stderr |
| `ALIBABA_CLOUD_IDAAS_PKSC11_PIN` | PKCS#11 PIN |
| `ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN` | YubiKey PIV PIN |

---

## Configuration Files

| File | Purpose |
|------|------|
| `~/.aliyun/alibaba-cloud-idaas.json` | Broker profile config (provider type + parameters) |
| `~/.aliyun/config.json` | aliyun-cli External profile (generated by `onboard`) |
| `~/.aws/config` | aws-cli credential_process profile (generated by `onboard`) |

The config file is selected as follows: `~/.aliyun/alibaba-cloud-idaas.json` takes priority when it exists, otherwise `~/.cloud_idaas/idaas-cli.json` is used.

The cache directory follows the config file location (encrypted storage for oidc_token / cloud_token / token_response):

| Config file in use | Cache directory |
|------|------|
| `~/.aliyun/alibaba-cloud-idaas.json` | `~/.aliyun/alibaba-cloud-idaas/` |
| `~/.cloud_idaas/idaas-cli.json` | `~/.cloud_idaas/cloud-cli/` |

---

## Build

```shell
go build -o alibaba-cloud-idaas .

# Disable hardware signers
go build -tags disable_pkcs11,disable_yubikey_piv

# Cross compile
GOOS=darwin  GOARCH=arm64 go build -o alibaba-cloud-idaas .
GOOS=linux   GOARCH=amd64 go build -o alibaba-cloud-idaas .
GOOS=windows GOARCH=amd64 go build -o alibaba-cloud-idaas.exe .

# Install to PATH
go install
```
