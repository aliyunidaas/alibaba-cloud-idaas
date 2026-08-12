# alibaba-cloud-idaas 命令参考

> 版本：v0.2.0-beta
> 更新：2026-08-11

## 命令总览

| 命令 | 用途 | 状态 |
|------|------|------|
| `onboard` | 发现实例 + 登录 + 列角色 + 生成 CLI 工具配置 | ✅ |
| `login` | 设备码登录 IDaaS 并缓存 access token | ✅ |
| `fetch-token` | 取凭证输出 JSON（credential_process 契约） | ✅ |
| `serve` | 启动本地 HTTP 凭证服务（供 SDK credentials_uri） | ✅ |
| `show` | 查询子命令族（profiles/roles/cache/token/status/instance/signer-key） | ✅ |
| `show-token` | 人类可读展示当前凭证 | ✅ |
| `show-profiles` | 列出已配置的 profile | ✅ |
| `show-cache` | 展示缓存条目 | ✅ |
| `clean-cache` | 清除全部缓存 | ✅ |
| `logout` | 清除缓存 token（保留 profile 配置） | ✅ |
| `status` | 展示当前 profile / 登录态 / serve daemon 状态 | ✅ |
| `execute` | 注入环境变量并执行命令 | ✅ |
| `show-signer-public-key` | 展示签名器公钥 | ✅ |
| `qr` | 生成二维码 | ✅ |
| `validate-jwt` | 验证 JWT（仅 RS256） | ✅ |
| `openclaw-secret` | 获取 OpenClaw 密钥 | ✅ |
| `agent` | Agent 子命令族（见下） | ✅ |

---

## 顶层命令

### `onboard`

零配置接入：发现实例 → 登录（自动触发 login）→ 列可用角色 → 为 CLI 工具生成配置。

```shell
# 首次（需指定实例 + client-id）
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx --target aliyun-cli,aws-cli

# 非首次（已有 profile，--instance / --client-id 自动从已有 profile 推断）
alibaba-cloud-idaas onboard
alibaba-cloud-idaas onboard --target aws-cli

# 不写 CLI 配置（只生成 broker profile）
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx --target none
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-i` | IDaaS 实例域名（未传时从已有 profile 推断） | — | 首次必填 |
| `--target` | — | 目标 CLI 工具（逗号分隔：`aliyun-cli`/`aws-cli`/`tencentcloud-cli`/`mcp`/`none`） | 全部适用 | 否 |
| `--prefix` | — | 生成 profile 名前缀 | `aliyun` | 否 |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` | 否 |
| `--vpc` | — | 优先 VPC 端点 | `false` | 否 |
| `--client-id` | — | broker 客户端应用 ID（透传给 login） | 显式 > 已有 profile > discovery `cli_client_id` > `iap_developer` | 首次必填 |
| `--force-new` | `-N` | 强制设备码登录（透传给 login） | `false` | 否 |

### `login`

设备码登录 IDaaS，获取 access token 并缓存。不写 profile、不写 CLI 配置。

```shell
# 首次登录（需指定实例 + client-id）
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx

# 刷新登录（从 profile 自动读 issuer + scope + client-id，不需再传 --instance / --client-id）
alibaba-cloud-idaas login --profile aliyun-readonly

# refresh token 也过期（需重新提供实例 + client-id）
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx --force-new

# 自定义 scope
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx --scope "urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential urn:cloud:idaas:pam|credential:obtain"
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-i` | 实例域名（首次登录模式，未传时从已有 profile 推断） | — | 与 `--profile` 二选一 |
| `--profile` | `-p` | 已有 profile 名（刷新模式，从 profile 自动读 issuer+scope+client-id） | — | 与 `--instance` 二选一 |
| `--scope` | `-s` | 空格分隔的 `audience\|scope` 组合串 | `urn:cloud:idaas:pam\|.all` | 否 |
| `--client-id` | — | broker 客户端应用 ID（`--profile` 模式从 profile 自动读） | `--instance` 模式需显式提供 | `--instance` 模式必填 |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` | 否 |
| `--force-new` | `-N` | 忽略缓存强制重新登录 | `false` | 否 |

