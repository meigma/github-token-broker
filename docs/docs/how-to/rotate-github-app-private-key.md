---
title: Rotate the GitHub App private key
sidebar_position: 1
description: Replace the GitHub App private key the broker signs JWTs with, without downtime.
---

# Rotate the GitHub App private key

Replace the RSA private key the broker uses to sign JWTs. The Lambda is stateless, so overwriting the SSM parameter takes effect on the next invocation — no restart, no redeploy.

## Before you start

- You have admin access to the GitHub App.
- You can write to the SSM parameter storing the private key PEM (default path `/github-token-broker/app/private-key-pem`).
- If the parameter is encrypted with a customer CMK, you have `kms:Encrypt` on that key.
- You can invoke the deployed Lambda to verify success.

GitHub Apps support **two active private keys at a time**. Use that overlap to make the rotation safe.

## Steps

### 1. Generate a new private key

From the GitHub App's settings page → **Private keys** → **Generate a private key**. Download the `.pem` file. **Do not delete the old key yet.**

### 2. Overwrite the SSM parameter

```sh
aws ssm put-parameter \
  --name /github-token-broker/app/private-key-pem \
  --type SecureString \
  --value "$(cat new-key.pem)" \
  --overwrite
```

No other change is needed. The broker reads the parameter on every invocation, so the next invoke uses the new key.

### 3. Verify

Invoke the Lambda once:

```sh
aws lambda invoke \
  --function-name github-token-broker \
  --payload 'null' \
  --cli-binary-format raw-in-base64-out \
  /tmp/out.json

jq -e .token /tmp/out.json > /dev/null && echo ok
```

A successful mint with the new key means the rotation worked. If the response contains an error referencing JWT signing or a GitHub 401, see [Rollback](#rollback).

### 4. Delete the old key

Return to the GitHub App's settings and delete the **old** private key. This is the step that actually retires the previous credential — until you do this, both keys are accepted.

## Verification

Check CloudWatch Logs for the Lambda and confirm the successful mint line:

```
level=info msg="minted GitHub installation token" repositories=[your-org/your-repo] expires_at=2026-04-23T17:12:00Z
```

The token itself is never logged; the mint line is your signal that the GitHub API accepted the new JWT signature.

## Rollback

If step 3 fails before step 4, the old key is still active on the App. Restore the previous PEM to SSM:

```sh
aws ssm put-parameter \
  --name /github-token-broker/app/private-key-pem \
  --type SecureString \
  --value "$(cat old-key.pem)" \
  --overwrite
```

Then investigate why the new key was rejected (wrong App? corrupted download? line-ending mangled?) before retrying.

## See also

- [SSM parameter shapes](../reference/ssm-parameter-shapes) — exact format of the private-key parameter.
- [Errors](../reference/errors) — what a rejected JWT looks like at the handler level.
- [Security model](../explanation/security-model) — why the PEM lives in SSM SecureString and never on disk.
