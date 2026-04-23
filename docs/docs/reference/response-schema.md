---
title: Lambda response schema
sidebar_position: 2
description: Request rules and response JSON shape.
---

# Lambda response schema

## Request

The broker accepts exactly one thing: an empty payload or the literal JSON value `null`. Anything else — including `{}` — is rejected. See [Why empty payloads are enforced](../explanation/why-empty-payloads).

When invoking with the AWS CLI, use `--payload 'null'`:

```sh
aws lambda invoke \
  --function-name github-token-broker \
  --payload 'null' \
  --cli-binary-format raw-in-base64-out \
  /tmp/out.json
```

## Success response

On success the broker returns a JSON object:

```json
{
  "token": "ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "expires_at": "2026-04-23T17:12:00Z",
  "repositories": ["your-org/your-repo"],
  "permissions": {"contents": "read"}
}
```

| Field | Type | Description |
|---|---|---|
| `token` | string | GitHub installation token in the `ghs_…` format. Use it as a Bearer token against the GitHub API. |
| `expires_at` | RFC 3339 timestamp | Token expiration, returned verbatim from the GitHub API. Do not assume a hardcoded lifetime. |
| `repositories` | array of strings | Single-element array: `["<owner>/<repo>"]`. Matches the `GITHUB_TOKEN_BROKER_REPOSITORY_OWNER` and `_NAME` configuration. |
| `permissions` | object of strings | The permission set requested at mint time. Mirrors `GITHUB_TOKEN_BROKER_PERMISSIONS`. |

## Token lifetime

GitHub returns the authoritative `expires_at`. The default lifetime is approximately one hour, but the GitHub API is the source of truth — callers should read `expires_at` rather than hardcoding a duration.

## Logging policy

The broker never logs the token. At `info` level, every successful mint emits:

```
level=info msg="minted GitHub installation token" repositories=[owner/repo] expires_at=<RFC3339>
```

That line is your signal that a mint succeeded. If it is absent, the mint failed. See [errors](./errors) for failure modes.

## Error response

On failure the Lambda returns an error (not a success response with an error body). See [errors](./errors) for the error messages the broker surfaces.

## See also

- [Errors](./errors) — message-by-message cause and resolution.
- [Environment variables](./environment-variables) — what controls `repositories` and `permissions` in the response.
- [Security model](../explanation/security-model) — what the logging invariants protect.
