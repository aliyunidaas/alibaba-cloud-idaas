# alibaba-cloud-idaas

**English** | [中文](README_zh.md)

> [!IMPORTANT]
> This is a preview release.

Connect Alibaba Cloud and other cloud command line with [Alibaba Cloud IDaaS](https://www.aliyun.com/product/idaas).

## Compile

Build `alibaba-cloud-idaas` just run `go build`:
```shell
go build
```

Build with an explicit output binary name:
```shell
go build -o alibaba-cloud-idaas .
```

Feature PKCS#11 and YubiKey can be turned off via:
```shell
go build -tags disable_pkcs11,disable_yubikey_piv
```

Install to `$GOPATH/bin` (so `alibaba-cloud-idaas` is on your `PATH`):
```shell
go install
```

Cross compile for other platforms:
```shell
GOOS=darwin  GOARCH=arm64 go build -o alibaba-cloud-idaas .
GOOS=linux   GOARCH=amd64 go build -o alibaba-cloud-idaas .
GOOS=windows GOARCH=amd64 go build -o alibaba-cloud-idaas.exe .
```

## External signers

Supported external signers:

- Yubikey PIV Signer - requires `pcsc-lite` on Linux
- PKCS#11 Signer
- Custom External Signer

## Environments

| Environment Key                              | Comments                                |
|----------------------------------------------|-----------------------------------------|
| ALIBABA_CLOUD_IDAAS_USER_AGENT               | User Agent when send OIDC HTTP requests |
| ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG             | Print unsafe(secure) data to log files  |
| ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT     | Copy log to console(std err)            |
| ALIBABA_CLOUD_IDAAS_PKSC11_PIN               | PKCS#11 PIN                             |
| ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN              | YubiKey PIN                             |


## Profile Config

## Config file location

`~/.aliyun/alibaba-cloud-idaas.json` priority use when file exists
or `~/.cloud_idaas/idaas-cli.json`
> `~` means `$HOME`

### 🆕 AKless via Device Code Flow

Fetch STS Token via IDaaS new AKless feature.

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

### Device Code Flow

Follow the specification: RFC 8628: OAuth 2.0 Device Authorization Grant.
> `client_secret` is not required for public client
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

### Public Key Sign with YubiKey
> read in from env `ALIBABA_CLOUD_IDAAS_YUBIKEY_PIN` when absent
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

### Public Key sign with PKCS#11
> read pin from env `ALIBABA_CLOUD_IDAAS_PKSC11_PIN` when absent
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

### Fetch AWS STS Token

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

### Fetch OIDC Token

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
### Fetch Static Credential

Obtain a static credential (e.g. API key) managed by IDaaS via the Developer API:

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

> Note: for `device_code`, the token type now defaults to **access_token** (instance authorization server era).
> Keyless STS providers (`alibaba_cloud_sts` / `aws_sts`) automatically use `id_token` as required by `AssumeRoleWithOIDC`.


## Run Commands

Display help message `alibaba-cloud-idaas --help`.

See the [Command Reference](docs/command-reference.md) for full flag documentation.

Subcommands:
- `onboard`             - Zero-config onboard: discover instance, device-code login, list assumable cloud roles, generate profiles
- `login`               - Device-code login to IDaaS instance and cache access token
- `fetch-token`         - Fetch STS token, output STS Token to `stdout` in JSON format
- `show-token`          - Show STS token (human-readable)
- `show-profiles`       - Show profiles from `~/.aliyun/alibaba-cloud-idaas.json` or `~/.cloud_idaas/idaas-cli.json`
- `show`                - Query subcommands (profiles / roles / cache / token / status / instance / signer-key)
- `status`              - Show current profile / login status / serve daemon status
- `serve`               - Start local HTTP credential service (for SDK `credentials_uri`)
- `execute`             - Export STS token to environment and run command
- `logout`              - Clear cached tokens (profile config is preserved)
- `clean-cache`         - Clean local cache, directory `~/.aliyun/alibaba-cloud-idaas/` or `~/.cloud_idaas/cloud-cli/` (follows the config file location)
- `show-signer-public-key` - Show signer public key
- `qr`                  - Generate QR code
- `validate-jwt`        - Validate JWT (RS256 only)
- `openclaw-secret`     - Get OpenClaw secret
- `agent`               - Agent subcommands (access-token / get-secret / put-secret / token-exchange / decrypt-secret)

### Zero-config onboard

One command to onboard: given only the instance domain, `onboard` discovers the instance
(`/.well-known/cloud-idaas-configuration` → `instance_id` / `default_authorization_server` /
`developer_api_endpoint`), performs a device-code login (broker client defaults to `iap_cloud_idaas_cli`,
override with `--client-id`), lists the cloud roles the current user can assume, and generates
`cloud_account_token` profiles (plus aliyun-cli `External` profiles) — no AK required.

```shell
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com
# override broker client / prefer VPC endpoint:
alibaba-cloud-idaas onboard --instance acme.aliyunidaas.com --client-id app_xxx --vpc
```

Login (device code) uses the composite scope `urn:cloud:idaas:pam|cloud_account_role:obtain_access_credential`
against the instance authorization server. After `onboard`, use any generated profile directly:

```shell
aliyun --profile aliyun-<role> sts GetCallerIdentity
aliyun --profile aliyun-<role> oss ls
```

> Prerequisites: the broker client application must be delegated to the PAM resource server
> (`urn:cloud:idaas:pam`) scope `cloud_account_role:obtain_access_credential` and have the
> `device_code` grant enabled; target cloud roles must be onboarded and authorized to the user.

### Fetch STS token

Run command: `alibaba-cloud-idaas fetch-token --profile aliyun2`, outputs:
```json
{
  "mode": "StsToken",
  "access_key_id": "STS.NVkY*********************",
  "access_key_secret": "CZPLzX**************************************",
  "sts_token": "CAIS0AJ1q6Ft5B2yfSjIr5XeEs3mm551gqHaMU7cjms0YeFeioDC************************",
  "expiration": "2025-05-22T02:29:05Z"
}
```

Run command: `alibaba-cloud-idaas fetch-token --profile aws1`, outputs:
```json
{
  "Version": 1,
  "AccessKeyId": "ASIAX***************",
  "SecretAccessKey": "05U0bVZ*********************************",
  "SessionToken": "IQoJb3JpZ2luX2VjEL7//////////wEaCXVzLWVhc3Qt****************************",
  "Expiration": "2025-09-02T07:20:46Z"
}
```

Run command: `alibaba-cloud-idaas fetch-token --profile oidc1`, outputs:
```json
{
  "id_token": "eyJraWQiOi*******************",
  "token_type": "Bearer",
  "access_token": "ATM4SoVDrDYt5***************************",
  "expires_in": 1200,
  "expires_at": 1756795270
}
```
Add parameter `--oidc-field id_token` or `--oidc-field access_token`, only fetch ID Token or Access Token.

Config Alibaba Cloud cli, file: `~/.aliyun/config.json`
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

Config AWS cli, file: `~/.aws/config`

```ini
[default]
region = us-east-2
credential_process = alibaba-cloud-idaas fetch-token --profile aws2
```

### Print STS Token in console

Run command: `alibaba-cloud-idaas show-token --profile aliyun2`, outputs:
```shell
Access Key ID     : STS.NVkY*********************
Access Key Secret : CZPLzX**************************************
Security Token    : CAIS0AJ1q6Ft5B2yfSjIr5XeEs3mm551gqHaMU7cjms0YeFeioDC************************
Expiration        : 2025-05-22 09:57:11 +0800 CST   [Expires in 34 minute(s)]
```

Run command: `alibaba-cloud-idaas show-token --profile aws1`, outputs:
```shell
Access Key ID     : ASIAX***************
Secret Access Key : 05U0bVZ*********************************
Session Token     : IQoJb3JpZ2luX2VjEL7//////////wEaCXVzLWVhc3Qt****************************
Expiration        : 2025-09-02 15:20:46 +0800 CST   [Expires in 49 minute(s)]
```

### Via aliyun-cli

#### Method 1 - config.json
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

#### Method 1.1 - direct execute

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

#### Method 1.2 - execute bash and run

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

You can start shell with `alibaba-cloud-idaas execute --profile aliyun2 bash`, then `terraform plan`.


### OpenClaw

> Specification: https://docs.openclaw.ai/gateway/secrets

PKCS#7 config sample:
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

ECS RAM Role config sample:
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

For example `alibaba-cloud-idaas` located in `/user/admin` directory, then config OpenClaw like this:

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


Test `openclaw-secret` subcommand:
```shell
$ echo '{ "protocolVersion": 1, "provider": "idaas", "ids": ["default_model"] }' | alibaba-cloud-idaas openclaw-secret -p agent1
{
  "protocolVersion": 1,
  "values": {
    "default_model": "sk-aac*****************************"
  }
}
```