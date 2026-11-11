# alibaba-cloud-idaas 命令参考

[English](command-reference.md) | **中文**

> 版本：v0.2.5-beta（开发中，待发布；以实际安装版本的 --help 为准）。

## 命令总览

| 命令 | 用途 | 状态 |
|------|------|------|
| `onboard` | 发现实例 + 登录 + 列角色 + 生成 CLI 工具配置 | ✅ |
| `configure switch` | 仅显式切换 Broker 默认 profile | ✅ |
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
# 首次（实例必需，client-id 可选）
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
| `--instance` | `-i` | IDaaS 实例域名（未传时从已有 profile 推断） | — | 无法推断时必填 |
| `--target` | — | 目标 CLI 工具（逗号分隔：`aliyun-cli`/`aws-cli`/`tencentcloud-cli`/`none`） | 全部适用 | 否 |
| `--aws-region` | — | 显式覆盖 AWS 地域，须匹配角色 ARN 分区；不传时保留已有合法地域 | 按分区选择 | 否 |
| `--result-file` | — | 将无凭证 JSON 结果写入全新的 .json 文件，不与登录提示混写、不覆盖旧文件 | 不生成 | 否 |
| `--prefix` | — | 生成 profile 名前缀 | 按厂商自动选择（`aliyun`/`aws`/`tencentcloud`） | 否 |
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） | 否 |
| `--vpc` | — | 优先 VPC 端点 | `false` | 否 |
| `--client-id` | — | broker 客户端应用 ID（透传给 login） | 显式 > 已有 profile > `iap_cloud_idaas_cli` | 否 |
| `--force-new` | `-N` | 强制设备码登录（透传给 login） | `false` | 否 |

### `configure switch`

显式切换 Broker 的 current_profile，保留云 CLI 默认及其他配置。

```shell
alibaba-cloud-idaas configure switch --profile <name> --config <absolute-path>
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
| --- | --- | --- | --- | --- |
| `--profile` | `-p` | 已存在的 Broker profile | — | 是 |
| `--config` | `-c` | Broker 配置文件路径 | 自动选择（见“配置文件”） | 否 |

### `login`

设备码登录 IDaaS，获取 access token 并缓存。不写 profile、不写 CLI 配置。

```shell
# 首次登录（实例必需，client-id 可选）
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx

# 刷新登录（从 profile 自动读 issuer + scope + client-id，不需再传 --instance / --client-id）
alibaba-cloud-idaas login --profile aliyun-readonly

# 确认需要重新授权时，沿用已有 profile 和本轮配置路径
alibaba-cloud-idaas login --profile aliyun-readonly --config <absolute-path> --force-new

# 自定义 scope
alibaba-cloud-idaas login --instance acme.aliyunidaas.com --client-id app_xxx --scope "urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential urn:cloud:idaas:pam|credential:obtain"
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-i` | 实例域名（首次登录模式，与 --profile 互斥） | — | 与 `--profile` 二选一 |
| `--profile` | `-p` | 已有 profile 名（刷新模式，从 profile 自动读 issuer+scope+client-id） | — | 与 `--instance` 二选一 |
| `--scope` | `-s` | 空格分隔的 `audience\|scope` 组合串 | `urn:cloud:idaas:pam\|.all offline_access` | 否 |
| `--client-id` | — | broker 客户端应用 ID（`--profile` 模式从 profile 自动读） | 显式 > 所选/同实例 profile > `iap_cloud_idaas_cli` | 否 |
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） | 否 |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
| `--profile-filter` | `-f` | profile 名过滤 | — |
| `--no-color` | — | 无色彩输出 | `false` |

#### `show roles`

列出当前用户可 Assume 的云角色（不生成 profile）。

```shell
# 首次（实例必需，client-id 可选）
alibaba-cloud-idaas show roles --instance acme.aliyunidaas.com --client-id app_xxx

# 非首次（已有 profile，自动推断）
alibaba-cloud-idaas show roles

