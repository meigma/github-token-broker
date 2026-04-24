---
title: Why permissions are deploy-time
sidebar_position: 4
description: One broker equals one (repository, permissions) pair. Deploy more brokers for more scopes.
---

# Why permissions are deploy-time

Each broker is a single `(repository, permissions)` pair chosen by the deployer. Nothing at runtime changes either side. If a caller needs a token with a different scope, it invokes a different broker.

## The principle

The scope of what a minted token can do is fixed by whoever deployed the broker — not by whoever calls it. This is a deliberate corollary of the [empty-payload rule](./why-empty-payloads): if callers cannot supply input, they cannot steer the scope.

At deploy time, the Terraform module takes:

- `repository_owner` and `repository_name` — the single repo the token is scoped to.
- `permissions` — a map of GitHub permission names to levels (default `{"contents":"read"}`).

These become Lambda environment variables, which the broker reads once at startup. The broker never looks at caller-supplied fields.

## Two shapes? Two brokers.

If you need one broker that mints `contents:read` tokens and another that mints `contents:write` tokens, deploy two brokers. The Terraform module is designed for this: pick a distinct `function_name`, set the permissions, apply.

This is cheap:

- Lambda has no idle cost — a broker that is never invoked costs nothing.
- The Terraform module is small; instantiating it twice is trivial.
- The per-broker IAM role is scoped to that broker's parameters, so adding a second broker does not widen the first's blast radius.

## Why this trade-off is worth making

The alternative — accepting `permissions` as caller input — would let callers narrow *or widen* the minted token's scope. Narrowing is mostly harmless; widening is catastrophic. Writing a validator that permits narrowing while forbidding widening is a maintenance liability (GitHub permissions change; what counts as "narrower" is non-trivial; a reviewer six months from now will not remember the rules).

Fixing the scope at deploy time makes the review cost a one-time cost: the person approving the Terraform apply is the person authorizing the scope. Runtime requests need no review.

## Audit story

Because every broker's scope is in its Terraform configuration, the full audit picture is:

```sh
# List every broker in an account
aws lambda list-functions --query 'Functions[?starts_with(FunctionName, `github-token-broker`)].FunctionName'

# Each function's scope is in its environment variables
aws lambda get-function-configuration --function-name <name> \
  --query 'Environment.Variables.{owner:GITHUB_TOKEN_BROKER_REPOSITORY_OWNER,repo:GITHUB_TOKEN_BROKER_REPOSITORY_NAME,perms:GITHUB_TOKEN_BROKER_PERMISSIONS}'
```

There is no "sometimes it issues different tokens depending on who asks." Each broker has one visible answer.

## See also

- [Why empty payloads are enforced](./why-empty-payloads) — the mechanical enforcement of this principle.
- [Change the target repository](../how-to/change-target-repository) — the deploy-time flow for adjusting scope.
- [Security model](./security-model) — how this shapes the blast radius of a minted token.
