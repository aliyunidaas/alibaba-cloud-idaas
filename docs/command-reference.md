# alibaba-cloud-idaas Command Reference

**English** | [中文](command-reference_zh.md)

> Version: v0.2.5-beta (development, pending release; check the installed --help).

## Command Overview

| Command | Purpose | Status |
|------|------|------|
| `onboard` | Discover instance + login + list roles + generate CLI tool config | ✅ |
| `configure switch` | Explicitly switch the Broker default profile only | ✅ |
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
# First time (instance required; client-id optional)
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
| `--instance` | `-i` | IDaaS instance domain (inferred from an existing profile when omitted) | — | When not inferable |
| `--target` | — | Target CLI tools (comma-separated: `aliyun-cli`/`aws-cli`/`tencentcloud-cli`/`none`) | all applicable | No |
| `--aws-region` | — | Explicit AWS region override; must match the role ARN partition. Compatible existing regions are retained when omitted | by partition | No |
| `--result-file` | — | Write credential-free JSON to a new .json file, separate from login prompts; never overwrite an old file | disabled | No |
| `--prefix` | — | Prefix for generated profile names | vendor-based (`aliyun`/`aws`/`tencentcloud`) | No |
| `--config` | `-c` | Config file path | auto (see Configuration Files) | No |
| `--vpc` | — | Prefer the VPC endpoint | `false` | No |
| `--client-id` | — | Broker client application ID (passed through to `login`) | explicit > matching selected/unambiguous profile > `iap_cloud_idaas_cli` | No |
| `--force-new` | `-N` | Force device-code login (passed through to `login`) | `false` | No |

### `configure switch`

Explicitly switch the Broker current_profile; cloud CLI defaults and other configuration are preserved.

```shell
alibaba-cloud-idaas configure switch --profile <name> --config <absolute-path>
```

| Flag | Alias | Description | Default | Required |
| --- | --- | --- | --- | --- |
| `--profile` | `-p` | Existing Broker profile | — | Yes |
| `--config` | `-c` | Broker configuration path | auto (see Configuration Files) | No |

### `login`

Device-code login to IDaaS: obtain an access token and cache it. Writes neither profiles nor CLI config.

```shell
# First login (instance required; client-id optional)
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx

# Refresh login (issuer + scope + client-id are read from the profile, so --instance / --client-id are not needed)
alibaba-cloud-idaas login --profile aliyun-readonly

# When reauthorization is confirmed necessary, retain the selected profile/config
alibaba-cloud-idaas login --profile aliyun-readonly --config <absolute-path> --force-new

# Custom scope
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx --scope "urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential urn:cloud:idaas:pam|credential:obtain"
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-i` | Instance domain (first-login mode; mutually exclusive with --profile) | — | One of `--instance` / `--profile` |
| `--profile` | `-p` | Existing profile name (refresh mode; issuer + scope + client-id are read from the profile) | — | One of `--instance` / `--profile` |
| `--scope` | `-s` | Space-separated `audience\|scope` combinations | `urn:cloud:idaas:pam\|.all offline_access` | No |
| `--client-id` | — | Broker client application ID (read from the profile in `--profile` mode) | explicit > selected/matching profile > `iap_cloud_idaas_cli` | No |
| `--config` | `-c` | Config file path | auto (see Configuration Files) | No |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
| `--profile-filter` | `-f` | Filter by profile name | — |
| `--no-color` | — | Colorless output | `false` |

#### `show roles`

List the cloud roles the current user can assume (does not generate profiles).

```shell
# First time (instance required; client-id optional)
alibaba-cloud-idaas show roles --instance acme.aliyunidaas.com --client-id app_xxx

# Subsequent runs (inferred automatically from an existing profile)
alibaba-cloud-idaas show roles

# JSON output
alibaba-cloud-idaas show roles --json
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-i` | IDaaS instance domain (inferred from an existing profile when omitted) | — | First run |
| `--scope` | `-s` | Scope | `urn:cloud:idaas:pam\|.all offline_access` | No |
| `--client-id` | — | Broker client application ID | explicit > matching selected/unambiguous profile > `iap_cloud_idaas_cli` | No |
| `--config` | `-c` | Config file path | auto (see Configuration Files) | No |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
| `--json` | — | Machine-readable JSON output | `false` |

