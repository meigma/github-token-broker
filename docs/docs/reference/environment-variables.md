---
title: Environment variables
sidebar_position: 1
description: Every environment variable the broker reads at startup.
---

# Environment variables

The broker reads all runtime configuration from environment variables. When deployed via the first-party Terraform module, these are set from the module's input variables; when running the binary directly (for testing), you set them yourself.

| Variable | Required | Default | Notes |
|---|---|---|---|
| `AWS_REGION` | Yes | — | Provided by the Lambda runtime. Required for SDK initialization. |
| `GITHUB_TOKEN_BROKER_REPOSITORY_OWNER` | Yes | — | GitHub owner the minted token is scoped to. Trimmed; may contain only letters, numbers, periods, underscores, and hyphens. |
| `GITHUB_TOKEN_BROKER_REPOSITORY_NAME` | Yes | — | GitHub repository name the minted token is scoped to. Trimmed; may contain only letters, numbers, periods, underscores, and hyphens. |
| `GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM` | No | `/github-token-broker/app/client-id` | SSM parameter path for the GitHub App client ID. Must be an absolute literal path. |
| `GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM` | No | `/github-token-broker/app/installation-id` | SSM parameter path for the installation ID. Must be an absolute literal path. |
| `GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM` | No | `/github-token-broker/app/private-key-pem` | SSM SecureString parameter path for the private key PEM. Must be an absolute literal path. |
| `GITHUB_TOKEN_BROKER_PERMISSIONS` | No | `{"contents":"read"}` | JSON object of string-to-string permission entries. Must parse to a non-empty object; keys and values must be non-empty. |
| `GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL` | No | `https://api.github.com` | GitHub API base URL. Override for GitHub Enterprise Server. Must use `https` except for loopback `http` URLs used in local tests. |
| `GITHUB_TOKEN_BROKER_LOG_LEVEL` | No | `info` | One of `debug`, `info`, `warn`, `error`. Passed to `slog`. |

## Notes

- `AWS_REGION` is reserved by the Lambda runtime and injected automatically. Do not set it in Terraform; the broker's configuration loader reads it from the process environment like any other variable.
- SSM parameter paths are validated at startup. They must start with `/` and contain only letters, numbers, periods, underscores, hyphens, and slashes; wildcard characters are rejected.
- The private-key parameter **must** be `SecureString` so SSM returns it encrypted and the broker decrypts it in-flight.
- An empty or missing `GITHUB_TOKEN_BROKER_PERMISSIONS` falls back to `{"contents":"read"}`.

## See also

- [SSM parameter shapes](./ssm-parameter-shapes) — expected value format for each of the three parameters.
- [IAM permissions](./iam-permissions) — the IAM statements the Lambda's role needs to read these parameters.
