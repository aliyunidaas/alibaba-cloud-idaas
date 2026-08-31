# alibaba-cloud-idaas

[English](README.md) | **中文**

> [!IMPORTANT]
> 当前为预览版本。

使用 [阿里云 IDaaS](https://www.aliyun.com/product/idaas) 连接阿里云及其他云厂商的命令行工具。

## 编译

编译 `alibaba-cloud-idaas` 直接执行 `go build`：
```shell
go build
```

指定输出的二进制文件名：
```shell
go build -o alibaba-cloud-idaas .
```

PKCS#11 与 YubiKey 特性可通过构建标签关闭：
```shell
go build -tags disable_pkcs11,disable_yubikey_piv
```

安装到 `$GOPATH/bin`（这样 `alibaba-cloud-idaas` 就在 `PATH` 中）：
```shell
go install
```

交叉编译到其他平台：
```shell
GOOS=darwin  GOARCH=arm64 go build -o alibaba-cloud-idaas .
GOOS=linux   GOARCH=amd64 go build -o alibaba-cloud-idaas .
GOOS=windows GOARCH=amd64 go build -o alibaba-cloud-idaas.exe .
```

## 外部签名器

支持的外部签名器：

- YubiKey PIV 签名器 —— Linux 上需要 `pcsc-lite`
- PKCS#11 签名器
- 自定义外部签名器

## 环境变量

| 环境变量                                       | 说明                        |
|----------------------------------------------|----------------------------|
| ALIBABA_CLOUD_IDAAS_USER_AGENT               | 发送 OIDC HTTP 请求时的 User-Agent |
| ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG             | 将敏感（安全）数据写入日志文件        |
| ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT     | 将日志复制到控制台（stderr）        |
| ALIBABA_CLOUD_IDAAS_PKSC11_PIN               | PKCS#11 PIN                |
| ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN              | YubiKey PIN                |


## Profile 配置

## 配置文件位置

`~/.aliyun/alibaba-cloud-idaas.json` 存在时优先使用，
否则使用 `~/.cloud_idaas/idaas-cli.json`
> `~` 表示 `$HOME`

### 🆕 通过设备码流程实现 AKless

使用 IDaaS 新的 AKless 能力获取 STS Token。

```json
{
  "version": "1",
  "profile": {
    "aliyun-akless1": {
      "cloud_account_token": {
        "cloud_account_region": "cn-hangzhou",
        "cloud_account_instance_id": "idaas_wrwsx*********************",
        "cloud_account_role_external_id": "acs:ram::1391************:role/hatter-test-akless-role",
        "access_token_provider":{
          "device_code": {
            "issuer": "https://ziw*****.aliyunidaas.com/api/v2/iauths_system/oauth2",
            "client_id": "app_m7jks3********************",
            "scope": "urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential offline_access openid",
            "auto_open_url": true,
            "show_qr_code": true,
            "small_qr_code": true
          }
        }
      }
    }
  }
}
```

### 设备码流程

遵循规范 RFC 8628: OAuth 2.0 Device Authorization Grant。
> 公有客户端（public client）不需要 `client_secret`
```json
{
  "version": "1",
  "profile": {
    "aliyun1": {
      "alibaba_cloud_sts": {
        "sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
        "oidc_provider_arn": "acs:ram::1391************:oidc-provider/hatter-sts-test",
        "role_arn": "acs:ram::1391************:role/hatter-sts-role",
        "oidc_token_provider": {
          "device_code": {
            "issuer": "https://eiam-api-cn-hangzhou.aliyuncs.com/v2/idaas_wrwsx*********************/app_m7jks3********************/oidc",
            "client_id": "app_m7jks3********************",
            "auto_open_url": true,
            "show_qr_code": true,
            "small_qr_code": true
          }
        }
      }
    }
  }
}
```

### ClientID/ClientSecret

```json
{
  "version": "1",
  "profile": {
    "aliyun2": {
      "alibaba_cloud_sts": {
        "sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
        "oidc_provider_arn": "acs:ram::1391************:oidc-provider/hatter-m2m",
        "role_arn": "acs:ram::1391************:role/hatter-sts-role",
        "oidc_token_provider": {
          "client_credentials": {
            "token_endpoint": "https://ziwd****.aliyunidaas.com/api/v2/iauths_system/oauth2/token",
            "client_id": "app_m7iug*********************",
            "client_secret": "CSFG*****************************************e",
            "scope": "https://test.example.com|.all"
          }
        }
      }
    }
  }
}
```

### 使用 YubiKey 进行公钥签名
> 未配置时从环境变量 `ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN` 读取
```json
{
  "version": "1",
  "profile": {
    "aliyun3": {
      "alibaba_cloud_sts": {
        "sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
        "oidc_provider_arn": "acs:ram::1391************:oidc-provider/hatter-m2m",
        "role_arn": "acs:ram::1391************:role/hatter-sts-role",
        "oidc_token_provider": {
          "client_credentials": {
            "token_endpoint": "https://ziwd****.aliyunidaas.com/api/v2/iauths_system/oauth2/token",
            "client_id": "app_m7iug*********************",
            "scope": "https://test.example.com|.all",
            "client_assertion_signer": {
              "key_id": "key1",
              "algorithm": "RS256",
              "yubikey_piv": {
                "slot": "R3",
                "pin": "******",
                "pin_policy": "once"
              }
            }
          }
        }
      }
    }
  }
}
```

### 使用 PKCS#11 进行公钥签名
> 未配置时从环境变量 `ALIBABA_CLOUD_IDAAS_PKSC11_PIN` 读取 PIN
```json
{
  "version": "1",
  "profile": {
    "aliyun4": {
      "alibaba_cloud_sts": {
        "sts_endpoint": "sts.cn-hangzhou.aliyuncs.com",
        "oidc_provider_arn": "acs:ram::1391************:oidc-provider/hatter-m2m",
        "role_arn": "acs:ram::1391************:role/hatter-sts-role",
        "oidc_token_provider": {
          "client_credentials": {
            "token_endpoint": "https://ziwd****.aliyunidaas.com/api/v2/iauths_system/oauth2/token",
            "client_id": "app_m7iug*********************",
            "scope": "https://test.example.com|.all",
            "client_assertion_signer": {
              "key_id": "key1",
              "algorithm": "RS256",
              "pkcs11": {
                "library_path": "/usr/local/lib/libykcs11.dylib",
                "token_label": "YubiKey PIV #16138686",
                "key_label": "Private key for Retired Key 3",
                "pin": "******"
              }
            }
          }
        }
      }
    }
  }
}
```

### 获取 AWS STS Token

```json
{
  "version": "1",
  "profile": {
    "aws1": {
      "aws_sts": {
        "region": "us-east-2",
        "role_arn": "arn:aws:iam::5418********:role/hatter-role-test",
        "oidc_token_provider": {
          "device_code": {
            "issuer": "https://eiam-api-cn-hangzhou.aliyuncs.com/v2/idaas_wrwsx*********************/app_m7jks3********************/oidc",
            "client_id": "app_m7jks3********************",
            "auto_open_url": true,
            "show_qr_code": true,
            "small_qr_code": true
          }
        }
      }
    }
  }
}
```

### 获取 OIDC Token

```json
{
  "version": "1",
  "profile": {
    "oidc1": {
      "oidc_token": {
        "device_code": {
          "issuer": "https://eiam-api-cn-hangzhou.aliyuncs.com/v2/idaas_wrwsx*********************/app_m7jks3********************/oidc",
          "client_id": "app_m7jks3********************",
          "auto_open_url": true,
          "show_qr_code": true,
          "small_qr_code": true
        }
      }
    }
  }
}
```
### 获取静态凭据

通过 Developer API 获取由 IDaaS 托管的静态凭据（例如 API Key）：

```json
{
  "version": "1",
  "profile": {
    "my-api-key": {
      "credential": {
        "instance_id": "idaas_wrwsx*********************",
        "developer_api_endpoint": "https://eiam-developerapi.cn-hangzhou.aliyuncs.com",
        "credential_identifier": "default_model",
        "access_token_provider": {
          "device_code": {
            "issuer": "https://<instance>/api/v2/<auth-server-id>/oauth2",
            "client_id": "iap_cloud_idaas_cli"
          }
        }
      }
    }
  }
}
```

```shell
alibaba-cloud-idaas fetch-token --profile my-api-key
```

> 注意：对于 `device_code`，token 类型现在默认为 **access_token**（实例授权服务器时代）。
> 无密钥 STS provider（`alibaba_cloud_sts` / `aws_sts`）会按 `AssumeRoleWithOIDC` 的要求自动使用 `id_token`。


## 运行命令

查看帮助信息 `alibaba-cloud-idaas --help`。

完整的参数说明见 [命令参考](docs/command-reference-zh_CN.md)。

子命令：
- `onboard`             - 零配置接入：发现实例、设备码登录、列出可 Assume 的云角色、生成 profile
- `login`               - 设备码登录 IDaaS 实例并缓存 access token
- `fetch-token`         - 获取 STS token，以 JSON 格式输出到 `stdout`
- `show-token`          - 展示 STS token（人类可读）
- `show-profiles`       - 展示 `~/.aliyun/alibaba-cloud-idaas.json` 或 `~/.cloud_idaas/idaas-cli.json` 中的 profile
- `show`               - 查询子命令族（profiles / roles / cache / token / status / instance / signer-key）
- `status`              - 展示当前 profile / 登录态 / serve daemon 状态
- `serve`               - 启动本地 HTTP 凭证服务（供 SDK `credentials_uri` 使用）
- `execute`             - 将 STS token 注入环境变量并执行命令
- `logout`              - 清除缓存的 token（保留 profile 配置）
- `clean-cache`         - 清除本地缓存，目录 `~/.aliyun/alibaba-cloud-idaas/` 或 `~/.cloud_idaas/cloud-cli/`（随配置文件位置变化）
- `show-signer-public-key` - 展示签名器公钥
- `qr`                  - 生成二维码
- `validate-jwt`        - 验证 JWT（仅 RS256）
- `openclaw-secret`     - 获取 OpenClaw 密钥
- `agent`               - Agent 子命令族（access-token / get-secret / put-secret / token-exchange / decrypt-secret）

### 零配置接入

一条命令完成接入：只需提供实例域名，`onboard` 会自动发现实例
（`/.well-known/cloud-idaas-configuration` → `instance_id` / `default_authorization_server` /
`developer_api_endpoint`），执行设备码登录（broker 客户端默认 `iap_cloud_idaas_cli`，
可用 `--client-id` 覆盖），列出当前用户可 Assume 的云角色，并生成
`cloud_account_token` profile（同时生成 aliyun-cli 的 `External` profile）—— 全程无需 AK。

```shell
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com
# 覆盖 broker 客户端 / 优先使用 VPC 端点：
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx --vpc
```

登录（设备码）使用组合 scope `urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential`
向实例授权服务器发起请求。`onboard` 完成后即可直接使用生成的任意 profile：

```shell
aliyun --profile aliyun-<role> sts GetCallerIdentity
aliyun --profile aliyun-<role> oss ls
```

> 前置条件：broker 客户端应用必须被授权访问 PAM 资源服务器
> （`urn:cloud:idaas:pam`）的 `cloud_account_role:obtain_access_credential` scope，
> 且已启用 `device_code` 授权类型；目标云角色必须已接入并授权给该用户。

### 获取 STS token

执行命令 `alibaba-cloud-idaas fetch-token --profile aliyun2`，输出：
```json
{
  "mode": "StsToken",
  "access_key_id": "STS.NVkY*********************",
  "access_key_secret": "CZPLzX**************************************",
  "sts_token": "CAIS0AJ1q6Ft5B2yfSjIr5XeEs3mm551gqHaMU7cjms0YeFeioDC************************",
  "expiration": "2025-05-22T02:29:05Z"
}
```

执行命令 `alibaba-cloud-idaas fetch-token --profile aws1`，输出：
```json
{
  "Version": 1,
  "AccessKeyId": "ASIAX***************",
  "SecretAccessKey": "05U0bVZ*********************************",
  "SessionToken": "IQoJb3JpZ2luX2VjEL7//////////wEaCXVzLWVhc3Qt****************************",
  "Expiration": "2025-09-02T07:20:46Z"
}
```

执行命令 `alibaba-cloud-idaas fetch-token --profile oidc1`，输出：
```json
{
  "id_token": "eyJraWQiOi*******************",
  "token_type": "Bearer",
  "access_token": "ATM4SoVDrDYt5***************************",
  "expires_in": 1200,
  "expires_at": 1756795270
}
```
加上参数 `--oidc-field id_token` 或 `--oidc-field access_token`，可只获取 ID Token 或 Access Token。

配置阿里云 cli，文件：`~/.aliyun/config.json`
```json
{
  "name": "test-idaas",
  "mode": "External",
  "region_id": "cn-hangzhou",
  "output_format": "json",
  "language": "en",
  "process_command": "alibaba-cloud-idaas fetch-token --profile aliyun2"
}
```

配置 AWS cli，文件：`~/.aws/config`

```ini
[default]
region = us-east-2
credential_process = alibaba-cloud-idaas fetch-token --profile aws2
```

### 在控制台打印 STS Token

执行命令 `alibaba-cloud-idaas show-token --profile aliyun2`，输出：
```shell
Access Key ID     : STS.NVkY*********************
Access Key Secret : CZPLzX**************************************
Security Token    : CAIS0AJ1q6Ft5B2yfSjIr5XeEs3mm551gqHaMU7cjms0YeFeioDC************************
Expiration        : 2025-05-22 09:57:11 +0800 CST   [Expires in 34 minute(s)]
```

执行命令 `alibaba-cloud-idaas show-token --profile aws1`，输出：
```shell
Access Key ID     : ASIAX***************
Secret Access Key : 05U0bVZ*********************************
Session Token     : IQoJb3JpZ2luX2VjEL7//////////wEaCXVzLWVhc3Qt****************************
Expiration        : 2025-09-02 15:20:46 +0800 CST   [Expires in 49 minute(s)]
```

### 通过 aliyun-cli 使用

#### 方式 1 —— config.json
`~/.aliyun/config.json`

```json
{
  "current": "test-sts",
  "profiles": [
    {
      "name": "test-idaas",
      "mode": "External",
      "region_id": "cn-hangzhou",
      "output_format": "json",
      "language": "en",
      "process_command": "alibaba-cloud-idaas fetch-token --profile aliyun2"
    }
  ]
}
```

```shell
aliyun --profile test-idaas oss ls
```

```shell
CreationTime                                 Region    StorageClass    BucketName
2025-02-23 22:14:37 +0800 CST        oss-cn-beijing        Standard    oss://ani********
2024-11-14 11:30:04 +0800 CST       oss-cn-hangzhou        Standard    oss://idaa*********************
2024-12-20 11:14:31 +0800 CST       oss-cn-hangzhou              IA    oss://ou***********************
Bucket Number is: 3

0.236787(s) elapsed
```

#### 方式 1.1 —— 直接执行

```shell
alibaba-cloud-idaas execute --profile aliyun2 --env-region cn-hangzhou -- aliyun sts GetCallerIdentity
```

```json
{
	"AccountId": "1391************",
	"Arn": "acs:ram::1391************:assumed-role/hatter-sts-role/idaas-assumed-role-1747877178164",
	"IdentityType": "AssumedRoleUser",
	"PrincipalId": "3007**************:idaas-assumed-role-1747877178164",
	"RequestId": "E885130B-2E04-5350-9E91-6CFACD2EC331",
	"RoleId": "3007**************"
}
```

#### 方式 1.2 —— 启动 bash 后执行

```shell
alibaba-cloud-idaas execute --profile aliyun2 --env-region cn-hangzhou bash
aliyun sts GetCallerIdentity
```

```json
{
	"AccountId": "1391************",
	"Arn": "acs:ram::1391************:assumed-role/hatter-sts-role/idaas-assumed-role-1747877344808",
	"IdentityType": "AssumedRoleUser",
	"PrincipalId": "3007**************:idaas-assumed-role-1747877344808",
	"RequestId": "7489F15F-4BBB-542E-BDB4-B5CA48FDBDCA",
	"RoleId": "3007**************"
}
```

### Terraform

`main.tf`

```
variable "region" {
  default = "cn-hangzhou"
}

provider "alicloud" {
  region = var.region
}

resource "random_uuid" "default" {
}

resource "alicloud_oss_bucket" "bucket" {
  bucket = substr("tf-example-${replace(random_uuid.default.result, "-", "")}", 0, 16)
}
```


```shell
$ alibaba-cloud-idaas execute --profile aliyun2 terraform plan

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # alicloud_oss_bucket.bucket will be created
  + resource "alicloud_oss_bucket" "bucket" {
      + acl                                      = (known after apply)
      + bucket                                   = (known after apply)
      + creation_date                            = (known after apply)
      + extranet_endpoint                        = (known after apply)
      + force_destroy                            = false
      + id                                       = (known after apply)
      + intranet_endpoint                        = (known after apply)
      + lifecycle_rule_allow_same_action_overlap = false
      + location                                 = (known after apply)
      + owner                                    = (known after apply)
      + redundancy_type                          = "LRS"
      + resource_group_id                        = (known after apply)
      + storage_class                            = "Standard"
    }

  # random_uuid.default will be created
  + resource "random_uuid" "default" {
      + id     = (known after apply)
      + result = (known after apply)
    }

Plan: 2 to add, 0 to change, 0 to destroy.

──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
```

也可以用 `alibaba-cloud-idaas execute --profile aliyun2 bash` 启动 shell，然后执行 `terraform plan`。


### OpenClaw

> 规范：https://docs.openclaw.ai/gateway/secrets

PKCS#7 配置示例：
```json
{
  "version": "1",
  "current_profile": "agent1",
  "profile": {
    "agent1": {
      "agent": {
        "instance_id": "idaas_wrws**********************",
        "developer_api_endpoint": "https://eiam-developerapi.cn-hangzhou.aliyuncs.com",
        "access_token_provider": {
          "client_credentials": {
            "token_endpoint": "https://zi******.aliyunidaas.com/api/v2/iauths_system/oauth2/token",
            "client_id": "app_ngfs**********************",
            "scope": "urn:cloud:idaas:pam|.all",
            "application_federated_credential_name": "alibabacloudp7",
            "client_assertion_pkcs7": {
              "provider": "alibaba_cloud",
              "alibaba_cloud_mode": "normal",
              "alibaba_cloud_idaas_instance_id": "idaas_wrws**********************"
            }
          }
        }
      }
    }
  }
}
```

ECS RAM Role 配置示例：
```json
{
  "version": "1",
  "current_profile": "agent2",
  "profile": {
    "agent2": {
      "instance_id": "idaas_wrws**********************",
      "developer_api_endpoint": "https://eiam-developerapi.cn-hangzhou.aliyuncs.com",
      "access_token_provider": {
        "open_api": {
          "instance_id": "idaas_wrws**********************",
          "application_id": "app_ngfs**********************",
          "audience": "urn:cloud:idaas:pam",
          "scope_values": [".all"],
          "type": "ecs_ram_role",
          "role_arn": "acs:ram::139*************:role/ecs-ram-role-test-***********"
        }
      }
    }
  }
}
```

例如 `alibaba-cloud-idaas` 位于 `/user/admin` 目录，则按如下方式配置 OpenClaw：

```json
{
  "secrets": {
    "providers": {
      "idaas": {
        "source": "exec",
        "command": "/home/admin/alibaba-cloud-idaas",
        "args": ["openclaw-secret", "-p", "agent1"],
        "passEnv": ["HOME"],
        "jsonOnly": true
      }
    }
  },
  "models": {
    "providers": {
      "llm": {
        "apiKey": { "source": "exec", "provider": "idaas", "id": "default_model" }
      }
    }
  }
}
```


测试 `openclaw-secret` 子命令：
```shell
$ echo '{ "protocolVersion": 1, "provider": "idaas", "ids": ["default_model"] }' | alibaba-cloud-idaas openclaw-secret -p agent1
{
  "protocolVersion": 1,
  "values": {
    "default_model": "sk-aac*****************************"
  }
}
```