### `fetch-token`

取凭证并输出 JSON（aliyun-cli External / aws-cli credential_process 契约）。

```shell
alibaba-cloud-idaas fetch-token --profile aliyun-readonly
alibaba-cloud-idaas fetch-token --profile aws-role --format raw
alibaba-cloud-idaas fetch-token --profile oidc1 --oidc-field access_token
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS Profile | `IDAAS_PROFILE` 或 `current_profile` |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--format` | `-f` | STS 输出格式：`aliyuncli`/`ossutilv2`/`raw` | `aliyuncli` |
| `--oidc-field` | — | OIDC token 字段：`id_token`/`access_token` | — |
| `--oidc-format` | — | OIDC 格式：`type1`/`type2` | `type1` |
| `--output` | `-o` | 输出到文件 | stdout |
| `--force-new` | `-N` | 强制刷新，忽略所有缓存 | `false` |
| `--force-new-cloud-token` | — | 强制刷新云凭证（低层缓存） | `false` |

### `serve`

启动本地 HTTP 凭证服务，供阿里云 SDK `credentials_uri` 使用。

```shell
alibaba-cloud-idaas serve --ssrf-token my-secret-token
alibaba-cloud-idaas serve --port 8080 --ssrf-token my-secret-token
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--port` | `-p` | 监听端口 | `1127` |
| `--ssrf-token` | — | SSRF token（header `X-Aliyun-Parameters-Secrets-Token` 或 query `__ssrf_token`） | — |
| `--unsafe-listen-host` | — | 监听地址（默认 `127.0.0.1`，可设 `0.0.0.0`） | `127.0.0.1` |
| `--unsafe-disable-ssrf` | — | 禁用 SSRF 校验 | `false` |

**端点**：
- `GET /cloud_token?profile=X&__ssrf_token=T` → STS JSON（`credentials_uri` 契约）
- `GET /version` → 版本信息

### `show`

查询子命令族，不修改配置。

#### `show profiles`

列出已配置的 profile。

```shell
alibaba-cloud-idaas show profiles
alibaba-cloud-idaas show profiles -f aliyun
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile-filter` | `-f` | profile 名过滤 | — |
| `--no-color` | — | 无色彩输出 | `false` |

#### `show roles`

列出当前用户可 Assume 的云角色（不生成 profile）。