# JSON 输出
alibaba-cloud-idaas show roles --json
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-i` | IDaaS 实例域名（未传时从已有 profile 推断） | — | 否 |
| `--scope` | `-s` | scope | `urn:cloud:idaas:pam\|.all offline_access` | 否 |
| `--client-id` | — | broker 客户端应用 ID | 显式 > 已有 profile > `iap_cloud_idaas_cli` | 否 |
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） | 否 |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
| `--json` | — | 机器可读 JSON 输出 | `false` |

只读输出字段（不会为填状态而解密、刷新或调用云 API）：

| 字段 | 类型 | 当前含义 |
| --- | --- | --- |
| `config_state` | string | 已选配置成功加载时为 loaded |
| `cache_readability` | string | not_checked：未核验缓存可读性 |
| `login_state` | string | not_checked：未核验登录有效性 |
| `cloud_api_verification` | string | not_checked：未执行云身份验证 |
| `client_context` | string | profile_login 或 not_applicable，不反推历史调用 |
| `client_id` / `client_id_source` / `issuer` | string | 所选 profile 在无命令行覆盖时的有效输入；无法解析时不编造 |

#### `show instance`

展示实例发现信息。

```shell
alibaba-cloud-idaas show instance --instance acme.aliyunidaas.com
```

| 参数 | 别名 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| `--instance` | `-d` | IDaaS 实例域名 | — | ✅ |
| `--vpc` | — | 优先 VPC 端点 | `false` | 否 |
| `--config` | `-c` | 用于 client-id 推断的 Broker 配置；输出其绝对路径 | 自动选择（见“配置文件”） | 否 |
| `--client-id` | — | 显式覆盖；否则使用同实例已选/无歧义 cloud profile，最后内置默认 | `iap_cloud_idaas_cli` | 否 |

#### `show cache` / `show token` / `show signer-key`

`show cache` 仅展示缓存文件元数据，不解密凭证；`show token` 明确返回不支持直接展示凭证的错误；`show signer-key` 展示签名公钥。旧的顶层命令保留兼容，其中 `show-token` 会输出敏感信息，仅在明确需要时使用。

### `show-token`

人类可读展示当前凭证（彩色输出）。

```shell
alibaba-cloud-idaas show-token --profile aliyun-readonly
```

| 参数 | 别名 | 说明 | 默认值 |
|------|------|------|--------|
| `--profile` | `-p` | IDaaS Profile | `IDAAS_PROFILE` 或 `current_profile` |
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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

> 注意：此处 `-c` 是 `--category` 的缩写，**不是**其他命令里的 `--config`。`--name` 需配合 `--category` 使用才生效。

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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |

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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） | 否 |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） | 否 |
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
| `--config` | `-c` | 配置文件路径 | 自动选择（见“配置文件”） | 否 |
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
| 自动选择（见“配置文件”） | broker profile 配置（provider 类型 + 参数） |
| `~/.aliyun/config.json` | aliyun-cli External profile（由 `onboard` 生成） |
| `~/.aws/config` | aws-cli credential_process profile（由 `onboard` 生成） |

配置文件选择顺序：显式 `--config`、环境变量 `ALIBABA_CLOUD_IDAAS_CONFIG_FILE`；均未指定时，优先使用已存在的旧路径 `~/.aliyun/alibaba-cloud-idaas.json`，否则使用 `~/.cloud_idaas/idaas-cli.json`。

缓存目录选择保持不变，取决于进程启动时旧默认配置是否存在。显式 `--config` 不会隔离缓存（加密存放 oidc_token / cloud_token / token_response）：

| 使用的配置文件 | 缓存目录 |
|------|------|
| 旧默认文件 `~/.aliyun/alibaba-cloud-idaas.json` 存在 | `~/.aliyun/alibaba-cloud-idaas/` |
| 旧默认文件不存在 | `~/.cloud_idaas/cloud-cli/` |

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

## 接入兼容性修复（0.2.5-beta，待发布）

- client-id 可选，默认 iap_cloud_idaas_cli。实例发现不提供 client-id；显式参数优先于同实例的已选/无歧义 profile，最后使用内置默认。多实例歧义明确报错，不随机选取。login --instance 也采用相同规则。
- `onboard --aws-region cn-north-1` 可明确覆盖 AWS 地域。新 profile 按 ARN partition 选择兼容默认，保留已有合法地域；已有不兼容地域报错，需用户明确覆盖。AWS 仅写具名 profile，不再自动建立 default；已有 default/SSO 保留。
- `configure switch --profile <name> [--config <path>]` 仅切换 Broker 默认身份，不同步修改云 CLI；保留配置 version=1。云工具默认必须分别明确选择。
- `show instance` 新增 --config/-c、--client-id，展示本次解析的 client-id 和来源。`show status --json` 保留原字段，增加配置路径、客户端来源、各侧默认及 login_state=not_checked；此只读命令不验证登录有效性。
- 部分写入失败返回非零退出码，输出各 target 的 SUCCESS/FAILED；Broker 配置可能已保存，修复具体原因后仅重试失败 target，不自动重新登录。
- 参数校验拒绝未知 target、none 混用或无对应角色；仅生成配置用 --target none。mcp 原先未实现，不再伪装成功。
- 同身份复用已有名称，跨身份同名生成不冲突名称，已有名称不被改绑。读取/格式错误停止写入；原子替换保留现有符号链接的目标，合作写入使用锁，发现外部编辑时拒绝旧快照。不要并发使用其他不支持该锁的编辑器；锁文件异常残留须核实进程已结束再处理。
- AWS credential_process 正确引用绝对路径。部分阿里云消费端只按空白拆分参数；对含空格路径返回明确错误，不写入无法运行的命令。请选无空格的配置/安装路径，不能把支持 AWS 引号当作阿里云也支持。
- `show cache` 仅列类别/文件元数据，不隐式解密；`show signer-key` 接通公钥查询；`show token` 不隐式输出凭证，需有意查看时使用旧 show-token。旧顶层命令保留。
- 缓存派生、seed、设备授权及 Token 复用不在本轮修复范围内。升级不清缓存、不更换配置格式；回退仍须检查实际 Token 生命周期。

AWS 配置遵循 `AWS_CONFIG_FILE`，共享凭证遵循 `AWS_SHARED_CREDENTIALS_FILE`；共享凭证中存在同名非空段时拒绝写入，避免 provider 优先级遮蔽。腾讯云静态凭证文件没有可靠身份引用，已有文件一律保留并明确失败；可改用新前缀，或由用户明确备份并移除选定文件后重试。prefix 仅接受字母、数字、点、下划线和连字符，其他字符在登录前报错。以上属于有意的行为变化；原有显式凭证导出命令未改动。

原子替换后的文件权限为 0600，不承诺保留自定义 ACL 或硬链接共享。共享配置部署需提前评估此变化。锁仅协调遵守该锁的 CLI 并检测已发生的外部修改，无法保证任意外部编辑器间的事务一致性；进程被强制终止后，需确认无写入进程再处理遗留锁。严格校验 status JSON 的脚本需允许新增字段。

## 结构化接入结果与只读诊断

`onboard --result-file /绝对路径/全新结果.json` 可选输出独立、无凭证的 JSON（schema_version=1）。要求全新的 .json 路径，不能是 Broker/云 CLI 配置，不覆盖旧结果。原终端及设备登录提示行为保留，不混入该 JSON；进程退出码仍须检查。

字段包括 status（success/partial_success/failed）、stage、config_file、broker_config_saved、profile_count、targets；每个 target 含 target、status、configuration_files 和可选 error_code。失败 target 的 configuration_files 是预期目标路径，不证明所有文件已写入或回滚。不序列化原始后端错误或凭证。结果文件发布失败也返回非零，即使配置已经写入；重试前核对现有状态。发布采用同目录临时文件与排他硬链接；文件系统不支持硬链接时明确返回输出错误，不替换已有文件。

`show instance` 展示解析后的绝对配置路径。`show status` 分别输出 config_state=loaded、cache_readability=not_checked、login_state=not_checked、cloud_api_verification=not_checked；适用时 client_context=profile_login。客户端元信息与 login --profile 共用解析 helper，保持最终 issuer/client-id/scope 默认语义；描述的是所选 profile 在无命令行覆盖时的登录输入，不能反推历史调用，更不表示云调用已成功。只读 status 不解密凭证、不刷新 Token、不弹登录页面。

不新增自动整份备份/回滚或遗留锁自动删除：多文件及多云配置不是单一事务，恢复整份旧文件可能覆盖其他进程的合法改动。失败时保护当前文件并依据逐 target 结果处理；确需显式备份，应本机受控保存，恢复前核对后续改动。

实例推断的兼容性边界：隐式 current_profile 属于非 cloud 工作流时，可从无歧义的 cloud 候选推断；多个角色若属于相同实例/客户端上下文不算歧义。多个不同实例/客户端仍明确报错。显式 IDAAS_PROFILE 无法使用、默认名称不存在或默认 cloud 身份损坏时不自动换用其他身份，须显式 --instance 或更正所选 profile。该逻辑不修改配置默认、设备授权或缓存。
