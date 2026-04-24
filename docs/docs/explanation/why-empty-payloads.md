---
title: Why empty payloads are enforced
sidebar_position: 3
description: The broker rejects any caller-supplied input. This is the reasoning.
---

# Why empty payloads are enforced

The broker's handler rejects any payload that is not empty or literal `null`. An empty JSON object `{}` is rejected. A JSON string, number, or array is rejected. Anything with content is rejected.

## The rule

[`internal/handler/handler.go`](https://github.com/meigma/github-token-broker/blob/master/internal/handler/handler.go) validates the payload with:

```go
func validateEmptyPayload(payload json.RawMessage) error {
  trimmed := bytes.TrimSpace(payload)
  if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
    return nil
  }
  return fmt.Errorf("github-token-broker does not accept invocation input")
}
```

Empty bytes or `null` pass. Everything else fails.

## Why not accept `{}`

`{}` is a reasonable "empty" sentinel in many APIs. The broker refuses it anyway. Two reasons:

1. **Once structured input is accepted, structured input grows.** "`{}` means no options" eventually meets "well, while we're here, let's accept `{permissions: {...}}` from trusted callers." That meeting is the start of scope escalation. Refusing all structured input keeps that conversation from starting.
2. **The distinction has no value to callers.** The broker takes no options, so accepting `{}` and rejecting `{"foo": "bar"}` provides no capability callers need. The simplest rule — "no input" — is both the most restrictive and the most memorable.

## What this buys

- **No caller can influence the scope of the minted token.** Repository, permissions, target installation — all fixed at deploy time, none reachable through the wire protocol. This is the core property the broker exists to provide.
- **No debate about trusted input.** There is no "only our service can pass this field" pattern to review, audit, or break. Every invocation looks identical on the wire.
- **The contract is documentable in one sentence.** "Invoke with an empty payload; receive a token." Nothing more.

## What this costs

- **No "mint me a narrower token" feature.** A caller cannot ask for, say, read-only on a specific file path if the deploy-time permissions are broader. If you need two scopes, deploy two brokers.
- **AWS CLI callers must use `--payload 'null'`, not `--payload '{}'`.** The CLI's JSON schema validator treats empty-string payloads as invalid, so `null` is the canonical empty form. See [response schema](../reference/response-schema).

## Consequence for deployers

Every broker is a `(repository, permissions)` pair. If two consumers need different scopes, deploy two brokers — they are cheap (no idle cost, and the Terraform module is small). See [Why permissions are deploy-time](./why-permissions-are-deploy-time) for the corollary.

## See also

- [Errors](../reference/errors) — what a rejected payload looks like at the handler level.
- [Security model](./security-model) — this rule is one piece of the defense-in-depth posture.
- [Why permissions are deploy-time](./why-permissions-are-deploy-time) — the broader design principle.
