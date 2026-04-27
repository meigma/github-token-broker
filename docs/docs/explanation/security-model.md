---
title: Security model
sidebar_position: 2
description: What the broker defends against, what it trusts, and what the blast radius of a minted token is.
---

# Security model

The broker's purpose is to narrow the surface over which a GitHub App's long-lived private key is usable. Instead of every workflow or service that needs a GitHub token holding a copy of the PEM, they invoke one Lambda that holds it — and that Lambda is the only thing that ever exchanges the PEM for a token.

## Threat model

### Defended against

- **Token theft from caller environments.** Callers only ever see short-lived installation tokens, not the PEM. Stealing a token buys the attacker roughly one hour on one repository with one permission set — and the window closes on its own.
- **Scope escalation via caller input.** The scope of a minted token — repository, permissions — is fixed at deploy time. A caller cannot request a wider scope. See [Why empty payloads are enforced](./why-empty-payloads).
- **Credential exfiltration via logs.** The broker never logs the minted token. Success logs contain only the repositories and expiration time, and GitHub error response bodies are not copied into failure logs. The PEM is never logged under any code path.
- **Casual access to the PEM at rest.** The PEM is stored as an SSM `SecureString`, encrypted with either the AWS-managed SSM key or a customer-managed KMS key. Reading the parameter value requires `ssm:GetParameters` with `WithDecryption`.

### Not defended against

- **Compromise of the AWS account or region.** An attacker with `ssm:GetParameters` on the parameter path and `kms:Decrypt` on its key has already won — they can read the PEM directly without touching the Lambda.
- **Compromise of the GitHub App itself.** If the App is transferred, its installations modified, or its private keys replaced by an attacker, the broker faithfully produces tokens under the attacker's control. Key rotation (see [rotate the private key](../how-to/rotate-github-app-private-key)) is the mitigation if a compromise is detected.
- **Denial of service by flooding invocations.** The broker does not rate-limit. Caller-side `lambda:InvokeFunction` permissions are the gate.

## Blast radius of a minted token

| Dimension | Limit |
|---|---|
| Scope | One repository (`owner/name`), fixed at deploy time. |
| Permissions | The set configured in `GITHUB_TOKEN_BROKER_PERMISSIONS`, upper-bounded by what the GitHub App itself is granted. |
| Lifetime | Approximately one hour. GitHub's API returns the authoritative `expires_at`. |
| Recall | None. A minted token cannot be revoked before it expires. |

A single stolen token is therefore bounded in space (one repo), in capability (the configured permissions, which default to `contents: read`), and in time (about an hour).

## Trust anchors

Two things must be trustworthy for the broker to be trustworthy:

1. **The GitHub App's private key.** The broker holds it and will sign JWTs with whatever is in the `private-key-pem` SSM parameter. If that value is replaced, the broker becomes whatever the new key authenticates as.
2. **The Lambda's execution role.** The role's IAM policy bounds what the broker can read. If the role is modified to grant broader SSM or KMS access, the broker gains capabilities it was not designed to have.

Both are under the AWS account's control. An attacker with the ability to modify either has other, likely worse, options.

## Operational invariants

- The PEM never lives on disk outside SSM. It is read into Lambda memory, used to sign a JWT, and discarded when the invocation ends.
- The PEM is never emitted to logs, traces, or response bodies.
- The token is never emitted to logs, traces, or any place other than the invocation response.
- IAM access to the three SSM parameters is tightly scoped — no wildcards, no `GetParameter` (singular), no write actions. See [IAM permissions](../reference/iam-permissions).
- Terraform and runtime configuration reject wildcard SSM paths; Terraform rejects wildcard KMS ARNs; the GitHub client rejects non-HTTPS API URLs except loopback `http` for local tests.

## See also

- [Architecture](./architecture) — the components behind these claims.
- [Why empty payloads are enforced](./why-empty-payloads) — the scope-escalation defense.
- [IAM permissions](../reference/iam-permissions) — the enforced least-privilege.
- [SECURITY.md](https://github.com/meigma/github-token-broker/blob/master/SECURITY.md) — disclosure and supported versions.