Read-only output fields (no token decryption, refresh or cloud call to populate them):

| Field | Type | Current meaning |
| --- | --- | --- |
| `config_state` | string | loaded after the selected configuration loads successfully |
| `cache_readability` | string | not_checked: cache readability has not been verified |
| `login_state` | string | not_checked: login validity has not been verified |
| `cloud_api_verification` | string | not_checked: no cloud identity call was performed |
| `client_context` | string | profile_login or not_applicable; not historical provenance |
| `client_id` / `client_id_source` / `issuer` | string | Effective selected-profile inputs without command-line overrides; never fabricate an unavailable value |

#### `show instance`

Display instance discovery information.

```shell
alibaba-cloud-idaas show instance --instance acme.aliyunidaas.com
```

| Flag | Alias | Description | Default | Required |
|------|------|------|--------|------|
| `--instance` | `-d` | IDaaS instance domain | — | ✅ |
| `--vpc` | — | Prefer the VPC endpoint | `false` | No |
| `--config` | `-c` | Broker config for client-id resolution; its absolute path is displayed | auto (see Configuration Files) | No |
| `--client-id` | — | Explicit override; otherwise use a matching selected/unambiguous cloud profile, then the built-in default | `iap_cloud_idaas_cli` | No |

#### `show cache` / `show token` / `show signer-key`

`show cache` lists metadata only; `show signer-key` delegates to the public-key implementation. `show token` reports an explicit unsupported/sensitive-operation error; the legacy `show-token` command remains available for deliberate credential display.

### `show-token`

Display current credentials in human-readable form (colored output).

```shell
alibaba-cloud-idaas show-token --profile aliyun-readonly
```

| Flag | Alias | Description | Default |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS profile | `IDAAS_PROFILE` or `current_profile` |
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |

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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) | No |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) | No |
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
| `--config` | `-c` | Config file path | auto (see Configuration Files) | No |
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
| auto (see Configuration Files) | Broker profile config (provider type + parameters) |
| `~/.aliyun/config.json` | aliyun-cli External profile (generated by `onboard`) |
| `~/.aws/config` | aws-cli credential_process profile (generated by `onboard`) |

Explicit `--config` takes precedence, followed by `ALIBABA_CLOUD_IDAAS_CONFIG_FILE`. Otherwise the legacy `~/.aliyun/alibaba-cloud-idaas.json` is used if it exists; if absent, `~/.cloud_idaas/idaas-cli.json` is used.

Cache directory selection is unchanged and depends on whether the legacy default config exists when the process starts. An explicit `--config` does not isolate the cache (encrypted oidc_token / cloud_token / token_response):

| Config file in use | Cache directory |
|------|------|
| Legacy `~/.aliyun/alibaba-cloud-idaas.json` exists | `~/.aliyun/alibaba-cloud-idaas/` |
| Legacy default does not exist | `~/.cloud_idaas/cloud-cli/` |

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

## Onboarding compatibility fixes (0.2.5-beta, pending release)

