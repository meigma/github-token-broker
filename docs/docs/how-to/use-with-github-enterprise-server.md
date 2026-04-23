---
title: Use with GitHub Enterprise Server
sidebar_position: 3
description: Point the broker at a GitHub Enterprise Server instance instead of github.com.
---

# Use with GitHub Enterprise Server

The broker targets `https://api.github.com` by default. Overriding the API base URL points it at GitHub Enterprise Server (GHES).

## Before you start

- You have a GitHub App **on your GHES instance** (not on github.com). Its private key, client ID, and installation ID are what you load into SSM.
- The Lambda can reach the GHES API host. If your Lambda runs outside a VPC, egress is via the Lambda's default internet route. If it's attached to a VPC, make sure the VPC has a route to the GHES host (NAT gateway, VPC endpoint, or direct peering).

## Configure the module

Set the `github_api_base_url` variable on the module:

```hcl
module "broker" {
  source = "github.com/meigma/github-token-broker//terraform?ref=v1.1.0"

  function_name    = "github-token-broker"
  repository_owner = "your-org"
  repository_name  = "your-repo"

  github_api_base_url = "https://ghe.example.com/api/v3"

  lambda_artifact = {
    release_version = "v1.1.0"
  }
}
```

GHES installs its API at `/api/v3` by default; adjust if your deployment is different. Then `tofu apply`.

## Load GHES-scoped SSM parameters

The three SSM parameters are the same shape as for github.com, but the values must come from your **GHES** App:

```sh
aws ssm put-parameter --name /github-token-broker/app/client-id \
  --type String --value "Iv23li..."            # GHES App client ID

aws ssm put-parameter --name /github-token-broker/app/installation-id \
  --type String --value "12345678"             # GHES App installation ID

aws ssm put-parameter --name /github-token-broker/app/private-key-pem \
  --type SecureString --value "$(cat ghes-key.pem)"
```

## Verification

Invoke the Lambda and confirm the token works against the **GHES** API:

```sh
aws lambda invoke \
  --function-name github-token-broker \
  --payload 'null' \
  --cli-binary-format raw-in-base64-out \
  /tmp/out.json

TOKEN=$(jq -r .token /tmp/out.json)
curl -sf "https://ghe.example.com/api/v3/repos/your-org/your-repo" \
  -H "Authorization: token $TOKEN" | jq .name
```

## Caveats

- The broker's supply-chain verification flow (`gh attestation verify` against GitHub's Attestations API) targets github.com. That's a property of the release artifact, not the runtime target — it is unaffected by `github_api_base_url`.
- `AWS_REGION` is still injected by the Lambda runtime; don't set it manually.

## See also

- [Environment variables](../reference/environment-variables) — `GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL`.
- [Architecture](../explanation/architecture) — the mint flow is identical regardless of GitHub variant.
