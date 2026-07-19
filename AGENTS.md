# Project Context

This repository is the public, standalone implementation of an internal service originally developed in a private monorepo. The reference implementation lives at:

```
/Users/josh/code/glab/platform/services/github-token-broker
```

It is a Go-based AWS Lambda function that vends short-lived GitHub App installation tokens from credentials stored in AWS SSM Parameter Store, intended for bootstrap workflows that need a scoped GitHub token without carrying long-lived credentials. See the [README](README.md) for the user-facing description.

When migrating code, config, or deployment artifacts from the reference implementation to this repo, consult the original directly — it is the source of truth for current behavior. This repo is **not** a fork; it is a deliberate rewrite into a public shape. Expect the following deltas from the source:

- **Configurable target repository.** The source hardcodes `GilmanLab/secrets` in config validation; this public version must accept an arbitrary repository via configuration.
- **Generic SSM parameter paths.** The source defaults to `/glab/bootstrap/github-app/*`; the public version should default to neutral paths or require them to be provided explicitly.
- **No internal references.** Strip any remaining references to `glab`, `meigma` internal tooling, or private-monorepo conventions that leak into code, docs, or config.
- **Dual-licensed.** The public version is Apache-2.0 / MIT; ensure new files carry appropriate headers only if the repo's license convention requires them.

Treat the source as authoritative for *what the service does* and this repo as authoritative for *how the public version should present it*.

---

<!-- BEGIN ai-protocol -->
# Agent Instructions

This repository's operating protocol lives in `.session.md`.

Before doing substantive work, read `.session.md` in full and follow it. It
covers startup context loading, session setup, session lifecycle, skill loading,
Worktrunk branching, session journaling, file schemas, architecture, and process
expectations.

If `.session.md` is missing, stop and tell the user the session protocol is not
installed correctly.
<!-- END ai-protocol -->
