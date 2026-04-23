---
title: SSM parameter shapes
sidebar_position: 4
description: Format and type for each of the three SSM parameters the broker reads.
---

# SSM parameter shapes

The broker reads three SSM parameters in a single `GetParameters` call with `WithDecryption=true` on every invocation.

| Default path | SSM type | Expected value |
|---|---|---|
| `/github-token-broker/app/client-id` | `String` | The GitHub App **client ID** in the `Iv23li…` format. See "Client ID, not App ID" below. |
| `/github-token-broker/app/installation-id` | `String` | The numeric installation ID as a string (e.g. `"12345678"`). Visible in the GitHub App's installation URL. |
| `/github-token-broker/app/private-key-pem` | `SecureString` | A PEM-encoded RSA private key, starting with `-----BEGIN RSA PRIVATE KEY-----`. The entire file contents, including the `BEGIN`/`END` lines and trailing newline. |

All three paths are overridable via environment variables — see [Environment variables](./environment-variables).

## Client ID, not App ID

GitHub App settings show two identifiers. The broker requires the **client ID** (the `Iv23li…`-prefixed string), not the numeric App ID. The distinction matters because the broker uses the client ID as the JWT `iss` claim; GitHub rejects JWTs signed with the numeric App ID as `iss`.

## KMS key selection

The `private-key-pem` parameter must be `SecureString`. Choose the encryption key:

- **AWS-managed SSM key** (`alias/aws/ssm`) — simplest. The Terraform module's IAM policy does not need a `kms:Decrypt` statement; SSM grants decrypt via its service principal automatically.
- **Customer-managed key** (CMK) — more audit control. Set the module's `kms_key_arn` input to the CMK ARN. The module then emits an additional `kms:Decrypt` statement scoped to that ARN.

The other two parameters (`client-id`, `installation-id`) are plain `String` and are not encrypted at rest by SSM.

## Creating the parameters

```sh
aws ssm put-parameter \
  --name /github-token-broker/app/client-id \
  --type String \
  --value "Iv23li..."

aws ssm put-parameter \
  --name /github-token-broker/app/installation-id \
  --type String \
  --value "12345678"

aws ssm put-parameter \
  --name /github-token-broker/app/private-key-pem \
  --type SecureString \
  --value "$(cat key.pem)"
```

## See also

- [Environment variables](./environment-variables) — overriding the default parameter paths.
- [Rotate the GitHub App private key](../how-to/rotate-github-app-private-key) — updating the `private-key-pem` parameter in place.
- [IAM permissions](./iam-permissions) — what the Lambda's role needs to read these.
