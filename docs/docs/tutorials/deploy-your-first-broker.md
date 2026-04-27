---
title: Deploy your first broker
sidebar_position: 1
description: A guided walkthrough from zero to a working Lambda that returns a short-lived GitHub token.
---

# Deploy your first broker

This tutorial takes you from a fresh AWS account and a newly-created GitHub App to a deployed broker that returns an installation token when invoked. You will use the first-party Terraform module pinned from git.

## Prerequisites

Before starting, make sure you have:

- An AWS account and credentials with permission to create IAM roles, Lambda functions, CloudWatch log groups, and SSM parameters. The default AWS CLI profile should be configured for your target region.
- A [GitHub App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app) you own. It needs:
  - A private key (downloaded as a `.pem` file).
  - At least `Contents: Read-only` repository permission.
  - An installation on the single repository the broker will issue tokens for.
  - Note the App's **client ID** (a value like `Iv23li…`, shown in the App's settings) and the **installation ID** (a numeric value visible in the installation URL).
- `tofu` ≥ 1.6 (or `terraform` ≥ 1.6) on your PATH. This tutorial uses `tofu`; the commands are identical for `terraform`.
- The `gh` CLI ≥ 2.40, authenticated. The Terraform module uses `gh` to download the Lambda release asset.
- `sha256sum` on your PATH. Installed by default on Linux; on macOS, `brew install coreutils` provides it.
- The `aws` CLI for invoking the deployed Lambda.

## What you'll build

A single Lambda function named `github-token-broker` that, when invoked, returns a JSON response containing an installation token scoped to one GitHub repository with `Contents: Read` permission.

## Step 1: Create the three SSM parameters

The broker reads three parameters at invoke time. Create them in the region you will deploy into:

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

The private-key parameter **must** be a `SecureString`. The other two are plain `String`.

## Step 2: Author a root Terraform configuration

In a new directory, create `main.tf`:

```hcl
terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0, < 7.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

module "broker" {
  source = "github.com/meigma/github-token-broker//terraform?ref=v1.1.0"

  function_name    = "github-token-broker"
  repository_owner = "your-org"
  repository_name  = "your-repo"

  lambda_artifact = {
    release_version = "v1.1.0"
  }
}

output "function_name" {
  value = module.broker.function_name
}
```

Replace `your-org`, `your-repo`, and `us-east-1` with your values. The `source` line pins the module to the `v1.1.0` tag via git — consumers of this module pin from git rather than the Terraform Registry. Always pin to a released tag; never use `ref=master` in production.

## Step 3: Apply

```sh
tofu init
tofu apply
```

`apply` will:

1. Download `github-token-broker.zip` and `checksums.txt` from the release via `gh release download`.
2. Verify the zip's SHA256 against `checksums.txt`.
3. Create the Lambda function, IAM role and inline policy, and CloudWatch log group.

When it completes, the output prints the function name.

## Step 4: Invoke and observe

Invoke the function with an empty payload and capture the response:

```sh
aws lambda invoke \
  --function-name github-token-broker \
  --payload 'null' \
  --cli-binary-format raw-in-base64-out \
  /tmp/out.json

jq . /tmp/out.json
```

The payload **must** be empty or literal `null`. Any other input — including `{}` — is rejected with `"github-token-broker does not accept invocation input"`. This is deliberate; see [Why empty payloads are enforced](../explanation/why-empty-payloads).

A healthy response looks like:

```json
{
  "token": "ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "expires_at": "2026-04-23T16:48:32Z",
  "repositories": ["your-org/your-repo"],
  "permissions": {"contents": "read"}
}
```

Use the token to confirm it works:

```sh
TOKEN=$(jq -r .token /tmp/out.json)
gh api "repos/your-org/your-repo" -H "Authorization: token $TOKEN" | jq .name
```

The response should be your repository name.

## What just happened

On each invocation the Lambda reads the three SSM parameters in one batched call (with decryption), signs an RS256 JWT valid for 9 minutes with a 60-second backdated `iat`, verifies that the configured repository belongs to the configured installation, exchanges the JWT for an installation token via `POST /app/installations/{id}/access_tokens`, and returns the token to you. It is stateless — nothing is cached across invocations. See [Architecture](../explanation/architecture) for the full diagram.

## Next steps

- [Rotate the GitHub App private key](../how-to/rotate-github-app-private-key) once you have a rotation cadence.
- [Change the target repository](../how-to/change-target-repository) if the broker needs to serve a different repo.
- Read the [security model](../explanation/security-model) to understand the token's blast radius and what the broker defends against.
- Browse the [environment variables reference](../reference/environment-variables) to see every configuration knob.