```shell
# 首次（需指定实例 + client-id）
alibaba-cloud-idaas show roles --instance acme.aliyunidaas.com --client-id app_xxx

# 非首次（已有 profile，自动推断）
alibaba-cloud-idaas show roles

# JSON 输出
alibaba-cloud-idaas show roles --json
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-i` | IDaaS 实例域名（未传时从已有 profile 推断） | — | 首次必填 |
| `--scope` | `-s` | scope | `urn:cloud:idaas:pam\|.all` | 否 |
| `--client-id` | — | broker 客户端应用 ID（未传时从已有 profile 推断） | — | 首次必填 |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` | 否 |
| `--vpc` | — | 优先 VPC 端点 | `false` | 否 |
| `--json` | — | 机器可读 JSON 输出 | `false` | 否 |

#### `show status`

展示当前 profile / provider / instance / serve daemon 状态。

```shell
alibaba-cloud-idaas show status
alibaba-cloud-idaas show status --profile aliyun-readonly
alibaba-cloud-idaas show status --json
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | 指定 profile | `IDAAS_PROFILE` 或 `current_profile` |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--json` | — | 机器可读 JSON 输出 | `false` |

#### `show instance`

展示实例发现信息。

```shell
alibaba-cloud-idaas show instance --instance acme.aliyunidaas.com
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-d` | IDaaS 实例域名 | — | ✅ |
| `--vpc` | — | 优先 VPC 端点 | `false` | 否 |

#### `show cache` / `show token` / `show signer-key`

占位子命令（当前调用旧的顶层命令 `show-cache` / `show-token` / `show-signer-public-key`）。

### `show-token`

人类可读展示当前凭证（彩色输出）。

```shell
alibaba-cloud-idaas show-token --profile aliyun-readonly
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS Profile | `IDAAS_PROFILE` 或 `current_profile` |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--oidc-field` | — | OIDC token 字段 | — |
| `--no-color` | — | 无色彩输出 | `false` |
| `--force-new` | `-N` | 强制刷新 | `false` |
| `--force-new-cloud-token` | — | 强制刷新云凭证 | `false` |

### `show-profiles`

列出已配置的 profile。

```shell
alibaba-cloud-idaas show-profiles
alibaba-cloud-idaas show-profiles --profile-filter aliyun
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile-filter` | `-p` | profile 名过滤 | — |
| `--no-color` | — | 无色彩输出 | `false` |

### `show-cache`

展示缓存条目。

```shell
alibaba-cloud-idaas show-cache
alibaba-cloud-idaas show-cache --category oidc_token --name app_xxx
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--category` | `-c` | 缓存类别（`oidc_token`/`cloud_token`/`token_response`） | 全部 |
| `--name` | `-n` | 缓存名过滤 | — |

### `clean-cache`

清除全部缓存。

```shell
alibaba-cloud-idaas clean-cache
```

无参数。

### `logout`

清除缓存 token，保留 profile 配置。

```shell
alibaba-cloud-idaas logout                          # 清除所有缓存
alibaba-cloud-idaas logout --profile aliyun-readonly # 清除指定 profile
alibaba-cloud-idaas logout --profile aliyun-readonly --dry-run  # 预览
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | 要清除的 profile（不传则清全部） | — |
| `--dry-run` | — | 只展示不实际删除 | `false` |

### `status`

展示当前 profile、登录态、serve daemon 状态。

```shell
alibaba-cloud-idaas status
alibaba-cloud-idaas status --profile aliyun-readonly
alibaba-cloud-idaas status --json
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | 指定 profile | `IDAAS_PROFILE` 或 `current_profile` |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--json` | — | 机器可读 JSON 输出 | `false` |

### `execute`

注入环境变量并执行命令。

```shell
alibaba-cloud-idaas execute --profile aliyun-readonly --env-region cn-hangzhou aliyun oss ls
alibaba-cloud-idaas execute --profile aliyun-readonly bash
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS Profile | `IDAAS_PROFILE` 或 `current_profile` |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--env-region` | `-R` | 设置环境变量 region | — |
| `--force-new` | `-N` | 强制刷新 | `false` |
| `--force-new-cloud-token` | — | 强制刷新云凭证 | `false` |
| `--show-token` | — | 执行前展示凭证 | `false` |

### `show-signer-public-key`

展示签名器公钥。