- `--client-id` is optional and defaults to `iap_cloud_idaas_cli`. Discovery does not supply it. Resolution is explicit arguments, then the matching selected/unambiguous profile, then the built-in default. Ambiguity is an error; clients are never borrowed from another instance. `login --instance` uses the same resolution.
- `onboard --aws-region cn-north-1` explicitly overrides the AWS region. New profiles use a partition-compatible default; compatible existing regions are retained. Conflicting existing regions fail with instructions to override explicitly. AWS profiles are named only; existing default and SSO sections are preserved.
- `configure switch --profile <name> [--config <path>]` changes only the Broker default. Cloud defaults are independent. Config version remains 1.
- `show instance` accepts `--config`/`-c` and `--client-id` and reports client resolution. `show status --json` retains old fields and adds context/default metadata and `login_state=not_checked`. It does not authenticate or validate login lifetime.
- Partial configuration failures now return nonzero, with per-target SUCCESS/FAILED output. Broker profiles may already be saved; retry only failed targets after resolving their errors, without automatically logging in again.
- Unknown targets, mixing none, and targets without matching roles are errors. Use `--target none` for Broker profiles only. The previously unimplemented mcp target no longer silently succeeds.
- Existing identity names are reused; other identities cannot be rebound through a name collision. Malformed/unreadable configs fail without overwriting. Atomic writes preserve symlink targets; cooperating writers use a lock and known external edits are rejected. Avoid concurrent non-cooperating editors; inspect abandoned locks only after the owning process has stopped.
- AWS external commands quote absolute paths. Some Alibaba credential consumers split on whitespace; whitespace paths are explicitly rejected instead of persisting an unusable command. Use installation/config paths without spaces for those consumers. AWS quoting support does not imply Alibaba quoting support.
- `show cache` lists metadata without decrypting; `show signer-key` queries the public key. `show token` refuses implicit credential output; the explicit legacy `show-token` remains available. Legacy top-level commands are retained.
- Cache derivation, seeds, device authorization and Token reuse are outside this change. No cache deletion or config-format migration is performed.

AWS configuration respects `AWS_CONFIG_FILE` and shared credentials respect `AWS_SHARED_CREDENTIALS_FILE`; a nonempty same-name credentials section is rejected to avoid provider precedence conflicts. TencentCloud static credential files have no reliable identity reference: existing files are preserved and onboarding fails with instructions to use a new prefix or explicitly back up and remove only the selected file. Prefixes accept letters, digits, dot, underscore and hyphen only; unsupported characters now fail before login. This is an intentional behavior change; the existing explicit credential export command is unchanged.

Atomic replacement writes mode 0600 and does not preserve custom ACLs/hard-link sharing. Shared configuration deployments must account for this. Locks serialize cooperating CLI writers and detect known intervening edits; they cannot guarantee coordination with every external editor. After a killed process, remove a stale lock only after verifying no writer remains. Strict consumers of status JSON must allow additional fields.

## Structured onboarding results and read-only diagnostics

`onboard --result-file /absolute/new-result.json` opts into a separate credential-free JSON result (schema_version=1). The file must be a new .json path, distinct from Broker/cloud configuration; existing reports are never overwritten. Terminal/device login prompts retain their existing behavior and are not mixed into this JSON. The normal exit code remains authoritative.

Fields: status (success/partial_success/failed), stage, config_file, broker_config_saved, profile_count and targets. Each target includes target, status, configuration_files and an optional error_code. For a failed target, configuration_files identifies intended destinations, not proof that every file was written or rolled back. Raw backend errors and credentials are not serialized. A result-file publication failure also returns nonzero, even if configurations were successfully written; inspect existing state before retrying. Atomic publication uses a same-directory temporary file and an exclusive hard link; a filesystem without hard-link support reports an output error without replacing an existing file.

`show instance` now prints the resolved absolute config path. `show status` reports config_state=loaded, cache_readability=not_checked, login_state=not_checked, cloud_api_verification=not_checked and client_context=profile_login when applicable. Client metadata is resolved by the same helper as login --profile, preserving its effective issuer/client-id/scope defaults. These values describe a selected profile login without command-line overrides; they do not reconstruct a previous invocation or claim a current cloud call has succeeded. Read-only status never decrypts credentials, refreshes tokens or opens a login page.

No automatic full-config backup/rollback or stale-lock deletion is added: multi-file/cloud configuration writes are not one transaction, and reverting a whole file can overwrite another process's legitimate changes. On failure, preserve the current file and inspect per-target results. Explicit backups, if needed, must remain local with restricted access; restore only after checking for subsequent changes.

Registration compatibility: when only the implicit current_profile belongs to a non-cloud workflow, an unambiguous cloud context may be inferred. Multiple roles sharing the same instance/client are not ambiguous; different instance/client contexts remain errors. An unusable explicit IDAAS_PROFILE, a missing default name, or a broken default cloud identity is never silently replaced. Supply --instance or correct the selected profile. Resolution does not change defaults, device authorization or credential caches.
