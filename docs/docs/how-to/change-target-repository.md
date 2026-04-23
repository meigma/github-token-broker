---
title: Change the target repository
sidebar_position: 2
description: Point an existing broker at a different repository.
---

# Change the target repository

The target repository is fixed at deploy time, not runtime. Changing it is a Terraform apply — see [Why permissions are deploy-time](../explanation/why-permissions-are-deploy-time) for the reasoning.

## Before you start

- You have write access to the Terraform configuration that deployed the broker.
- You know whether the new target is covered by the **same** GitHub App installation or a **different** one.

## Option A: same installation, different repo

If the GitHub App installation covers the new repository, update the Terraform configuration:

```hcl
module "broker" {
  source = "github.com/meigma/github-token-broker//terraform?ref=v1.1.0"

  function_name    = "github-token-broker"
  repository_owner = "your-org"
  repository_name  = "your-new-repo"   // changed

  lambda_artifact = {
    release_version = "v1.1.0"
  }
}
```

Apply:

```sh
tofu apply
```

The only change is the `GITHUB_TOKEN_BROKER_REPOSITORY_NAME` environment variable on the Lambda; Terraform updates it in place.

**Gotcha:** if the new repository is not actually covered by the installation, the broker will reject the request at the GitHub API step. The error surfaces as a broker mint failure. Verify the installation's **Repository access** list before applying.

## Option B: different installation

If the new target is under a different GitHub App installation — same App, different org — update the `installation-id` SSM parameter in addition to the Terraform module.

### 1. Update the SSM parameter

```sh
aws ssm put-parameter \
  --name /github-token-broker/app/installation-id \
  --type String \
  --value "98765432" \
  --overwrite
```

### 2. Update Terraform and apply

```hcl
module "broker" {
  # ...
  repository_owner = "different-org"
  repository_name  = "different-repo"
  # ...
}
```

```sh
tofu apply
```

The order matters if the apply and the SSM write land in different windows: invocations between the two may mint tokens for the old target. If that's a concern, point callers away from the Lambda before applying.

## Verification

Invoke the Lambda and confirm the `repositories` field in the response:

```sh
aws lambda invoke \
  --function-name github-token-broker \
  --payload 'null' \
  --cli-binary-format raw-in-base64-out \
  /tmp/out.json

jq .repositories /tmp/out.json
# ["different-org/different-repo"]
```

## See also

- [Environment variables](../reference/environment-variables) — `GITHUB_TOKEN_BROKER_REPOSITORY_OWNER` and `_NAME`.
- [SSM parameter shapes](../reference/ssm-parameter-shapes) — the `installation-id` parameter format.
- [Why permissions are deploy-time](../explanation/why-permissions-are-deploy-time) — the design principle behind this.