```shell
alibaba-cloud-idaas show-signer-public-key --profile aliyun3
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS Profile | — |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |

### `qr`

生成二维码。

```shell
alibaba-cloud-idaas qr --content "https://example.com"
alibaba-cloud-idaas qr --content "https://example.com" --small
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--content` | 二维码内容 | — |
| `--small` | 小尺寸二维码 | `false` |

### `validate-jwt`

验证 JWT（仅 RS256）。

```shell
alibaba-cloud-idaas validate-jwt --token "eyJhbGciOi..."
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--token` | `-t` | JWT token | — | ✅ |

### `openclaw-secret`

获取 OpenClaw 密钥。

```shell
alibaba-cloud-idaas openclaw-secret --profile agent1
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS Profile | — |
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--force-new` | `-N` | 强制刷新 | `false` |

---

## agent 子命令

### `agent access-token`

获取 agent access token。

```shell
alibaba-cloud-idaas agent access-token --profile agent1
alibaba-cloud-idaas agent access-token --profile agent1 --scope "urn:cloud:idaas:pam|.all"
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile` | `-p` | IDaaS Profile | — |
| `--scope` | `-s` | scope，格式 `audience\|scope-value` | 从 config 读取 |
| `--force-new` | `-N` | 强制刷新 | `false` |

### `agent get-secret`

获取密钥。

```shell
alibaba-cloud-idaas agent get-secret --profile agent1 --name default_model
alibaba-cloud-idaas agent get-secret --profile agent1 --name default_model --json-query .default_model.value.apiKeyContent.apiKey
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` |
| `--profile` | `-p` | IDaaS Profile | — |
| `--scope` | `-s` | scope | `urn:cloud:idaas:pam\|credential:obtain` |
| `--json-query` | `-q` | JSON 查询表达式 | — |
| `--name` | `-n` | 密钥名（可多次指定） | — |
| `--raw` | — | 输出原始响应 | `false` |
| `--string-raw` | — | 输出原始 JSON 字符串 | `false` |
| `--force-new` | `-N` | 强制刷新 | `false` |

### `agent put-secret`

存储密钥。

```shell
alibaba-cloud-idaas agent put-secret --profile agent1 --name my-key --value "sk-xxx"
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` | 否 |
| `--profile` | `-p` | IDaaS Profile | — | 否 |
| `--scope` | `-s` | scope | `urn:cloud:idaas:pam\|credential:manage` | 否 |
| `--name` | `-n` | 密钥名 | — | ✅ |
| `--display-name` | — | 显示名 | 同 `--name` | 否 |
| `--value` | — | 密钥值 | — | ✅ |

### `agent decrypt-secret`

解密密钥。

```shell
alibaba-cloud-idaas agent decrypt-secret --profile agent1 --name default_model --ciphertext "encrypted..."
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` | 否 |
| `--profile` | `-p` | IDaaS Profile | — | 否 |
| `--scope` | `-s` | scope | `urn:cloud:idaas:pam\|credential:decrypt` | 否 |
| `--name` | `-n` | 凭据标识 | — | ✅ |
| `--ciphertext` | — | 密文 | — | ✅ |

### `agent token-exchange`

Token Exchange (RFC 8693)。

```shell
alibaba-cloud-idaas agent token-exchange --profile agent1 --subject-token "eyJ..." 
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--config` | `-c` | 配置文件路径 | `~/.aliyun/alibaba-cloud-idaas.json` | 否 |
| `--profile` | `-p` | IDaaS Profile | — | 否 |
| `--scope` | `-s` | scope | — | 否 |
| `--subject-token-type` | `-T` | subject token 类型 | `urn:ietf:params:oauth:token-type:access_token` | 否 |
| `--subject-token` | `-S` | subject token | — | ✅ |

---

## 环境变量

| 变量 | 说明 |
|------|------|
| `IDAAS_PROFILE` | 默认 profile（优先级：`--profile` > `IDAAS_PROFILE` > `current_profile`） |
| `ALIBABA_CLOUD_IDAAS_USER_AGENT` | OIDC 请求 User-Agent |
| `ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG` | 日志输出敏感数据 |
| `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT` | 日志复制到 stderr |
| `ALIBABA_CLOUD_IDAAS_PKSC11_PIN` | PKCS#11 PIN |
| `ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN` | YubiKey PIV PIN |

---

## 配置文件

| 文件 | 作用 |
|------|------|
| `~/.aliyun/alibaba-cloud-idaas.json` | broker profile 配置（provider 类型 + 参数） |
| `~/.aliyun/config.json` | aliyun-cli External profile（由 `onboard` 生成） |
| `~/.aws/config` | aws-cli credential_process profile（由 `onboard` 生成） |
| `~/.aliyun/alibaba-cloud-idaas/` | 加密缓存目录（oidc_token / cloud_token / token_response） |

---

## 编译

```shell
go build -o alibaba-cloud-idaas .

# 禁用硬件签名器
go build -tags disable_pkcs11,disable_yubikey_piv

# 交叉编译
GOOS=darwin  GOARCH=arm64 go build -o alibaba-cloud-idaas .
GOOS=linux   GOARCH=amd64 go build -o alibaba-cloud-idaas .
GOOS=windows GOARCH=amd64 go build -o alibaba-cloud-idaas.exe .

# 安装到 PATH
go install
```
